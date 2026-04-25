package extension

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/orderbook"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

// pricePrecision is the multiplier applied to human-readable prices before they
// are stored in the TEE. This allows 3 decimal places of price precision (0.001)
// even when base and quote tokens have equal decimals, by dividing price*quantity
// back by this factor wherever a quote-token amount is computed.
// The frontend must multiply user-entered prices by this same constant.
const pricePrecision = 1000

// appendBounded appends v to s and trims the head so len(s) <= maxLen.
func appendBounded[T any](s []T, v T, maxLen int) []T {
	s = append(s, v)
	if maxLen > 0 && len(s) > maxLen {
		s = s[len(s)-maxLen:]
	}
	return s
}

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

	pairConfig, ok := e.pairs[req.Pair]
	if !ok {
		return buildResult(action, df, nil, 0, fmt.Errorf("unknown trading pair: %s", req.Pair))
	}
	ob, ok := e.orderbooks[req.Pair]
	if !ok {
		return buildResult(action, df, nil, 0, fmt.Errorf("orderbook not found for pair: %s", req.Pair))
	}

	// Per-user open-order cap. Reject before holding funds.
	e.mu.RLock()
	openCount := len(e.userOrders[user])
	e.mu.RUnlock()
	if openCount >= MaxOrdersPerUser {
		return buildResult(action, df, nil, 0, fmt.Errorf("too many open orders (max %d)", MaxOrdersPerUser))
	}

	order := &orderbook.Order{
		ID:        e.nextOrderID(),
		Owner:     user,
		Pair:      req.Pair,
		Side:      req.Side,
		Type:      req.Type,
		Price:     req.Price,
		Quantity:  req.Quantity,
		Timestamp: time.Now().UnixNano(),
	}

	holdToken, holdAmount, err := e.calculateHold(user, pairConfig, order)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("calculating hold: %w", err))
	}

	if err := e.balances.Hold(user, holdToken, holdAmount); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("insufficient balance: %w", err))
	}

	var matches []orderbook.Match
	switch req.Type {
	case orderbook.Limit:
		matches, err = ob.PlaceLimitOrder(order)
	case orderbook.Market:
		matches, err = ob.PlaceMarketOrder(order)
	default:
		_ = e.balances.Release(user, holdToken, holdAmount)
		return buildResult(action, df, nil, 0, fmt.Errorf("invalid order type: %s", req.Type))
	}

	if err != nil {
		_ = e.balances.Release(user, holdToken, holdAmount)
		return buildResult(action, df, nil, 0, fmt.Errorf("placing order: %w", err))
	}

	// Process matches: transfer funds, ring-buffer the trades, update candles, clean up filled orders.
	e.mu.Lock()
	for _, m := range matches {
		e.processMatch(m, pairConfig)
	}

	// For market orders, release any unused held amount.
	if req.Type == orderbook.Market {
		filled := totalFilled(matches, order)
		if filled < holdAmount {
			_ = e.balances.Release(user, holdToken, holdAmount-filled)
		}
	}

	// Track the order if it's resting.
	if order.Remaining > 0 {
		e.orders[order.ID] = req.Pair
		e.userOrders[user] = append(e.userOrders[user], order.ID)
	}
	e.history.orders[user] = appendBounded(e.history.orders[user], order, MaxUserHistoryOrders)
	e.mu.Unlock()

	// Enforce per-side level cap. Eviction takes ob.mu, then we refund + clean up under e.mu.
	if order.Remaining > 0 {
		if evicted := ob.EvictExcessLevels(MaxLevelsPerSide); len(evicted) > 0 {
			e.handleEvictions(req.Pair, pairConfig, evicted)
		}
	}

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

	cancelled, err := ob.CancelOrder(req.OrderID, user)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("cancelling order: %w", err))
	}

	pairConfig := e.pairs[pairName]
	releaseToken, releaseAmount := e.calculateRelease(pairConfig, cancelled)
	if releaseAmount > 0 {
		_ = e.balances.Release(user, releaseToken, releaseAmount)
	}

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

