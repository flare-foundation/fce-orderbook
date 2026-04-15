package orderbook

import "errors"

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

type OrderType string

const (
	Limit  OrderType = "limit"
	Market OrderType = "market"
)

type Order struct {
	ID        string
	Owner     string
	Pair      string
	Side      Side
	Type      OrderType
	Price     uint64
	Quantity  uint64
	Remaining uint64
	Timestamp int64
}

type Match struct {
	BuyOrderID  string
	SellOrderID string
	Pair        string
	Price       uint64
	Quantity    uint64
	Timestamp   int64
}

type PriceLevel struct {
	Price      uint64
	Quantity   uint64
	OrderCount int
}

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrNotOwner        = errors.New("not the order owner")
	ErrNoLiquidity     = errors.New("no liquidity available")
	ErrInvalidPrice    = errors.New("price must be greater than zero")
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
	ErrInvalidSide     = errors.New("invalid order side")
	ErrInvalidType     = errors.New("invalid order type")
	ErrInvalidPair     = errors.New("unknown trading pair")
)
