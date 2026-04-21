package stress

import (
	"math/rand"
	"testing"
)

func TestMarketMaker_PlacesBothSides(t *testing.T) {
	p := NewMarketMaker(MarketMakerConfig{
		Pair:     "FLR/USDT",
		MidPrice: 100_000,
		Spread:   2_000,
		QtyMin:   1,
		QtyMax:   5,
		Refresh:  0,
	})
	r := rand.New(rand.NewSource(1))

	actions := make(map[string]int)
	for i := 0; i < 20; i++ {
		act := p.NextAction(r)
		actions[act.Side]++
		if act.Pair != "FLR/USDT" {
			t.Fatalf("wrong pair: %s", act.Pair)
		}
		if act.Type != "limit" {
			t.Fatalf("market-maker must use limit orders, got %s", act.Type)
		}
		if act.Quantity < 1 || act.Quantity > 5 {
			t.Fatalf("qty out of range: %d", act.Quantity)
		}
	}
	if actions["buy"] == 0 || actions["sell"] == 0 {
		t.Fatalf("market maker must place both sides, got %v", actions)
	}
}

func TestAggressiveTaker_AlwaysMarket(t *testing.T) {
	p := NewAggressiveTaker(TakerConfig{Pair: "FLR/USDT", QtyMin: 1, QtyMax: 3})
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 10; i++ {
		act := p.NextAction(r)
		if act.Type != "market" {
			t.Fatalf("taker must use market orders, got %s", act.Type)
		}
	}
}

func TestRandomWalker_AllSidesAndTypes(t *testing.T) {
	p := NewRandomWalker(WalkerConfig{Pair: "FLR/USDT", PriceMin: 50_000, PriceMax: 150_000, QtyMin: 1, QtyMax: 10})
	r := rand.New(rand.NewSource(3))
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		a := p.NextAction(r)
		seen[a.Side+":"+a.Type] = true
		if a.Type == "limit" && (a.Price < 50_000 || a.Price > 150_000) {
			t.Fatalf("price out of range: %d", a.Price)
		}
	}
	for _, k := range []string{"buy:limit", "sell:limit", "buy:market", "sell:market"} {
		if !seen[k] {
			t.Fatalf("walker never produced %s", k)
		}
	}
}