// processMatch handles a single match: transfers funds, records the trade, updates candles,
// and cleans up resting orders that were fully filled.
// Caller must hold e.mu.Lock().
func (e *Extension) processMatch(m orderbook.Match, pairConfig config.TradingPairConfig) {
	buyOwner := m.BuyOwner
	sellOwner := m.SellOwner

	quoteAmount := m.Quantity * m.Price / pricePrecision
	_ = e.balances.Transfer(buyOwner, sellOwner, pairConfig.QuoteToken, quoteAmount)
	_ = e.balances.Transfer(sellOwner, buyOwner, pairConfig.BaseToken, m.Quantity)

	// Per-pair recent-trades ring.
	if ring := e.matchesByPair[m.Pair]; ring != nil {
		ring.Push(m)
	}

	// Bounded per-user match history.
	if buyOwner != "" {
		e.history.matches[buyOwner] = appendBounded(e.history.matches[buyOwner], m, MaxUserHistoryMatches)
	}
	if sellOwner != "" && sellOwner != buyOwner {
		e.history.matches[sellOwner] = appendBounded(e.history.matches[sellOwner], m, MaxUserHistoryMatches)
	}

	// Roll the candle rings.
	e.updateCandles(m)

	// Clean up any resting order that was fully filled by this match.
	e.cleanupIfFilled(m.BuyOrderID, buyOwner, m.Pair)
	e.cleanupIfFilled(m.SellOrderID, sellOwner, m.Pair)
}

// updateCandles rolls each timeframe's ring for the pair.
// Caller must hold e.mu.Lock().
func (e *Extension) updateCandles(m orderbook.Match) {
	pair := e.candles[m.Pair]
	if pair == nil {
		return
	}
	secs := m.Seconds()
	for _, tf := range orderbook.Timeframes {
		ring := pair[tf]
		if ring == nil {
			continue
		}
		bucket := secs - (secs % int64(tf))
		last, ok := ring.Latest()
		if !ok || last.OpenTime != bucket {
			ring.Push(orderbook.Candle{
				OpenTime: bucket,
				Open:     m.Price,
				High:     m.Price,
				Low:      m.Price,
				Close:    m.Price,
				Volume:   m.Quantity,
				Trades:   1,
			})
			continue
		}
		if m.Price > last.High {
			last.High = m.Price
		}
		if m.Price < last.Low {
			last.Low = m.Price
		}
		last.Close = m.Price
		last.Volume += m.Quantity
		last.Trades++
		ring.SetLatest(last)
	}
}

// cleanupIfFilled drops orderID from active tracking if it's no longer resting on the book.
// Caller must hold e.mu.Lock().
func (e *Extension) cleanupIfFilled(orderID, owner, pair string) {
	if _, tracked := e.orders[orderID]; !tracked {
		return
	}
	ob, ok := e.orderbooks[pair]
	if !ok {
		delete(e.orders, orderID)
		if owner != "" {
			e.removeUserOrder(owner, orderID)
		}
		return
	}
	if ob.GetOrder(orderID) == nil {
		delete(e.orders, orderID)
		if owner != "" {
			e.removeUserOrder(owner, orderID)
		}
	}
}

// handleEvictions refunds held funds for evicted orders and clears them from tracking.
func (e *Extension) handleEvictions(pair string, pairConfig config.TradingPairConfig, evicted []*orderbook.Order) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ev := range evicted {
		owner := strings.ToLower(ev.Owner)
		token, amt := e.calculateRelease(pairConfig, ev)
		if amt > 0 {
			_ = e.balances.Release(owner, token, amt)
		}
		delete(e.orders, ev.ID)
		e.removeUserOrder(owner, ev.ID)
		ev.Remaining = 0 // shared pointer with history.orders entry
		logger.Infof("evicted order %s pair=%s side=%s price=%d (cap=%d)",
			ev.ID, pair, ev.Side, ev.Price, MaxLevelsPerSide)
	}
}

// removeUserOrder removes an orderID from the user's order list.
// Caller must hold e.mu.Lock().
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
		holdToken = pair.QuoteToken
		if order.Type == orderbook.Market {
			holdAmount = e.balances.AvailableBalance(user, holdToken)
			if holdAmount == 0 {
				return holdToken, 0, fmt.Errorf("no available %s balance for market buy", holdToken.Hex())
			}
		} else {
			holdAmount = order.Quantity * order.Price / pricePrecision
		}
	case orderbook.Sell:
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
		return pair.QuoteToken, order.Remaining * order.Price / pricePrecision
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
			total += m.Quantity * m.Price / pricePrecision
		} else {
			total += m.Quantity
		}
	}
	return total
}
