package balance

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrInsufficientBalance = errors.New("insufficient available balance")
	ErrInsufficientHeld    = errors.New("insufficient held balance")
	ErrZeroAmount          = errors.New("amount must be greater than zero")
)

// TokenBalance tracks a user's balance for a single token.
type TokenBalance struct {
	Available uint64 `json:"available"` // free to use for new orders or withdrawal
	Held      uint64 `json:"held"`      // locked by open orders
}

// Manager tracks per-(user, token) balances.
type Manager struct {
	mu       sync.RWMutex
	balances map[string]map[common.Address]*TokenBalance // user address -> token address -> balance
}

// NewManager creates an empty balance manager.
func NewManager() *Manager {
	return &Manager{
		balances: make(map[string]map[common.Address]*TokenBalance),
	}
}

// Deposit credits the user's available balance.
func (m *Manager) Deposit(user string, token common.Address, amount uint64) error {
	if amount == 0 {
		return ErrZeroAmount
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tb := m.getOrCreate(user, token)
	tb.Available += amount
	return nil
}

// Withdraw debits the user's available balance.
func (m *Manager) Withdraw(user string, token common.Address, amount uint64) error {
	if amount == 0 {
		return ErrZeroAmount
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tb := m.getOrCreate(user, token)
	if tb.Available < amount {
		return ErrInsufficientBalance
	}
	tb.Available -= amount
	return nil
}

// Hold moves funds from available to held (for placing an order).
func (m *Manager) Hold(user string, token common.Address, amount uint64) error {
	if amount == 0 {
		return ErrZeroAmount
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tb := m.getOrCreate(user, token)
	if tb.Available < amount {
		return ErrInsufficientBalance
	}
	tb.Available -= amount
	tb.Held += amount
	return nil
}

// Release moves funds from held back to available (for cancelling an order).
func (m *Manager) Release(user string, token common.Address, amount uint64) error {
	if amount == 0 {
		return ErrZeroAmount
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tb := m.getOrCreate(user, token)
	if tb.Held < amount {
		return ErrInsufficientHeld
	}
	tb.Held -= amount
	tb.Available += amount
	return nil
}

// Transfer debits held funds from one user and credits available funds to another.
// Used when a match executes: the seller's held base token goes to the buyer,
// and the buyer's held quote token goes to the seller.
func (m *Manager) Transfer(from, to string, token common.Address, amount uint64) error {
	if amount == 0 {
		return ErrZeroAmount
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	fromBal := m.getOrCreate(from, token)
	if fromBal.Held < amount {
		return ErrInsufficientHeld
	}
	fromBal.Held -= amount

	toBal := m.getOrCreate(to, token)
	toBal.Available += amount
	return nil
}

// Get returns a copy of the user's balance for a single token.
func (m *Manager) Get(user string, token common.Address) TokenBalance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokens, ok := m.balances[user]
	if !ok {
		return TokenBalance{}
	}
	tb, ok := tokens[token]
	if !ok {
		return TokenBalance{}
	}
	return *tb
}

// GetAll returns a copy of all token balances for a user.
func (m *Manager) GetAll(user string) map[common.Address]TokenBalance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokens, ok := m.balances[user]
	if !ok {
		return nil
	}

	result := make(map[common.Address]TokenBalance, len(tokens))
	for addr, tb := range tokens {
		result[addr] = *tb
	}
	return result
}

// AvailableBalance returns the user's available balance for a token (convenience for hold calculations).
func (m *Manager) AvailableBalance(user string, token common.Address) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokens, ok := m.balances[user]
	if !ok {
		return 0
	}
	tb, ok := tokens[token]
	if !ok {
		return 0
	}
	return tb.Available
}

// getOrCreate returns the TokenBalance for a user/token, creating it if needed.
// Caller must hold the write lock.
func (m *Manager) getOrCreate(user string, token common.Address) *TokenBalance {
	tokens, ok := m.balances[user]
	if !ok {
		tokens = make(map[common.Address]*TokenBalance)
		m.balances[user] = tokens
	}
	tb, ok := tokens[token]
	if !ok {
		tb = &TokenBalance{}
		tokens[token] = tb
	}
	return tb
}
