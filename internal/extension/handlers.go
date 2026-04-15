package extension

import (
	"encoding/json"
	"fmt"
	"strings"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/orderbook"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

// processPlaceOrder handles PLACE_ORDER direct instructions.
func (e *Extension) processPlaceOrder(action teetypes.Action, df *instruction.DataFixed, msg hexutil.Bytes) teetypes.ActionResult {
	var req types.PlaceOrderRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	user := strings.ToLower(req.Sender)
	if user == "" {
		return buildResult(action, df, nil, 0, fmt.Errorf("sender address is required"))
	}

	// Validate the pair.
	pairConfig, ok := e.pairs[req.Pair]
	if !ok {
		return buildResult(action, df, nil, 0, fmt.Errorf("unknown trading pair: %s", req.Pair))
	}
	ob, ok := e.orderbooks[req.Pair]
	if !ok {
		return buildResult(action, df, nil, 0, fmt.Errorf("orderbook not found for pair: %s", req.Pair))
	}

	// Build the order.
	order := &orderbook.Order{
		ID:       e.nextOrderID(),
		Owner:    user,
		Pair:     req.Pair,
		Side:     req.Side,
		Type:     req.Type,
		Price:    req.Price,
		Quantity: req.Quantity,
	}

	// Calculate hold amount and determine the hold token.
	holdToken, holdAmount, err := e.calculateHold(user, pairConfig, order)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("calculating hold: %w", err))
	}

	// Hold funds.
	if err := e.balances.Hold(user, holdToken, holdAmount); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("insufficient balance: %w", err))
	}

	// Place the order.
	var matches []orderbook.Match
	switch req.Type {
	case orderbook.Limit:
		matches, err = ob.PlaceLimitOrder(order)
	case orderbook.Market:
		matches, err = ob.PlaceMarketOrder(order)
	default:
		// Release hold since we won't place the order.
		_ = e.balances.Release(user, holdToken, holdAmount)
		return buildResult(action, df, nil, 0, fmt.Errorf("invalid order type: %s", req.Type))
	}

	if err != nil {
		// Release hold on failure.
		_ = e.balances.Release(user, holdToken, holdAmount)
		return buildResult(action, df, nil, 0, fmt.Errorf("placing order: %w", err))
	}

	// Process matches: transfer funds between counterparties.
	e.mu.Lock()
	for _, m := range matches {
		e.processMatch(m, pairConfig)
	}

	// For market orders, release unused held amount.
	if req.Type == orderbook.Market {
		filled := totalFilled(matches, order)
		if filled < holdAmount {
			_ = e.balances.Release(user, holdToken, holdAmount-filled)
		}
	}

	// Track the order.
	if order.Remaining > 0 {
		e.orders[order.ID] = req.Pair
		e.userOrders[user] = append(e.userOrders[user], order.ID)
	}
	e.history.orders[user] = append(e.history.orders[user], order)
	e.mu.Unlock()

	// Determine status.
	status := "resting"
	if order.Remaining == 0 {
		status = "filled"
	} else if len(matches) > 0 {
		status = "partial"
	}

	resp := types.PlaceOrderResponse{
		OrderID:   order.ID,
		Status:    status,
		Matches:   matches,
		Remaining: order.Remaining,
	}
	data, _ := json.Marshal(resp)

	logger.Infof("order placed: %s %s %s %s price=%d qty=%d matches=%d remaining=%d",
		order.ID, req.Pair, req.Side, req.Type, req.Price, req.Quantity, len(matches), order.Remaining)

	return buildResult(action, df, data, 1, nil)
}

// processCancelOrder handles CANCEL_ORDER direct instructions.
func (e *Extension) processCancelOrder(action teetypes.Action, df *instruction.DataFixed, msg hexutil.Bytes) teetypes.ActionResult {
	var req types.CancelOrderRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	user := strings.ToLower(req.Sender)
	if user == "" {
		return buildResult(action, df, nil, 0, fmt.Errorf("sender address is required"))
	}

	e.mu.Lock()
	pairName, ok := e.orders[req.OrderID]
	if !ok {
		e.mu.Unlock()
		return buildResult(action, df, nil, 0, fmt.Errorf("order not found: %s", req.OrderID))
	}

	ob, ok := e.orderbooks[pairName]
	if !ok {
		e.mu.Unlock()
		return buildResult(action, df, nil, 0, fmt.Errorf("orderbook not found for pair: %s", pairName))
	}
	e.mu.Unlock()

	// Cancel on the orderbook (checks ownership).
	cancelled, err := ob.CancelOrder(req.OrderID, user)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("cancelling order: %w", err))
	}

	// Release held funds.
	pairConfig := e.pairs[pairName]
	releaseToken, releaseAmount := e.calculateRelease(pairConfig, cancelled)
	if releaseAmount > 0 {
		_ = e.balances.Release(user, releaseToken, releaseAmount)
	}

	// Clean up tracking.
	e.mu.Lock()
	delete(e.orders, req.OrderID)
	e.removeUserOrder(user, req.OrderID)
	e.mu.Unlock()

	resp := types.CancelOrderResponse{
		OrderID:   cancelled.ID,
		Pair:      cancelled.Pair,
		Side:      string(cancelled.Side),
		Remaining: cancelled.Remaining,
	}
	data, _ := json.Marshal(resp)

	logger.Infof("order cancelled: %s pair=%s remaining=%d", cancelled.ID, pairName, cancelled.Remaining)

	return buildResult(action, df, data, 1, nil)
}

