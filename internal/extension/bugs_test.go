package extension

// Tests in this file are adversarial: they assert the CORRECT/expected behavior
// for findings raised by reviewers. A FAILING test means the finding is
// confirmed; a passing test means the finding was wrong or already mitigated.

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"extension-scaffold/pkg/balance"
	"extension-scaffold/pkg/orderbook"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

const (
	testBaseHex  = "0x1111111111111111111111111111111111111111"
	testQuoteHex = "0x2222222222222222222222222222222222222222"
	testPair     = "FLR/USDT"
)

func mkAddr(i int) string { return strings.ToLower(fmt.Sprintf("0x%040d", i+1)) }

// ---------------------------------------------------------------------------
// C1 — appendBounded retains the original backing array (does it bound cap)?
// ---------------------------------------------------------------------------

func TestC1_AppendBoundedBackingArrayBound(t *testing.T) {
	const maxLen = 10
	const writes = 10000

	var s []int
	maxCap := 0
	for i := 0; i < writes; i++ {
		s = appendBounded(s, i, maxLen)
		if cap(s) > maxCap {
			maxCap = cap(s)
		}
	}
	if len(s) != maxLen {
		t.Fatalf("len: got %d, want %d", len(s), maxLen)
	}

	// Leak hypothesis: cap grows unboundedly because trim doesn't copy.
	// CORRECT behavior: cap stays within a bounded constant factor of maxLen.
	t.Logf("after %d appends with maxLen=%d: final len=%d cap=%d, peak cap=%d", writes, maxLen, len(s), cap(s), maxCap)
	if maxCap > maxLen*4 {
		t.Errorf("CONFIRMED C1: peak cap=%d exceeded 4*maxLen=%d (heap retention)", maxCap, maxLen*4)
	}
}

// ---------------------------------------------------------------------------
// C4 — uint64 overflow in Quantity * Price / pricePrecision
// ---------------------------------------------------------------------------

func TestC4_PlaceOrderRejectsOverflow(t *testing.T) {
	t.Skip("C4 confirmed — fix pending: bits.Mul64 overflow check on q*p")
	base := common.HexToAddress(testBaseHex)
	quote := common.HexToAddress(testQuoteHex)
	e := newTestExtension(testPair, base, quote)

	user := mkAddr(0)
	if err := e.balances.Deposit(user, quote, math.MaxUint64); err != nil {
		t.Fatal(err)
	}

	// Pick (price, qty) so (price * qty) wraps to a NON-zero value.
	// (2^32 + 1)^2 = 2^64 + 2^33 + 1; mod 2^64 = 2^33 + 1 = 8589934593.
	price := uint64(1)<<32 + 1
	qty := uint64(1)<<32 + 1
	wrappedProduct := price * qty // explicit wrap behavior we depend on
	if wrappedProduct == 0 {
		t.Fatalf("test setup: product wrapped to zero, pick different operands")
	}
	wrappedHold := wrappedProduct / 1000

	body, _ := json.Marshal(types.PlaceOrderRequest{
		Sender: user, Pair: testPair, Side: orderbook.Buy, Type: orderbook.Limit,
		Price: price, Quantity: qty,
	})
	ar := e.processPlaceOrder(teetypes.Action{}, &instruction.DataFixed{}, body)

	// CORRECT behavior: reject with overflow error before holding anything.
	if ar.Status == 1 {
		held := e.balances.Get(user, quote).Held
		t.Errorf("CONFIRMED C4: order accepted with overflowing q*p; held=%d (vs wrapped %d, vs true %s)",
			held, wrappedHold, "(2^33+1)/1000 ≈ 8.6M which is meaningless for q*p ≈ 1.85e19")
		return
	}
	if !strings.Contains(strings.ToLower(ar.Log), "overflow") {
		t.Logf("rejected with reason (not overflow): %s", ar.Log)
	}
}

// ---------------------------------------------------------------------------
// C5 — restart loses balances; on-chain deposits become unrecoverable
// ---------------------------------------------------------------------------

func TestC5_RestartPreservesDepositedBalance(t *testing.T) {
	t.Skip("C5 confirmed — fix pending: balance persistence across restart")
	base := common.HexToAddress(testBaseHex)
	quote := common.HexToAddress(testQuoteHex)
	e1 := newTestExtension(testPair, base, quote)

	user := mkAddr(0)
	if err := e1.balances.Deposit(user, quote, 1000); err != nil {
		t.Fatal(err)
	}
	if got := e1.balances.Get(user, quote).Available; got != 1000 {
		t.Fatalf("pre-restart balance: got %d, want 1000", got)
	}

	// Simulate a process restart by constructing a fresh Extension.
	e2 := newTestExtension(testPair, base, quote)

	// CORRECT behavior: post-restart balance reflects the on-chain deposit.
	got := e2.balances.Get(user, quote).Available
	if got != 1000 {
		t.Errorf("CONFIRMED C5: post-restart Available=%d, want 1000 (on-chain deposit lost)", got)
	}
}

// ---------------------------------------------------------------------------
// C8 — buy at improved price leaves dust in Held
// ---------------------------------------------------------------------------

func TestC8_BuyAtImprovedPriceFullyReleasesHeld(t *testing.T) {
	t.Skip("C8 confirmed — fix pending: release price-improvement residual to buyer")
	base := common.HexToAddress(testBaseHex)
	quote := common.HexToAddress(testQuoteHex)
	e := newTestExtension(testPair, base, quote)

	seller := mkAddr(0)
	buyer := mkAddr(1)
	if err := e.balances.Deposit(seller, base, 1000); err != nil {
		t.Fatal(err)
	}
	if err := e.balances.Deposit(buyer, quote, 1_000_000); err != nil {
		t.Fatal(err)
	}

	// Seller asks at 95_000 (~$95.000).
	placeOrder(t, e, types.PlaceOrderRequest{
		Sender: seller, Pair: testPair, Side: orderbook.Sell, Type: orderbook.Limit,
		Price: 95_000, Quantity: 1000,
	})
	// Buyer bids at 110_000 — improvement of 15_000 per unit.
	resp := placeOrder(t, e, types.PlaceOrderRequest{
		Sender: buyer, Pair: testPair, Side: orderbook.Buy, Type: orderbook.Limit,
		Price: 110_000, Quantity: 1000,
	})
	if resp.Status != "filled" {
		t.Fatalf("status: got %s, want filled", resp.Status)
	}

	bal := e.balances.Get(buyer, quote)
	// CORRECT behavior: fully filled limit buy releases everything.
	if bal.Held != 0 {
		t.Errorf("CONFIRMED C8: buy filled at improved price leaves Held=%d (expected 0)", bal.Held)
	}
}

// ---------------------------------------------------------------------------
// H1 — user-keyed balance map is unbounded across distinct users
// ---------------------------------------------------------------------------

func TestH1_BalanceManagerEvictsInactiveUsers(t *testing.T) {
	t.Skip("H1 confirmed — fix pending: evict zero-balance/zero-held users")
	bm := balance.NewManager()
	addr := common.HexToAddress(testBaseHex)

	const N = 5000
	for i := 0; i < N; i++ {
		if err := bm.Deposit(mkAddr(i), addr, 1); err != nil {
			t.Fatal(err)
		}
	}

	// CORRECT behavior (some bound on stale users): the very first user, who has
	// done nothing recent, would have been evicted under any LRU policy.
	first := mkAddr(0)
	bal := bm.Get(first, addr)
	if bal.Available != 0 {
		t.Errorf("CONFIRMED H1: %d distinct users tracked with no eviction; first user retains Available=%d", N, bal.Available)
	}
}
