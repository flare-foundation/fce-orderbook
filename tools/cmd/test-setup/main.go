// test-setup sets up the test environment for the orderbook extension.
//
// It performs these steps:
//  1. Allow deployer to deposit (idempotent)
//  2. Deploy two TestToken contracts (quote + base)
//  3. Update config/pairs.json with deployed token addresses
//  4. Mint tokens to the deployer
//  5. Approve InstructionSender to spend both tokens
//  6. Write config/test-tokens.env for use by other test commands
//
// Note: setExtensionId is handled by post-build.sh, not here.
//
// Usage:
//
//	go run ./cmd/test-setup -a <addresses-file> -c <chain-url> -instructionSender <addr>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/contracts/orderbook"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	instrutils "extension-scaffold/tools/pkg/utils"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

// txSpacing is a small pause between consecutive transactions, giving
// load-balanced public RPC nodes time to propagate nonce state between peers.
const txSpacing = 500 * time.Millisecond

const (
	testTokenArtifact = "../out/TestToken.sol/TestToken.json"
	pairsConfigPath   = "../config/pairs.json"
	testTokensEnvPath = "../config/test-tokens.env"
	mintAmount        = 1_000_000
	approveAmount     = 1_000_000
)

type pairConfig struct {
	Name       string `json:"name"`
	BaseToken  string `json:"baseToken"`
	QuoteToken string `json:"quoteToken"`
}

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	instructionSenderF := flag.String("instructionSender", "", "InstructionSender contract address")
	flag.Parse()

	if *instructionSenderF == "" {
		logger.Errorf("--instructionSender is required")
		os.Exit(1)
	}

	instructionSenderAddr := common.HexToAddress(*instructionSenderF)

	s, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	deployer := crypto.PubkeyToAddress(s.Prv.PublicKey)
	logger.Infof("Deployer: %s", deployer.Hex())
	logger.Infof("InstructionSender: %s", instructionSenderAddr.Hex())

	// --- Step 1: Allow deployer to deposit (idempotent) ---
	logger.Infof("Step 1: Allowing deployer to deposit...")
	err = allowUser(s, instructionSenderAddr, deployer)
	if err != nil {
		logger.Infof("  allowUser: %s (may already be allowed)", err)
	} else {
		logger.Infof("  Deployer allowed")
	}

	// --- Step 2: Deploy test tokens ---
	logger.Infof("Step 2: Deploying test tokens...")

	time.Sleep(txSpacing)
	quoteToken, err := instrutils.DeployTestToken(s, testTokenArtifact, "TestUSDT", "TUSDT")
	if err != nil {
		fccutils.FatalWithCause(fmt.Errorf("deploying quote token: %w", err))
	}
	logger.Infof("  Quote token (TUSDT) deployed at: %s", quoteToken.Hex())

	time.Sleep(txSpacing)
	baseToken, err := instrutils.DeployTestToken(s, testTokenArtifact, "TestFLR", "TFLR")
	if err != nil {
		fccutils.FatalWithCause(fmt.Errorf("deploying base token: %w", err))
	}
	logger.Infof("  Base token (TFLR) deployed at: %s", baseToken.Hex())

	// --- Step 3: Update pairs.json ---
	logger.Infof("Step 3: Updating config/pairs.json...")
	pairs := []pairConfig{
		{Name: "FLR/USDT", BaseToken: baseToken.Hex(), QuoteToken: quoteToken.Hex()},
	}
	pairsJSON, _ := json.MarshalIndent(pairs, "", "    ")
	if err := os.WriteFile(pairsConfigPath, pairsJSON, 0644); err != nil {
		fccutils.FatalWithCause(fmt.Errorf("writing pairs.json: %w", err))
	}
	logger.Infof("  Updated with FLR/USDT pair")

	// --- Step 4: Mint tokens ---
	logger.Infof("Step 4: Minting tokens to deployer...")
	amount := big.NewInt(mintAmount)

	time.Sleep(txSpacing)
	if err := instrutils.MintERC20(s, quoteToken, deployer, amount); err != nil {
		fccutils.FatalWithCause(fmt.Errorf("minting quote token: %w", err))
	}
	logger.Infof("  Minted %d TUSDT to %s", mintAmount, deployer.Hex())

	time.Sleep(txSpacing)
	if err := instrutils.MintERC20(s, baseToken, deployer, amount); err != nil {
		fccutils.FatalWithCause(fmt.Errorf("minting base token: %w", err))
	}
	logger.Infof("  Minted %d TFLR to %s", mintAmount, deployer.Hex())

	// --- Step 5: Approve InstructionSender ---
	logger.Infof("Step 5: Approving InstructionSender to spend tokens...")
	approveAmt := big.NewInt(approveAmount)

	time.Sleep(txSpacing)
	if err := instrutils.ApproveERC20(s, quoteToken, instructionSenderAddr, approveAmt); err != nil {
		fccutils.FatalWithCause(fmt.Errorf("approving quote token: %w", err))
	}
	logger.Infof("  Approved %d TUSDT", approveAmount)

	time.Sleep(txSpacing)
	if err := instrutils.ApproveERC20(s, baseToken, instructionSenderAddr, approveAmt); err != nil {
		fccutils.FatalWithCause(fmt.Errorf("approving base token: %w", err))
	}
	logger.Infof("  Approved %d TFLR", approveAmount)

	// --- Step 6: Write test-tokens.env ---
	logger.Infof("Step 6: Writing config/test-tokens.env...")
	os.MkdirAll(filepath.Dir(testTokensEnvPath), 0755)
	envContent := fmt.Sprintf("# Auto-generated by test-setup — do not edit manually\nQUOTE_TOKEN=%s\nBASE_TOKEN=%s\n", quoteToken.Hex(), baseToken.Hex())
	if err := os.WriteFile(testTokensEnvPath, []byte(envContent), 0644); err != nil {
		fccutils.FatalWithCause(fmt.Errorf("writing test-tokens.env: %w", err))
	}

	logger.Infof("")
	logger.Infof("Setup complete!")
	logger.Infof("  QUOTE_TOKEN=%s", quoteToken.Hex())
	logger.Infof("  BASE_TOKEN=%s", baseToken.Hex())
	logger.Infof("")
	logger.Infof("Next: restart the extension (to pick up new pairs.json), then run test-deposit")
}

