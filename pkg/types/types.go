// Package types contains request/response types for the orderbook extension.
package types

import (
	"fmt"
	"math/big"

	"extension-scaffold/pkg/balance"
	"extension-scaffold/pkg/orderbook"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// --- Deposit (on-chain instruction) ---

type DepositRequest struct {
	Token  common.Address `json:"token"`
	Amount uint64         `json:"amount"`
}

type DepositResponse struct {
	Token     common.Address `json:"token"`
	Amount    uint64         `json:"amount"`
	Available uint64         `json:"available"`
}

// --- Withdraw (on-chain instruction) ---

type WithdrawRequest struct {
	Token   common.Address `json:"token"`
	Amount  uint64         `json:"amount"`
	Address common.Address `json:"address"`
}

type WithdrawResponse struct {
	Token        common.Address `json:"token"`
	Amount       uint64         `json:"amount"`
	To           common.Address `json:"to"`
	WithdrawalID common.Hash    `json:"withdrawalId"`
	Signature    hexutil.Bytes  `json:"signature"`
	Available    uint64         `json:"available"`
}

// --- Place Order (direct instruction) ---

type PlaceOrderRequest struct {
	Sender   string              `json:"sender"`
	Pair     string              `json:"pair"`
	Side     orderbook.Side      `json:"side"`
	Type     orderbook.OrderType `json:"type"`
	Price    uint64              `json:"price"`
	Quantity uint64              `json:"quantity"`
}

type PlaceOrderResponse struct {
	OrderID   string            `json:"orderId"`
	Status    string            `json:"status"` // "filled", "partial", "resting"
	Matches   []orderbook.Match `json:"matches,omitempty"`
	Remaining uint64            `json:"remaining"`
}

// --- Cancel Order (direct instruction) ---

type CancelOrderRequest struct {
	Sender  string `json:"sender"`
	OrderID string `json:"orderId"`
}

type CancelOrderResponse struct {
	OrderID   string `json:"orderId"`
	Pair      string `json:"pair"`
	Side      string `json:"side"`
	Remaining uint64 `json:"remaining"`
}

// --- Get My State (direct instruction) ---

type GetMyStateRequest struct {
	Sender string `json:"sender"`
}

type GetMyStateResponse struct {
	Balances   map[common.Address]balance.TokenBalance `json:"balances"`
	OpenOrders []orderbook.Order                       `json:"openOrders"`
	Matches    []orderbook.Match                       `json:"matches"`
}

// --- Get Book State (direct instruction) ---
// Public orderbook depth + (optional) recent matches scoped to a single pair.
// Pair: when set, response includes the most recent matches for that pair.
// MatchLimit: cap on returned matches; default DefaultBookMatchLimit, max ring capacity.

type GetBookStateRequest struct {
	Sender     string `json:"sender,omitempty"`
	Pair       string `json:"pair,omitempty"`
	MatchLimit int    `json:"matchLimit,omitempty"`
}

// --- Get Candles (direct instruction) ---

type GetCandlesRequest struct {
	Sender    string `json:"sender,omitempty"`
	Pair      string `json:"pair"`
	Timeframe string `json:"timeframe"`
	Limit     int    `json:"limit,omitempty"`
}

type GetCandlesResponse struct {
	Pair      string             `json:"pair"`
	Timeframe string             `json:"timeframe"`
	Candles   []orderbook.Candle `json:"candles"`
}

// --- Export History (direct instruction) ---

type ExportHistoryRequest struct {
	Sender     string `json:"sender"`
	TargetUser string `json:"targetUser,omitempty"` // admin only
}

type ExportHistoryResponse struct {
	User        string                                  `json:"user"`
	Balances    map[common.Address]balance.TokenBalance  `json:"balances"`
	Orders      []orderbook.Order                       `json:"orders"`
	Matches     []orderbook.Match                       `json:"matches"`
	Deposits    []DepositRecord                         `json:"deposits"`
	Withdrawals []WithdrawalRecord                      `json:"withdrawals"`
}

type DepositRecord struct {
	Token     common.Address `json:"token"`
	Amount    uint64         `json:"amount"`
	Timestamp int64          `json:"timestamp"`
}

type WithdrawalRecord struct {
	Token     common.Address `json:"token"`
	Amount    uint64         `json:"amount"`
	Address   common.Address `json:"address"`
	Timestamp int64          `json:"timestamp"`
}

// --- FSA (Flare Smart Accounts / Xaman) direct ops ---

// BindSessionSigRequest carries a Xaman-signed XRPL SignIn blob whose memo is a
// binding statement (domain, contract, session pubkey, nonce). The TEE verifies
// the XRPL signature in-enclave, derives the signer's r-address, resolves its
// deterministic PersonalAccount via the MasterAccountController, and binds the
// statement's session key to it. Plain JSON — the statement holds no secrets
// and the XRPL signature covers it.
type BindSessionSigRequest struct {
	Contract common.Address `json:"contract"`
	XrplBlob hexutil.Bytes  `json:"xrplBlob"`
}

type BindSessionSigResponse struct {
	User        common.Address `json:"user"`
	XrplAddress string         `json:"xrplAddress"`
	SessionPub  hexutil.Bytes  `json:"sessionPub"`
	Fingerprint string         `json:"fingerprint"`
}

// GetBindingRequest is an unauthenticated public lookup: "is target bound, and
// to which session pubkey?".
type GetBindingRequest struct {
	Target common.Address `json:"target"`
}

type GetBindingResponse struct {
	Bound       bool           `json:"bound"`
	Target      common.Address `json:"target"`
	SessionPub  hexutil.Bytes  `json:"sessionPub,omitempty"`
	Fingerprint string         `json:"fingerprint,omitempty"`
}

// WithdrawRequestPayload is the off-chain twin of the on-chain WITHDRAW
// instruction: same outcome (balance debit + TEE-signed withdrawal slip), but
// authorized by an inner signature over the canonical bytes — the user's own
// key or the session key bound to them — instead of msg.sender transport.
// Exists so gasless PersonalAccounts can request withdrawals in seconds; the
// returned slip is then relayed on-chain by anyone (executeWithdrawal is
// permissionless).
type WithdrawRequestPayload struct {
	Contract  common.Address `json:"contract"`
	User      common.Address `json:"user"`
	Token     common.Address `json:"token"`
	To        common.Address `json:"to"`
	Amount    uint64         `json:"amount"`
	Nonce     uint64         `json:"nonce"`
	Signature hexutil.Bytes  `json:"signature"`
}

// WithdrawRequestDomain is the canonical-encoding domain separator, Solidity
// bytes32("FlareOrderbookWithdrawReqV1")-style (left-aligned, zero-padded).
var WithdrawRequestDomain = mkDomain("FlareOrderbookWithdrawReqV1")

func mkDomain(s string) [32]byte {
	var d [32]byte
	copy(d[:], s)
	return d
}

// CanonicalWithdrawRequestBytes returns the byte-string signed for an off-chain
// WITHDRAW_REQUEST. Layout: abi.encode(domain, contract, user, token, to, amount, nonce).
// The frontend (lib/fsaTee.ts) must produce the identical encoding.
func CanonicalWithdrawRequestBytes(contract, user, token, to common.Address, amount, nonce uint64) ([]byte, error) {
	bytes32Ty, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, fmt.Errorf("bytes32 type: %w", err)
	}
	addrTy, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, fmt.Errorf("address type: %w", err)
	}
	uint256Ty, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, fmt.Errorf("uint256 type: %w", err)
	}
	args := abi.Arguments{
		{Type: bytes32Ty},
		{Type: addrTy}, {Type: addrTy}, {Type: addrTy}, {Type: addrTy},
		{Type: uint256Ty}, {Type: uint256Ty},
	}
	return args.Pack(
		WithdrawRequestDomain,
		contract, user, token, to,
		new(big.Int).SetUint64(amount), new(big.Int).SetUint64(nonce),
	)
}

// --- State (returned by GET_BOOK_STATE) ---

type State struct {
	Pairs      map[string]PairState `json:"pairs"`
	MatchCount int                  `json:"matchCount"`
	Matches    []orderbook.Match    `json:"matches,omitempty"`
}

type PairState struct {
	Bids []orderbook.PriceLevel `json:"bids"`
	Asks []orderbook.PriceLevel `json:"asks"`
}

// --- DO NOT MODIFY below this line. ---

// StateResponse is the envelope returned by GET /state.
type StateResponse struct {
	StateVersion common.Hash `json:"stateVersion"`
	State        State       `json:"state"`
}
