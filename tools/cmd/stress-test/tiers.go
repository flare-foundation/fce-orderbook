package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"extension-scaffold/tools/pkg/stress"
)

// PersonaMix is how many traders of each type to spawn.
type PersonaMix struct {
	MarketMakers     int // limit orders around mid
	AggressiveTakers int // market orders
	RandomWalkers    int // random limits/markets
	Whales           int // occasional big orders
	Flickers         int // place + cancel
}

func (m PersonaMix) Total() int {
	return m.MarketMakers + m.AggressiveTakers + m.RandomWalkers + m.Whales + m.Flickers
}

// Tier bundles the stock configurations. Per-persona cadence fields default
// to zero, in which case each persona uses its own fast built-in default —
// tune them only for the "day" soak profile where longer pauses matter.
type Tier struct {
	Name        string
	Mix         PersonaMix
	Duration    time.Duration
	BaseMid     uint64
	BaseSpread  uint64
	WalkerLow   uint64
	WalkerHigh  uint64
	MMRefresh   time.Duration // 0 = persona default (2s)
	TakerPause  time.Duration // 0 = persona default (500ms)
	WalkerPause time.Duration // 0 = persona default (500ms-2s random)
	WhalePause  time.Duration // 0 = persona default (30s)
}

func tierByName(name string) (Tier, error) {
	switch strings.ToUpper(name) {
	case "L1":
		return Tier{Name: "L1", Mix: PersonaMix{1, 1, 1, 0, 0}, Duration: 60 * time.Second, BaseMid: 100_000, BaseSpread: 2_000, WalkerLow: 80_000, WalkerHigh: 120_000}, nil
	case "L2":
		return Tier{Name: "L2", Mix: PersonaMix{2, 3, 4, 1, 0}, Duration: 5 * time.Minute, BaseMid: 100_000, BaseSpread: 2_000, WalkerLow: 80_000, WalkerHigh: 120_000}, nil
	case "L3":
		return Tier{Name: "L3", Mix: PersonaMix{5, 15, 25, 3, 2}, Duration: 10 * time.Minute, BaseMid: 100_000, BaseSpread: 3_000, WalkerLow: 70_000, WalkerHigh: 130_000}, nil
	case "L4":
		return Tier{Name: "L4", Mix: PersonaMix{10, 60, 100, 20, 10}, Duration: 15 * time.Minute, BaseMid: 100_000, BaseSpread: 5_000, WalkerLow: 50_000, WalkerHigh: 150_000}, nil
	case "L5":
		return Tier{Name: "L5", Mix: PersonaMix{20, 150, 250, 50, 30}, Duration: 30 * time.Second, BaseMid: 100_000, BaseSpread: 10_000, WalkerLow: 20_000, WalkerHigh: 180_000}, nil
	case "DAY":
		// Soak profile: simulate a quiet but active trading day. Low throughput
		// (~10 orders/min) and all traders Persistent so the run continues until
		// SIGTERM/SIGINT. Balance-neutral by design (MMs post both sides, takers
		// cross both sides), so traders don't drift toward zero over hours.
		return Tier{
			Name: "day", Mix: PersonaMix{2, 2, 1, 0, 0},
			Duration: 0, BaseMid: 100_000, BaseSpread: 2_000,
			WalkerLow: 90_000, WalkerHigh: 110_000,
			MMRefresh: 20 * time.Second, TakerPause: 45 * time.Second, WalkerPause: 60 * time.Second,
		}, nil
	default:
		return Tier{}, fmt.Errorf("unknown tier %q (want L1..L5, day)", name)
	}
}

// ParseMixOverride reads "mm:2,taker:3,walker:5,whale:1,flicker:1".
func ParseMixOverride(s string) (PersonaMix, error) {
	var m PersonaMix
	if s == "" {
		return m, nil
	}
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			return m, fmt.Errorf("bad pair %q", part)
		}
		n, err := strconv.Atoi(kv[1])
		if err != nil {
			return m, fmt.Errorf("bad count in %q: %w", part, err)
		}
		switch strings.ToLower(kv[0]) {
		case "mm":
			m.MarketMakers = n
		case "taker":
			m.AggressiveTakers = n
		case "walker":
			m.RandomWalkers = n
		case "whale":
			m.Whales = n
		case "flicker":
			m.Flickers = n
		default:
			return m, fmt.Errorf("unknown persona %q", kv[0])
		}
	}
	return m, nil
}

// BuildAssignments converts a tier + mix into concrete Assignment objects.
// Market makers are always Persistent (they run indefinitely per user requirement).
// If duration is 0 (perpetual), ALL traders become Persistent.
func BuildAssignments(tier Tier, traders []*stress.Trader, pair string, duration time.Duration) []stress.Assignment {
	out := make([]stress.Assignment, 0, len(traders))
	idx := 0
	add := func(n int, persona stress.Persona, role stress.TraderRole) {
		for k := 0; k < n && idx < len(traders); k++ {
			out = append(out, stress.Assignment{Trader: traders[idx], Persona: persona, Role: role})
			idx++
		}
	}
	mkRole := stress.Ephemeral
	if duration == 0 {
		mkRole = stress.Persistent
	}
	mmRefresh := tier.MMRefresh
	if mmRefresh == 0 {
		mmRefresh = 3 * time.Second
	}
	add(tier.Mix.MarketMakers, stress.NewMarketMaker(stress.MarketMakerConfig{
		Pair: pair, MidPrice: tier.BaseMid, Spread: tier.BaseSpread, QtyMin: 1, QtyMax: 5, Refresh: mmRefresh,
	}), stress.Persistent)
	add(tier.Mix.AggressiveTakers, stress.NewAggressiveTaker(stress.TakerConfig{
		Pair: pair, QtyMin: 1, QtyMax: 3, Pause: tier.TakerPause,
	}), mkRole)
	add(tier.Mix.RandomWalkers, stress.NewRandomWalker(stress.WalkerConfig{
		Pair: pair, PriceMin: tier.WalkerLow, PriceMax: tier.WalkerHigh, QtyMin: 1, QtyMax: 10, Pause: tier.WalkerPause,
	}), mkRole)
	add(tier.Mix.Whales, stress.NewWhale(stress.WhaleConfig{
		Pair: pair, QtyMin: 50, QtyMax: 200, Price: tier.BaseMid, Pause: tier.WhalePause,
	}), mkRole)
	add(tier.Mix.Flickers, stress.NewFlicker(stress.FlickerConfig{
		Pair: pair, MidPrice: tier.BaseMid, Spread: tier.BaseSpread, QtyMin: 1, QtyMax: 3,
	}), mkRole)
	return out
}
