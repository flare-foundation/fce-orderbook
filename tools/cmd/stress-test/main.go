// stress-test runs parallel mock traders against a deployed orderbook extension.
//
// See docs/superpowers/plans/2026-04-20-orderbook-stress-test.md for design.
//
// Usage:
//
//	go run ./cmd/stress-test -tier=L2 -instructionSender=0x... -duration=5m
//	go run ./cmd/stress-test -tier=L3 -duration=0                 # perpetual
package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/stress"
	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/joho/godotenv"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	pf := flag.String("p", configs.ExtensionProxyURL, "extension proxy url")
	instructionSenderF := flag.String("instructionSender", "", "InstructionSender contract address")
	tierF := flag.String("tier", "L1", "stress tier: L1..L5")
	tradersF := flag.Int("traders", 0, "override tier total (0 = use tier default)")
	durationF := flag.Duration("duration", 0, "0 = perpetual, else e.g. 10m")
	personaMixF := flag.String("persona-mix", "", "override: mm:2,taker:3,walker:5,whale:1,flicker:1")
	keysFileF := flag.String("keys", "./traders.json", "trader keys cache")
	fundPerTraderF := flag.String("fund-per-trader", "50000000000000000", "native wei per trader (default 0.05 FLR)")
	fundMinF := flag.String("fund-min", "10000000000000000", "top up if below this (default 0.01 FLR)")
	mintAmountF := flag.Uint64("mint", 1_000_000, "tokens to mint per trader")
	depositAmountF := flag.Uint64("deposit", 10_000, "tokens to deposit per trader (both sides)")
	pairF := flag.String("pair", "FLR/USDT", "trading pair")
	flag.Parse()

	if *instructionSenderF == "" {
		fmt.Fprintln(os.Stderr, "--instructionSender is required")
		os.Exit(1)
	}

	_ = godotenv.Load("../config/test-tokens.env")
	quoteTokenStr := os.Getenv("QUOTE_TOKEN")
	baseTokenStr := os.Getenv("BASE_TOKEN")
	if quoteTokenStr == "" || baseTokenStr == "" {
		fmt.Fprintln(os.Stderr, "QUOTE_TOKEN and BASE_TOKEN env vars must be set — run test-setup first")
		os.Exit(1)
	}
	quoteToken := common.HexToAddress(quoteTokenStr)
	baseToken := common.HexToAddress(baseTokenStr)
	instructionSender := common.HexToAddress(*instructionSenderF)

	// --- Resolve tier + persona mix ---
	tier, err := tierByName(*tierF)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	if *personaMixF != "" {
		m, err := ParseMixOverride(*personaMixF)
		if err != nil {
			fccutils.FatalWithCause(fmt.Errorf("parsing persona-mix: %w", err))
		}
		tier.Mix = m
	}
	if *durationF != 0 {
		tier.Duration = *durationF
	} else {
		tier.Duration = 0
	}
	totalTraders := tier.Mix.Total()
	if *tradersF > 0 {
		totalTraders = *tradersF
	}

	logger.Infof("tier=%s total_traders=%d duration=%s pair=%s", tier.Name, totalTraders, tier.Duration, *pairF)

	// --- Load funder support + keys ---
	funder, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	keys, err := stress.GenerateOrLoadTraderKeys(totalTraders, *keysFileF)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Build Trader objects ---
	traders := make([]*stress.Trader, totalTraders)
	for i := 0; i < totalTraders; i++ {
		tr, err := stress.NewTrader(i, keys[i], funder.ChainClient, funder.Addresses)
		if err != nil {
			fccutils.FatalWithCause(fmt.Errorf("trader %d: %w", i, err))
		}
		traders[i] = tr
	}

	// --- Fund traders with native gas ---
	logger.Infof("funding %d traders with native gas...", totalTraders)
	perTrader, _ := new(big.Int).SetString(*fundPerTraderF, 10)
	minNative, _ := new(big.Int).SetString(*fundMinF, 10)
	if err := stress.FundTraders(funder, traders, perTrader, minNative); err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Bootstrap (mint + approve + deposit) ---
	logger.Infof("bootstrapping %d traders (mint+approve+deposit)...", totalTraders)
	bcfg := stress.BootstrapConfig{
		InstructionSender: instructionSender,
		QuoteToken:        quoteToken,
		BaseToken:         baseToken,
		MintAmount:        *mintAmountF,
		ApproveAmount:     *mintAmountF,
		DepositAmount:     *depositAmountF,
		Concurrency:       8,
	}
	if err := stress.BootstrapTraders(traders, bcfg); err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Build assignments from tier config ---
	assignments := BuildAssignments(tier, traders, *pairF, tier.Duration)
	if len(assignments) == 0 {
		fccutils.FatalWithCause(fmt.Errorf("no assignments — check persona mix"))
	}

	// --- Run ---
	metrics := stress.NewMetrics()
	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		logger.Infof("SIGINT received — cancelling all traders")
		cancel()
	}()

	// Periodic metrics reporter.
	reportTicker := time.NewTicker(30 * time.Second)
	defer reportTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-reportTicker.C:
				printSnapshot(metrics.Snapshot())
			}
		}
	}()

	start := time.Now()
	logger.Infof("starting run (persistent traders run until SIGINT even if duration elapses)")
	stress.Run(ctx, assignments, stress.RunConfig{
		ProxyURL: *pf,
		Duration: tier.Duration,
		Metrics:  metrics,
	})
	logger.Infof("run loop exited after %s", time.Since(start))

	// --- Final sweep (always runs, even on SIGINT) ---
	logger.Infof("sweeping open orders and withdrawing balances...")
	stress.Sweep(traders, stress.SweepConfig{
		ProxyURL:          *pf,
		InstructionSender: instructionSender,
		Tokens:            []common.Address{quoteToken, baseToken},
	})

	// --- Final metrics ---
	logger.Infof("")
	logger.Infof("=== FINAL METRICS ===")
	printSnapshot(metrics.Snapshot())
}

func printSnapshot(s stress.MetricsSnapshot) {
	for action, st := range s.Actions {
		logger.Infof("  %-14s count=%d p50=%s p95=%s p99=%s errors=%v (rate=%.3f)",
			action, st.Count, st.P50, st.P95, st.P99, st.Errors, st.ErrorRate)
	}
}