func allowUser(s *support.Support, instructionSenderAddr, user common.Address) error {
	sender, err := orderbook.NewOrderbookInstructionSender(instructionSenderAddr, s.ChainClient)
	if err != nil {
		return fmt.Errorf("binding contract: %w", err)
	}

	var tx *types.Transaction
	var sendErr error
	for attempt := 0; attempt < 3; attempt++ {
		opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
		if err != nil {
			return fmt.Errorf("creating transactor: %w", err)
		}
		if attempt > 0 {
			gp, gerr := s.ChainClient.SuggestGasPrice(context.Background())
			if gerr != nil {
				return fmt.Errorf("suggesting gas price: %w", gerr)
			}
			mul := new(big.Int).Mul(gp, big.NewInt(int64(100+20*attempt)))
			opts.GasPrice = new(big.Int).Div(mul, big.NewInt(100))
		}

		tx, sendErr = sender.AllowUser(opts, user)
		if sendErr == nil {
			break
		}
		if !instrutils.IsRetryableTxError(sendErr) {
			return fmt.Errorf("calling allowUser: %w", sendErr)
		}
		if attempt < 2 {
			time.Sleep(2 * time.Second)
		}
	}
	if sendErr != nil {
		return fmt.Errorf("calling allowUser after retries: %w", sendErr)
	}

	_, err = support.CheckTx(tx, s.ChainClient)
	return err
}
