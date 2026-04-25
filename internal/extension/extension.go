package extension

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/balance"
	"extension-scaffold/pkg/orderbook"
	"extension-scaffold/pkg/types"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"

	"github.com/flare-foundation/tee-node/pkg/processorutils"
)

// History tracks per-user deposit/withdrawal/order/match records.
// All slices are kept bounded (see caps.go); the oldest entries fall off silently.
type History struct {
	deposits    map[string][]types.DepositRecord    // user -> deposits
	withdrawals map[string][]types.WithdrawalRecord // user -> withdrawals
	orders      map[string][]*orderbook.Order       // user -> all orders (including filled/cancelled)
	matches     map[string][]orderbook.Match        // user -> matches
}

func newHistory() *History {
	return &History{
		deposits:    make(map[string][]types.DepositRecord),
		withdrawals: make(map[string][]types.WithdrawalRecord),
		orders:      make(map[string][]*orderbook.Order),
		matches:     make(map[string][]orderbook.Match),
	}
}

// Extension is the orderbook extension handler.
type Extension struct {
	mu     sync.RWMutex
	Server *http.Server

	orderbooks    map[string]*orderbook.OrderBook                                        // pair name -> orderbook
	balances      *balance.Manager                                                       // per-(user, token) balances
	pairs         map[string]config.TradingPairConfig                                    // pair name -> token addresses
	matchesByPair map[string]*orderbook.Ring[orderbook.Match]                            // pair -> ring of recent matches
	candles       map[string]map[orderbook.Timeframe]*orderbook.Ring[orderbook.Candle]   // pair -> tf -> ring
	orders        map[string]string                                                      // orderID -> pair (for cancel routing)
	userOrders    map[string][]string                                                    // user address -> list of orderIDs
	history       *History                                                               // deposit/withdrawal/order history per user
	admins        map[string]bool                                                        // admin addresses
	signPort      int                                                                    // TEE sign server port
}

func New(extensionPort, signPort int) *Extension {
	e := &Extension{
		orderbooks:    make(map[string]*orderbook.OrderBook),
		balances:      balance.NewManager(),
		pairs:         make(map[string]config.TradingPairConfig),
		matchesByPair: make(map[string]*orderbook.Ring[orderbook.Match]),
		candles:       make(map[string]map[orderbook.Timeframe]*orderbook.Ring[orderbook.Candle]),
		orders:        make(map[string]string),
		userOrders:    make(map[string][]string),
		history:       newHistory(),
		admins:        make(map[string]bool),
		signPort:      signPort,
	}

	for _, addr := range config.AdminAddresses {
		e.admins[strings.ToLower(addr)] = true
	}

	for _, pair := range config.TradingPairs {
		e.pairs[pair.Name] = pair
		e.orderbooks[pair.Name] = orderbook.NewOrderBook(pair.Name)
		e.matchesByPair[pair.Name] = orderbook.NewRing[orderbook.Match](MaxMatchesPerPair)
		tfRings := make(map[orderbook.Timeframe]*orderbook.Ring[orderbook.Candle], len(orderbook.Timeframes))
		for _, tf := range orderbook.Timeframes {
			tfRings[tf] = orderbook.NewRing[orderbook.Candle](MaxCandlesPerTF)
		}
		e.candles[pair.Name] = tfRings
		logger.Infof("registered trading pair: %s (base=%s, quote=%s)", pair.Name, pair.BaseToken.Hex(), pair.QuoteToken.Hex())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /action", e.actionHandler)

	e.Server = &http.Server{Addr: fmt.Sprintf(":%d", extensionPort), Handler: mux}
	return e
}

// processAction routes by action type (instruction vs direct) and then by OPType/OPCommand.
func (e *Extension) processAction(action teetypes.Action) (int, []byte) {
	switch action.Data.Type {
	case teetypes.Instruction:
		return e.processInstruction(action)
	case teetypes.Direct:
		return e.processDirect(action)
	default:
		return http.StatusBadRequest, []byte(fmt.Sprintf("unsupported action type: %s", action.Data.Type))
	}
}

// processInstruction handles on-chain instruction actions (deposits, withdrawals).
func (e *Extension) processInstruction(action teetypes.Action) (int, []byte) {
	df, err := processorutils.Parse[instruction.DataFixed](action.Data.Message)
	if err != nil {
		return http.StatusBadRequest, []byte(fmt.Sprintf("decoding fixed data: %v", err))
	}

	if df.OPType != teeutils.ToHash(config.OPTypeOrderbook) {
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op type: received %s, expected %s (%s)",
			df.OPType.Hex(), teeutils.ToHash(config.OPTypeOrderbook).Hex(), config.OPTypeOrderbook,
		))
	}

	var ar teetypes.ActionResult

	switch {
	case df.OPCommand == teeutils.ToHash(config.OPCommandDeposit):
		ar = e.processDeposit(action, df)
	case df.OPCommand == teeutils.ToHash(config.OPCommandWithdraw):
		ar = e.processWithdraw(action, df)
	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported instruction op command: %s", df.OPCommand.Hex(),
		))
	}

	b, _ := json.Marshal(ar)
	return http.StatusOK, b
}