// processMatch handles a single match: transfers funds between buyer and seller.
// Caller must hold e.mu.Lock().
func (e *Extension) processMatch(m orderbook.Match, pairConfig config.TradingPairConfig) {
	buyOwner := e.getOrderOwner(m.BuyOrderID)
	sellOwner := e.getOrderOwner(m.SellOrderID)

	// The buyer's held quote tokens go to the seller.
	quoteAmount := m.Quantity * m.Price
	_ = e.balances.Transfer(buyOwner, sellOwner, pairConfig.QuoteToken, quoteAmount)

	// The seller's held base tokens go to the buyer.
	_ = e.balances.Transfer(sellOwner, buyOwner, pairConfig.BaseToken, m.Quantity)

	// Record the match globally and per user.
	e.matches = append(e.matches, m)
	if buyOwner != "" {
		e.history.matches[buyOwner] = append(e.history.matches[buyOwner], m)
	}
	if sellOwner != "" {
		e.history.matches[sellOwner] = append(e.history.matches[sellOwner], m)
	}

	// Clean up fully filled orders from tracking.
	e.cleanupFilledOrder(m.BuyOrderID)
	e.cleanupFilledOrder(m.SellOrderID)
}

// getOrderOwner looks up the owner of an order by ID from history.
func (e *Extension) getOrderOwner(orderID string) string {
	for _, orders := range e.history.orders {
		for _, o := range orders {
			if o.ID == orderID {
				return o.Owner
			}
		}
	}
	return ""
}

// cleanupFilledOrder removes a fully filled order from active tracking.
func (e *Extension) cleanupFilledOrder(orderID string) {
	pair, exists := e.orders[orderID]
	if !exists {
		return
	}
	// Check if the order still has remaining quantity by looking at history.
	for _, orders := range e.history.orders {
		for _, o := range orders {
			if o.ID == orderID && o.Remaining == 0 {
				delete(e.orders, orderID)
				e.removeUserOrder(o.Owner, orderID)
				_ = pair // suppress unused
				return
			}
		}
	}
}

// removeUserOrder removes an orderID from the user's order list.
func (e *Extension) removeUserOrder(user, orderID string) {
	ids := e.userOrders[user]
	for i, id := range ids {
		if id == orderID {
			e.userOrders[user] = append(ids[:i], ids[i+1:]...)
			return
		}
	}
}

// calculateHold determines what token and how much to hold for an order.
func (e *Extension) calculateHold(user string, pair config.TradingPairConfig, order *orderbook.Order) (holdToken common.Address, holdAmount uint64, err error) {
	switch order.Side {
	case orderbook.Buy:
		// Buy: hold quote tokens (quantity * price for limit, all available for market).
		holdToken = pair.QuoteToken
		if order.Type == orderbook.Market {
			holdAmount = e.balances.AvailableBalance(user, holdToken)
			if holdAmount == 0 {
				return holdToken, 0, fmt.Errorf("no available %s balance for market buy", holdToken.Hex())
			}
		} else {
			holdAmount = order.Quantity * order.Price
		}
	case orderbook.Sell:
		// Sell: hold base tokens (quantity for limit, all available for market).
		holdToken = pair.BaseToken
		if order.Type == orderbook.Market {
			holdAmount = e.balances.AvailableBalance(user, holdToken)
			if holdAmount == 0 {
				return holdToken, 0, fmt.Errorf("no available %s balance for market sell", holdToken.Hex())
			}
		} else {
			holdAmount = order.Quantity
		}
	default:
		return holdToken, 0, fmt.Errorf("invalid side: %s", order.Side)
	}
	return holdToken, holdAmount, nil
}

// calculateRelease determines what token and how much to release for a cancelled order.
func (e *Extension) calculateRelease(pair config.TradingPairConfig, order *orderbook.Order) (common.Address, uint64) {
	switch order.Side {
	case orderbook.Buy:
		return pair.QuoteToken, order.Remaining * order.Price
	case orderbook.Sell:
		return pair.BaseToken, order.Remaining
	default:
		return common.Address{}, 0
	}
}

// totalFilled calculates the total amount of the hold token that was actually used in matches.
func totalFilled(matches []orderbook.Match, order *orderbook.Order) uint64 {
	var total uint64
	for _, m := range matches {
		if order.Side == orderbook.Buy {
			total += m.Quantity * m.Price // quote token used
		} else {
			total += m.Quantity // base token used
		}
	}
	return total
}
