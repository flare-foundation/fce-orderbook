package support

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"extension-scaffold/tools/pkg/configs"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/fdc2hub"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/system"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teeextensionregistry"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teemachineregistry"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teeownerallowlist"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teeverification"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teewalletkeymanager"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teewalletmanager"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teewalletprojectmanager"
	"github.com/joho/godotenv"

	"github.com/pkg/errors"
)

type Support struct {
	Prv *ecdsa.PrivateKey // funded private key

	FlareSystemManager      *system.FlareSystemsManager
	TeeMachineRegistry      *teemachineregistry.TeeMachineRegistry
	TeeWalletProjectManager *teewalletprojectmanager.TeeWalletProjectManager
	TeeWalletManager        *teewalletmanager.TeeWalletManager
	TeeWalletKeyManager     *teewalletkeymanager.TeeWalletKeyManager
	Fdc2Hub                 *fdc2hub.Fdc2Hub
	TeeVerification         *teeverification.TeeVerification
	TeeExtensionRegistry    *teeextensionregistry.TeeExtensionRegistry
	TeeOwnerAllowlist       *teeownerallowlist.TeeOwnerAllowlist

	Addresses *Addresses

	ChainClient *ethclient.Client
	ChainID     *big.Int
}

type Addresses struct {
	TeeMachineRegistry      common.Address `json:"TeeMachineRegistry"`
	TeeWalletManager        common.Address `json:"TeeWalletManager"`
	TeeWalletKeyManager     common.Address `json:"TeeWalletKeyManager"`
	TeeWalletProjectManager common.Address `json:"TeeWalletProjectManager"`
	FlareSystemManager      common.Address `json:"FlareSystemsManager"`
	Fdc2Hub                 common.Address `json:"Fdc2Hub"`
	TeeVerification         common.Address `json:"TeeVerification"`
	TeeExtensionRegistry    common.Address `json:"TeeExtensionRegistry"`
	TeeOwnerAllowlist       common.Address `json:"TeeOwnerAllowlist"`
}

func DefaultSupport(AddressesFilePath, chainNodeURL string) (*Support, error) {
	addr := &Addresses{}
	err := configs.ReadAddresses(AddressesFilePath, addr)
	if err != nil {
		addr, err = ParseAddresses(AddressesFilePath)
		if err != nil {
			return nil, errors.Errorf("%s", err)
		}
	}

	cc, err := ethclient.Dial(chainNodeURL)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	privKey, err := DefaultPrivateKey()
	if err != nil {
		return nil, err
	}

	return NewSupport(privKey, cc, addr)
}

func DefaultPrivateKey() (*ecdsa.PrivateKey, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
	}
	privKeyString := os.Getenv("PRIV_KEY")
	fmt.Printf("privKeyString: %s\n", privKeyString)

	if privKeyString == "" {
		fmt.Println("Warning: PRIV_KEY not set, falling back to hardcoded dev key (only works on local devnet)")
		return configs.PrvWithFunds, nil
	} else {
		if strings.HasPrefix(privKeyString, "0x") || strings.HasPrefix(privKeyString, "0X") {
			privKeyString = privKeyString[2:]
		}

		privKey, err := crypto.HexToECDSA(privKeyString)
		if err != nil {
			return nil, errors.Errorf("%s", err)
		}
		return privKey, nil
	}
}

func NewSupport(prv *ecdsa.PrivateKey, chainClient *ethclient.Client, addresses *Addresses) (*Support, error) {
	tr, err := teemachineregistry.NewTeeMachineRegistry(addresses.TeeMachineRegistry, chainClient)
	if err != nil {
		return nil, err
	}

	twm, err := teewalletmanager.NewTeeWalletManager(addresses.TeeWalletManager, chainClient)
	if err != nil {
		return nil, err
	}

	twkm, err := teewalletkeymanager.NewTeeWalletKeyManager(addresses.TeeWalletKeyManager, chainClient)
	if err != nil {
		return nil, err
	}

	twpm, err := teewalletprojectmanager.NewTeeWalletProjectManager(addresses.TeeWalletProjectManager, chainClient)
	if err != nil {
		return nil, err
	}

	sm, err := system.NewFlareSystemsManager(addresses.FlareSystemManager, chainClient)
	if err != nil {
		return nil, err
	}

	ftdc, err := fdc2hub.NewFdc2Hub(addresses.Fdc2Hub, chainClient)
	if err != nil {
		return nil, err
	}

	tv, err := teeverification.NewTeeVerification(addresses.TeeVerification, chainClient)
	if err != nil {
		return nil, err
	}

	ter, err := teeextensionregistry.NewTeeExtensionRegistry(addresses.TeeExtensionRegistry, chainClient)
	if err != nil {
		return nil, err
	}

	toal, err := teeownerallowlist.NewTeeOwnerAllowlist(addresses.TeeOwnerAllowlist, chainClient)
	if err != nil {
		return nil, err
	}

	chainID, err := chainClient.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	return &Support{
		Prv:                     prv,
		TeeMachineRegistry:      tr,
		TeeWalletManager:        twm,
		TeeWalletKeyManager:     twkm,
		TeeWalletProjectManager: twpm,
		FlareSystemManager:      sm,
		Fdc2Hub:                 ftdc,
		TeeVerification:         tv,
		TeeExtensionRegistry:    ter,
		ChainClient:             chainClient,
		ChainID:                 chainID,
		TeeOwnerAllowlist:       toal,
		Addresses:               addresses,
	}, nil
}

func CheckTx(tx *types.Transaction, client *ethclient.Client) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}
	if receipt.Status == 0 {
		reason, err := getFailingMessage(client, tx.Hash())
		if err != nil {
			return nil, err
		}
		return nil, errors.Errorf("error: Transaction fail: %s", reason)
	}

	return receipt, nil
}

func getFailingMessage(client *ethclient.Client, hash common.Hash) (string, error) {
	tx, _, err := client.TransactionByHash(context.Background(), hash)
	if err != nil {
		return "", errors.Errorf("%s", err)
	}

	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return "", errors.Errorf("%s", err)
	}

	msg := ethereum.CallMsg{
		From:     from,
		To:       tx.To(),
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice(),
		Value:    tx.Value(),
		Data:     tx.Data(),
	}

	res, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return "", errors.Errorf("%s", err)
	}

	return string(res), nil
}

// RawContract mirrors the JSON entries
type RawContract struct {
	Name         string `json:"name"`
	ContractName string `json:"contractName"`
	Address      string `json:"address"`
}

func ParseAddresses(filePath string) (*Addresses, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var raw []RawContract
	err = json.Unmarshal(file, &raw)
	if err != nil {
		return nil, err
	}

	var addrs Addresses
	targets := fieldMap(&addrs)

	for _, c := range raw {
		if dest, ok := targets[c.Name]; ok {
			*dest = common.HexToAddress(c.Address)
		}
	}

	return &addrs, err
}

// fieldMap builds: "<json tag or field name>" -> pointer to field
func fieldMap(addrStruct *Addresses) map[string]*common.Address {
	out := make(map[string]*common.Address)

	v := reflect.ValueOf(addrStruct).Elem()
	t := v.Type()
	addrType := reflect.TypeOf(common.Address{})

	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		ft := t.Field(i)

		if fv.Type() != addrType {
			continue
		}

		tag := ft.Tag.Get("json")
		key := tag
		if key == "" || key == "-" {
			key = ft.Name
		}

		out[key] = fv.Addr().Interface().(*common.Address)
	}

	return out
}