// processDirect handles off-chain direct instruction actions (orders, cancels, state, history).
func (e *Extension) processDirect(action teetypes.Action) (int, []byte) {
	di, err := processorutils.Parse[teetypes.DirectInstruction](action.Data.Message)
	if err != nil {
		return http.StatusBadRequest, []byte(fmt.Sprintf("decoding direct instruction: %v", err))
	}

	if di.OPType != teeutils.ToHash(config.OPTypeOrderbook) {
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported op type: received %s, expected %s (%s)",
			di.OPType.Hex(), teeutils.ToHash(config.OPTypeOrderbook).Hex(), config.OPTypeOrderbook,
		))
	}

	df := &instruction.DataFixed{
		InstructionID: action.Data.ID,
		OPType:        di.OPType,
		OPCommand:     di.OPCommand,
	}

	var ar teetypes.ActionResult

	switch {
	case di.OPCommand == teeutils.ToHash(config.OPCommandPlaceOrder):
		ar = e.processPlaceOrder(action, df, di.Message)
	case di.OPCommand == teeutils.ToHash(config.OPCommandCancelOrder):
		ar = e.processCancelOrder(action, df, di.Message)
	case di.OPCommand == teeutils.ToHash(config.OPCommandGetMyState):
		ar = e.processGetMyState(action, df, di.Message)
	case di.OPCommand == teeutils.ToHash(config.OPCommandGetBookState):
		ar = e.processGetBookState(action, df, di.Message)
	case di.OPCommand == teeutils.ToHash(config.OPCommandGetCandles):
		ar = e.processGetCandles(action, df, di.Message)
	case di.OPCommand == teeutils.ToHash(config.OPCommandExportHistory):
		ar = e.processExportHistory(action, df, di.Message)
	default:
		return http.StatusNotImplemented, []byte(fmt.Sprintf(
			"unsupported direct op command: %s", di.OPCommand.Hex(),
		))
	}

	b, _ := json.Marshal(ar)
	return http.StatusOK, b
}

// getUserOpenOrders returns all currently-resting orders for a user.
// Caller must hold e.mu (read or write).
func (e *Extension) getUserOpenOrders(user string) []orderbook.Order {
	ids := e.userOrders[user]
	if len(ids) == 0 {
		return nil
	}
	orders := make([]orderbook.Order, 0, len(ids))
	for _, id := range ids {
		pair, ok := e.orders[id]
		if !ok {
			continue
		}
		ob, ok := e.orderbooks[pair]
		if !ok {
			continue
		}
		if o := ob.GetOrder(id); o != nil {
			orders = append(orders, *o)
		}
	}
	return orders
}

// getUserMatches returns the bounded ring of matches involving a user.
// Caller must hold e.mu (read or write).
func (e *Extension) getUserMatches(user string) []orderbook.Match {
	return e.history.matches[user]
}

// nextOrderID generates a unique order ID.
var orderCounter uint64

func (e *Extension) nextOrderID() string {
	orderCounter++
	return fmt.Sprintf("ORD-%d-%d", time.Now().UnixNano(), orderCounter)
}
