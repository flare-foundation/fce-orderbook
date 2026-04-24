package utils

import (
	"context"
	"math/big"
	"time"

	"extension-scaffold/tools/pkg/contracts/orderbook"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

func DeployInstructionSender(s *support.Support) (common.Address, *orderbook.OrderbookInstructionSender, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to create transactor: %s", err)
	}

	deployer := crypto.PubkeyToAddress(s.Prv.PublicKey)
	admins := []common.Address{deployer}

	address, tx, contract, err := orderbook.DeployOrderbookInstructionSender(
		opts, s.ChainClient, s.Addresses.TeeExtensionRegistry, s.Addresses.TeeMachineRegistry, admins,
	)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to deploy contract: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("deployment tx not mined within 2 minutes (tx: %s): %s", tx.Hash().Hex(), err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, nil, errors.New("contract deployment failed")
	}

	return address, contract, nil
}

func SetExtensionId(s *support.Support, instructionSenderAddress common.Address) error {
	sender, err := orderbook.NewOrderbookInstructionSender(instructionSenderAddress, s.ChainClient)
	if err != nil {
		return errors.Errorf("failed to bind contract: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("failed to create transactor: %s", err)
	}

	tx, err := sender.SetExtensionId(opts)
	if err != nil {
		reason := fccutils.DecodeRevertReason(err)
		if reason == "" {
			parsed, _ := orderbook.OrderbookInstructionSenderMetaData.GetAbi()
			if parsed != nil {
				callData, packErr := parsed.Pack("setExtensionId")
				if packErr == nil {
					from := crypto.PubkeyToAddress(s.Prv.PublicKey)
					reason = fccutils.SimulateAndDecodeRevert(
						s.ChainClient, from, instructionSenderAddress, nil, callData,
					)
				}
			}
		}
		if reason != "" {
			return errors.Errorf("failed to call setExtensionId: %s (revert reason: %s)", err, reason)
		}
		return errors.Errorf("failed to call setExtensionId: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return errors.Errorf("failed waiting for transaction: %s", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		parsed, _ := orderbook.OrderbookInstructionSenderMetaData.GetAbi()
		if parsed != nil {
			callData, packErr := parsed.Pack("setExtensionId")
			if packErr == nil {
				from := crypto.PubkeyToAddress(s.Prv.PublicKey)
				reason := fccutils.SimulateAndDecodeRevert(
					s.ChainClient, from, instructionSenderAddress, nil, callData,
				)
				if reason != "" {
					return errors.Errorf("setExtensionId transaction failed (revert reason: %s)", reason)
				}
			}
		}
		return errors.New("setExtensionId transaction failed")
	}

	return nil
}

func Deposit(s *support.Support, instructionSenderAddress common.Address, token common.Address, amount *big.Int) (common.Hash, common.Hash, error) {
	sender, err := orderbook.NewOrderbookInstructionSender(instructionSenderAddress, s.ChainClient)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed to bind contract: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed to create transactor: %s", err)
	}
	opts.Value = big.NewInt(1000000) // Instruction fee in wei

	tx, err := sender.Deposit(opts, token, amount)
	if err != nil {
		reason := fccutils.DecodeRevertReason(err)
		if reason == "" {
			parsed, _ := orderbook.OrderbookInstructionSenderMetaData.GetAbi()
			if parsed != nil {
				callData, packErr := parsed.Pack("deposit", token, amount)
				if packErr == nil {
					from := crypto.PubkeyToAddress(s.Prv.PublicKey)
					reason = fccutils.SimulateAndDecodeRevert(
						s.ChainClient, from, instructionSenderAddress,
						big.NewInt(1000000), callData,
					)
				}
			}
		}
		if reason != "" {
			return common.Hash{}, common.Hash{}, errors.Errorf("failed to send deposit: %s (revert reason: %s)", err, reason)
		}
		return common.Hash{}, common.Hash{}, errors.Errorf("failed to send deposit: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed waiting for deposit tx (tx: %s): %s", tx.Hash().Hex(), err)
	}

	if receipt.Status != 1 {
		parsed, _ := orderbook.OrderbookInstructionSenderMetaData.GetAbi()
		if parsed != nil {
			callData, packErr := parsed.Pack("deposit", token, amount)
			if packErr == nil {
				from := crypto.PubkeyToAddress(s.Prv.PublicKey)
				reason := fccutils.SimulateAndDecodeRevert(
					s.ChainClient, from, instructionSenderAddress,
					big.NewInt(1000000), callData,
				)
				if reason != "" {
					return common.Hash{}, common.Hash{}, errors.Errorf("deposit tx failed (revert reason: %s)", reason)
				}
			}
		}
		return common.Hash{}, common.Hash{}, errors.Errorf("deposit tx failed with status: %d", receipt.Status)
	}

	instructionID, err := findInstructionID(s, receipt)
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}

	return instructionID, receipt.TxHash, nil
}

func Withdraw(s *support.Support, instructionSenderAddress common.Address, token common.Address, amount *big.Int, to common.Address) (common.Hash, common.Hash, error) {
	sender, err := orderbook.NewOrderbookInstructionSender(instructionSenderAddress, s.ChainClient)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed to bind contract: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed to create transactor: %s", err)
	}
	opts.Value = big.NewInt(1000000) // Instruction fee in wei

	tx, err := sender.Withdraw(opts, token, amount, to)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed to send withdraw: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
	if err != nil {
		return common.Hash{}, common.Hash{}, errors.Errorf("failed waiting for withdraw tx (tx: %s): %s", tx.Hash().Hex(), err)
	}

	if receipt.Status != 1 {
		return common.Hash{}, common.Hash{}, errors.Errorf("withdraw tx failed with status: %d", receipt.Status)
	}

	instructionID, err := findInstructionID(s, receipt)
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}

	return instructionID, receipt.TxHash, nil
}

// findInstructionID iterates through receipt logs to find the TeeInstructionsSent
// event. The event may not be the first log (e.g. ERC20 Transfer events precede it).
func findInstructionID(s *support.Support, receipt *types.Receipt) (common.Hash, error) {
	if len(receipt.Logs) == 0 {
		return common.Hash{}, errors.New("no logs found in receipt")
	}

	for _, log := range receipt.Logs {
		instructionSent, err := s.TeeExtensionRegistry.ParseTeeInstructionsSent(*log)
		if err == nil {
			return instructionSent.InstructionId, nil
		}
	}

	return common.Hash{}, errors.New("TeeInstructionsSent event not found in receipt logs")
}
