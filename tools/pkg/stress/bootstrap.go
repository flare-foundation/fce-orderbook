package stress

import (
	"fmt"
	"math/big"
	"sync"

	instrutils "extension-scaffold/tools/pkg/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

// BootstrapConfig defines per-trader initial funding amounts.
type BootstrapConfig struct {
	InstructionSender common.Address
	QuoteToken        common.Address
	BaseToken         common.Address
	MintAmount        uint64 // minted to each trader (of both tokens)
	ApproveAmount     uint64 // approval for InstructionSender (must be >= MintAmount)
	DepositAmount     uint64 // initial deposit of each token
	Concurrency       int    // max concurrent trader bootstraps; 1-8 sensible for Coston2
}

// BootstrapTraders runs mint → approve → deposit for each trader in parallel
// (bounded by cfg.Concurrency). Each trader's actions are sequential from that
// trader's own nonce, so no nonce races across traders.
func BootstrapTraders(traders []*Trader, cfg BootstrapConfig) error {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, len(traders))

	for _, t := range traders {
		wg.Add(1)
		sem <- struct{}{}
		go func(t *Trader) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := bootstrapOne(t, cfg); err != nil {
				errCh <- fmt.Errorf("trader %d: %w", t.Index, err)
			}
		}(t)
	}
	wg.Wait()
	close(errCh)

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("bootstrap errors: %v", errs)
	}
	return nil
}

func bootstrapOne(t *Trader, cfg BootstrapConfig) error {
	mint := new(big.Int).SetUint64(cfg.MintAmount)
	approve := new(big.Int).SetUint64(cfg.ApproveAmount)

	// Mint quote.
	if err := instrutils.MintERC20(t.Support, cfg.QuoteToken, t.Addr, mint); err != nil {
		return fmt.Errorf("mint quote: %w", err)
	}
	// Mint base.
	if err := instrutils.MintERC20(t.Support, cfg.BaseToken, t.Addr, mint); err != nil {
		return fmt.Errorf("mint base: %w", err)
	}
	// Approve quote.
	if err := instrutils.ApproveERC20(t.Support, cfg.QuoteToken, cfg.InstructionSender, approve); err != nil {
		return fmt.Errorf("approve quote: %w", err)
	}
	// Approve base.
	if err := instrutils.ApproveERC20(t.Support, cfg.BaseToken, cfg.InstructionSender, approve); err != nil {
		return fmt.Errorf("approve base: %w", err)
	}

	// Deposit both tokens.
	if _, err := t.Deposit(cfg.InstructionSender, cfg.QuoteToken, cfg.DepositAmount); err != nil {
		return fmt.Errorf("deposit quote: %w", err)
	}
	if _, err := t.Deposit(cfg.InstructionSender, cfg.BaseToken, cfg.DepositAmount); err != nil {
		return fmt.Errorf("deposit base: %w", err)
	}

	logger.Infof("  trader %d (%s) bootstrapped: minted=%d deposited=%d each", t.Index, t.Addr.Hex(), cfg.MintAmount, cfg.DepositAmount)
	return nil
}
