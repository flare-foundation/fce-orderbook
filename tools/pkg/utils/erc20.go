package utils

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"time"

	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

// forgeArtifact is the relevant subset of a forge compilation output JSON.
type forgeArtifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

// DeployTestToken deploys a TestToken contract from the forge build output.
// artifactPath is typically "out/TestToken.sol/TestToken.json".
// Returns the deployed address.
func DeployTestToken(s *support.Support, artifactPath, name, symbol string) (common.Address, error) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return common.Address{}, errors.Errorf("reading artifact %s: %s", artifactPath, err)
	}

	var artifact forgeArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return common.Address{}, errors.Errorf("parsing artifact: %s", err)
	}

	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return common.Address{}, errors.Errorf("parsing ABI: %s", err)
	}

	bytecodeHex := strings.TrimPrefix(artifact.Bytecode.Object, "0x")
	bytecode, err := hex.DecodeString(bytecodeHex)
	if err != nil {
		return common.Address{}, errors.Errorf("decoding bytecode: %s", err)
	}

	// Encode constructor args.
	constructorArgs, err := parsed.Pack("", name, symbol)
	if err != nil {
		return common.Address{}, errors.Errorf("packing constructor args: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Address{}, errors.Errorf("creating transactor: %s", err)
	}

	// Deploy: bytecode + constructor args.
	deployData := append(bytecode, constructorArgs...)

	gasPrice, err := s.ChainClient.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Address{}, errors.Errorf("suggesting gas price: %s", err)
	}

	tx := types.NewContractCreation(
		mustNonce(s),
		big.NewInt(0),
		5_000_000, // gas limit
		gasPrice,
		deployData,
	)

	signedTx, err := opts.Signer(opts.From, tx)
	if err != nil {
		return common.Address{}, errors.Errorf("signing tx: %s", err)
	}

	if err := s.ChainClient.SendTransaction(context.Background(), signedTx); err != nil {
		return common.Address{}, errors.Errorf("sending deploy tx: %s", err)
	}

	receipt, err := support.CheckTx(signedTx, s.ChainClient)
	if err != nil {
		return common.Address{}, errors.Errorf("deploy failed: %s", err)
	}

	return receipt.ContractAddress, nil
}

// erc20ABI is a minimal ERC20 ABI for mint, approve, balanceOf.
var erc20ABI = mustParseABI(`[
	{"type":"function","name":"mint","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[]},
	{"type":"function","name":"approve","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
	{"type":"function","name":"balanceOf","inputs":[{"name":"","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}
]`)

func mustParseABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic("invalid ABI: " + err.Error())
	}
	return parsed
}

// MintERC20 calls mint(to, amount) on a TestToken contract.
func MintERC20(s *support.Support, token, to common.Address, amount *big.Int) error {
	return sendERC20Tx(s, token, "mint", to, amount)
}

// ApproveERC20 calls approve(spender, amount) on an ERC20 contract.
func ApproveERC20(s *support.Support, token, spender common.Address, amount *big.Int) error {
	return sendERC20Tx(s, token, "approve", spender, amount)
}

// BalanceOfERC20 calls balanceOf(account) on an ERC20 contract.
func BalanceOfERC20(s *support.Support, token, account common.Address) (*big.Int, error) {
	callData, err := erc20ABI.Pack("balanceOf", account)
	if err != nil {
		return nil, errors.Errorf("packing balanceOf: %s", err)
	}

	result, err := s.ChainClient.CallContract(context.Background(), toCallMsg(token, callData), nil)
	if err != nil {
		return nil, errors.Errorf("calling balanceOf: %s", err)
	}

	values, err := erc20ABI.Unpack("balanceOf", result)
	if err != nil {
		return nil, errors.Errorf("unpacking balanceOf: %s", err)
	}

	return values[0].(*big.Int), nil
}

func sendERC20Tx(s *support.Support, token common.Address, method string, args ...any) error {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("creating transactor: %s", err)
	}

	tx, err := bind.NewBoundContract(token, erc20ABI, s.ChainClient, s.ChainClient, s.ChainClient).Transact(opts, method, args...)
	if err != nil {
		return errors.Errorf("%s failed: %s", method, err)
	}

	_, err = support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return errors.Errorf("%s tx failed: %s", method, err)
	}

	return nil
}

func toCallMsg(to common.Address, data []byte) ethereum.CallMsg {
	return ethereum.CallMsg{To: &to, Data: data}
}

func mustNonce(s *support.Support) uint64 {
	from := crypto.PubkeyToAddress(s.Prv.PublicKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nonce, err := s.ChainClient.PendingNonceAt(ctx, from)
	if err != nil {
		panic("cannot get nonce: " + err.Error())
	}
	return nonce
}
