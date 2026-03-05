package fccutils

import (
	"context"
	"math/big"
	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teeextensionregistry"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teeownerallowlist"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/tee-node/pkg/wallets"
	"github.com/pkg/errors"
)

var DefaultExtensionId = big.NewInt(0)

func SetupExtension(
	s *support.Support,
	governanceHash common.Hash,
	instructionsSenderAddress, stateVerifierAddress common.Address,
) (*big.Int, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return nil, err
	}
	// Register the extension
	extRegistered, _, err := registerExtension(s, opts, instructionsSenderAddress, stateVerifierAddress)
	if err != nil {
		return nil, err
	}
	extensionID := extRegistered.ExtensionId

	logger.Infof("Extension registered with ID: %s", extensionID.String())

	// Allow TEE machine owners and wallet project managers for this extension
	ownersAdded, err := allowTeeMachineOwners(s, opts, extensionID, []common.Address{crypto.PubkeyToAddress(s.Prv.PublicKey)})
	if err != nil {
		return nil, err
	}
	_, err = allowTeeProjectManagerOwners(s, opts, extensionID, []common.Address{crypto.PubkeyToAddress(s.Prv.PublicKey)})
	if err != nil {
		return nil, err
	}

	logger.Infof("TEE machine owners and wallet project managers allowed: %s", ownersAdded.Owners)

	// Allow an EVM type of keys on the extension
	isKeyTypeSupported, err := IsKeyTypeSupported(s, extensionID, wallets.EVMType)
	if err != nil {
		return nil, err
	}
	if isKeyTypeSupported {
		return nil, errors.New("key already supported")
	}

	logger.Infof("Adding key type %s to extension %s", wallets.EVMType, extensionID)
	err = AddSupportedKeyTypes(s, extensionID, []common.Hash{wallets.EVMType})
	if err != nil {
		return nil, err
	}

	return extensionID, nil
}

func AddSupportedKeyTypes(s *support.Support, extensionId *big.Int, keyTypes []common.Hash) error {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	keyTypesBytes32 := HashArrayToBytes32Array(keyTypes)

	err = addSupportedKeyTypesTx(s, opts, extensionId, keyTypesBytes32)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	return nil
}

func IsKeyTypeSupported(s *support.Support, extensionId *big.Int, keyType common.Hash) (bool, error) {
	callOpts := &bind.CallOpts{
		From:    crypto.PubkeyToAddress(s.Prv.PublicKey),
		Context: context.Background(),
	}

	isSupported, err := isKeyTypeSupportedCall(s, callOpts, extensionId, keyType)
	if err != nil {
		return false, errors.Errorf("%s", err)
	}

	return isSupported, nil
}

func registerExtension(
	s *support.Support, opts *bind.TransactOpts, instructionsSenderAddress, stateVerifierAddress common.Address,
) (
	*teeextensionregistry.TeeExtensionRegistryTeeExtensionRegistered, *teeextensionregistry.TeeExtensionRegistryTeeExtensionContractsSet, error,
) {
	tx, err := s.TeeExtensionRegistry.Register(opts, stateVerifierAddress, instructionsSenderAddress)
	if err != nil {
		return nil, nil, errors.Errorf("TeeExtensionRegistry.Register failed: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return nil, nil, errors.Errorf("%s", err)
	}

	extensionRegistered, err := s.TeeExtensionRegistry.ParseTeeExtensionRegistered(*receipt.Logs[0])
	if err != nil || receipt.Status != 1 {
		return nil, nil, errors.Errorf("error %s, or receipt status not 1", err)
	}

	extensionContractsSet, err := s.TeeExtensionRegistry.ParseTeeExtensionContractsSet(*receipt.Logs[1])
	if err != nil || receipt.Status != 1 {
		return nil, nil, errors.Errorf("error %s, or receipt status not 1", err)
	}

	return extensionRegistered, extensionContractsSet, nil
}

func allowTeeMachineOwners(s *support.Support, opts *bind.TransactOpts, extensionId *big.Int, owners []common.Address) (*teeownerallowlist.TeeOwnerAllowlistAllowedTeeMachineOwnersAdded, error) {
	tx, err := s.TeeOwnerAllowlist.AddAllowedTeeMachineOwners(opts, extensionId, owners)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	ownersAdded, err := s.TeeOwnerAllowlist.ParseAllowedTeeMachineOwnersAdded(*receipt.Logs[0])
	if err != nil || receipt.Status != 1 {
		return nil, errors.Errorf("error %s, or receipt status not 1", err)
	}

	return ownersAdded, nil
}

func allowTeeProjectManagerOwners(s *support.Support, opts *bind.TransactOpts, extensionId *big.Int, owners []common.Address) (*teeownerallowlist.TeeOwnerAllowlistAllowedTeeWalletProjectOwnersAdded, error) {
	tx, err := s.TeeOwnerAllowlist.AddAllowedTeeWalletProjectOwners(opts, extensionId, owners)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	ownersAdded, err := s.TeeOwnerAllowlist.ParseAllowedTeeWalletProjectOwnersAdded(*receipt.Logs[0])
	if err != nil || receipt.Status != 1 {
		return nil, errors.Errorf("error %s, or receipt status not 1", err)
	}

	return ownersAdded, nil
}

func addSupportedKeyTypesTx(s *support.Support, opts *bind.TransactOpts, extensionId *big.Int, keyTypesBytes32 [][32]byte) error {
	tx, err := s.TeeExtensionRegistry.AddSupportedKeyTypes(opts, extensionId, keyTypesBytes32)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	_, err = support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return errors.Errorf("%s", err)
	}
	return nil
}

func isKeyTypeSupportedCall(s *support.Support, opts *bind.CallOpts, extensionId *big.Int, keyType common.Hash) (bool, error) {
	return s.TeeExtensionRegistry.IsKeyTypeSupported(opts, extensionId, keyType)
}
