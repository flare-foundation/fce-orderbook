// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package myextension

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// MyExtensionInstructionSenderMetaData contains all meta data concerning the MyExtensionInstructionSender contract.
var MyExtensionInstructionSenderMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_teeExtensionRegistry\",\"type\":\"address\",\"internalType\":\"contractITeeExtensionRegistry\"},{\"name\":\"_teeMachineRegistry\",\"type\":\"address\",\"internalType\":\"contractITeeMachineRegistry\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"sendMyInstruction\",\"inputs\":[{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"setExtensionId\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"teeExtensionRegistry\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractITeeExtensionRegistry\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"teeMachineRegistry\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractITeeMachineRegistry\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x0000",
}

// MyExtensionInstructionSenderABI is the input ABI used to generate the binding from.
// Deprecated: Use MyExtensionInstructionSenderMetaData.ABI instead.
var MyExtensionInstructionSenderABI = MyExtensionInstructionSenderMetaData.ABI

// MyExtensionInstructionSenderBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MyExtensionInstructionSenderMetaData.Bin instead.
var MyExtensionInstructionSenderBin = MyExtensionInstructionSenderMetaData.Bin

// DeployMyExtensionInstructionSender deploys a new Ethereum contract, binding an instance of MyExtensionInstructionSender to it.
func DeployMyExtensionInstructionSender(auth *bind.TransactOpts, backend bind.ContractBackend, _teeExtensionRegistry common.Address, _teeMachineRegistry common.Address) (common.Address, *types.Transaction, *MyExtensionInstructionSender, error) {
	parsed, err := MyExtensionInstructionSenderMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MyExtensionInstructionSenderBin), backend, _teeExtensionRegistry, _teeMachineRegistry)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MyExtensionInstructionSender{MyExtensionInstructionSenderCaller: MyExtensionInstructionSenderCaller{contract: contract}, MyExtensionInstructionSenderTransactor: MyExtensionInstructionSenderTransactor{contract: contract}, MyExtensionInstructionSenderFilterer: MyExtensionInstructionSenderFilterer{contract: contract}}, nil
}

// MyExtensionInstructionSender is an auto generated Go binding around an Ethereum contract.
type MyExtensionInstructionSender struct {
	MyExtensionInstructionSenderCaller     // Read-only binding to the contract
	MyExtensionInstructionSenderTransactor // Write-only binding to the contract
	MyExtensionInstructionSenderFilterer   // Log filterer for contract events
}

// MyExtensionInstructionSenderCaller is an auto generated read-only Go binding around an Ethereum contract.
type MyExtensionInstructionSenderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MyExtensionInstructionSenderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MyExtensionInstructionSenderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MyExtensionInstructionSenderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MyExtensionInstructionSenderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MyExtensionInstructionSenderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MyExtensionInstructionSenderSession struct {
	Contract     *MyExtensionInstructionSender // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// MyExtensionInstructionSenderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MyExtensionInstructionSenderCallerSession struct {
	Contract *MyExtensionInstructionSenderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// MyExtensionInstructionSenderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MyExtensionInstructionSenderTransactorSession struct {
	Contract     *MyExtensionInstructionSenderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// MyExtensionInstructionSenderRaw is an auto generated low-level Go binding around an Ethereum contract.
type MyExtensionInstructionSenderRaw struct {
	Contract *MyExtensionInstructionSender // Generic contract binding to access the raw methods on
}

// MyExtensionInstructionSenderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MyExtensionInstructionSenderCallerRaw struct {
	Contract *MyExtensionInstructionSenderCaller // Generic read-only contract binding to access the raw methods on
}

// MyExtensionInstructionSenderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MyExtensionInstructionSenderTransactorRaw struct {
	Contract *MyExtensionInstructionSenderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMyExtensionInstructionSender creates a new instance of MyExtensionInstructionSender, bound to a specific deployed contract.
func NewMyExtensionInstructionSender(address common.Address, backend bind.ContractBackend) (*MyExtensionInstructionSender, error) {
	contract, err := bindMyExtensionInstructionSender(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MyExtensionInstructionSender{MyExtensionInstructionSenderCaller: MyExtensionInstructionSenderCaller{contract: contract}, MyExtensionInstructionSenderTransactor: MyExtensionInstructionSenderTransactor{contract: contract}, MyExtensionInstructionSenderFilterer: MyExtensionInstructionSenderFilterer{contract: contract}}, nil
}

// NewMyExtensionInstructionSenderCaller creates a new read-only instance of MyExtensionInstructionSender, bound to a specific deployed contract.
func NewMyExtensionInstructionSenderCaller(address common.Address, caller bind.ContractCaller) (*MyExtensionInstructionSenderCaller, error) {
	contract, err := bindMyExtensionInstructionSender(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MyExtensionInstructionSenderCaller{contract: contract}, nil
}

// NewMyExtensionInstructionSenderTransactor creates a new write-only instance of MyExtensionInstructionSender, bound to a specific deployed contract.
func NewMyExtensionInstructionSenderTransactor(address common.Address, transactor bind.ContractTransactor) (*MyExtensionInstructionSenderTransactor, error) {
	contract, err := bindMyExtensionInstructionSender(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MyExtensionInstructionSenderTransactor{contract: contract}, nil
}

// NewMyExtensionInstructionSenderFilterer creates a new log filterer instance of MyExtensionInstructionSender, bound to a specific deployed contract.
func NewMyExtensionInstructionSenderFilterer(address common.Address, filterer bind.ContractFilterer) (*MyExtensionInstructionSenderFilterer, error) {
	contract, err := bindMyExtensionInstructionSender(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MyExtensionInstructionSenderFilterer{contract: contract}, nil
}

// bindMyExtensionInstructionSender binds a generic wrapper to an already deployed contract.
func bindMyExtensionInstructionSender(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MyExtensionInstructionSenderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MyExtensionInstructionSender.Contract.MyExtensionInstructionSenderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.MyExtensionInstructionSenderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.MyExtensionInstructionSenderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MyExtensionInstructionSender.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.contract.Transact(opts, method, params...)
}

// TeeExtensionRegistry is a free data retrieval call binding the contract method 0xa435d58a.
//
// Solidity: function teeExtensionRegistry() view returns(address)
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderCaller) TeeExtensionRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MyExtensionInstructionSender.contract.Call(opts, &out, "teeExtensionRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TeeExtensionRegistry is a free data retrieval call binding the contract method 0xa435d58a.
//
// Solidity: function teeExtensionRegistry() view returns(address)
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderSession) TeeExtensionRegistry() (common.Address, error) {
	return _MyExtensionInstructionSender.Contract.TeeExtensionRegistry(&_MyExtensionInstructionSender.CallOpts)
}

// TeeExtensionRegistry is a free data retrieval call binding the contract method 0xa435d58a.
//
// Solidity: function teeExtensionRegistry() view returns(address)
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderCallerSession) TeeExtensionRegistry() (common.Address, error) {
	return _MyExtensionInstructionSender.Contract.TeeExtensionRegistry(&_MyExtensionInstructionSender.CallOpts)
}

// TeeMachineRegistry is a free data retrieval call binding the contract method 0x524967d7.
//
// Solidity: function teeMachineRegistry() view returns(address)
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderCaller) TeeMachineRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MyExtensionInstructionSender.contract.Call(opts, &out, "teeMachineRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TeeMachineRegistry is a free data retrieval call binding the contract method 0x524967d7.
//
// Solidity: function teeMachineRegistry() view returns(address)
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderSession) TeeMachineRegistry() (common.Address, error) {
	return _MyExtensionInstructionSender.Contract.TeeMachineRegistry(&_MyExtensionInstructionSender.CallOpts)
}

// TeeMachineRegistry is a free data retrieval call binding the contract method 0x524967d7.
//
// Solidity: function teeMachineRegistry() view returns(address)
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderCallerSession) TeeMachineRegistry() (common.Address, error) {
	return _MyExtensionInstructionSender.Contract.TeeMachineRegistry(&_MyExtensionInstructionSender.CallOpts)
}

// SendMyInstruction is a paid mutator transaction binding the contract method.
//
// Solidity: function sendMyInstruction(bytes _message) payable returns()
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderTransactor) SendMyInstruction(opts *bind.TransactOpts, _message []byte) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.contract.Transact(opts, "sendMyInstruction", _message)
}

// SendMyInstruction is a paid mutator transaction binding the contract method.
//
// Solidity: function sendMyInstruction(bytes _message) payable returns()
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderSession) SendMyInstruction(_message []byte) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.SendMyInstruction(&_MyExtensionInstructionSender.TransactOpts, _message)
}

// SendMyInstruction is a paid mutator transaction binding the contract method.
//
// Solidity: function sendMyInstruction(bytes _message) payable returns()
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderTransactorSession) SendMyInstruction(_message []byte) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.SendMyInstruction(&_MyExtensionInstructionSender.TransactOpts, _message)
}

// SetExtensionId is a paid mutator transaction binding the contract method 0xaa5032c6.
//
// Solidity: function setExtensionId() returns()
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderTransactor) SetExtensionId(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MyExtensionInstructionSender.contract.Transact(opts, "setExtensionId")
}

// SetExtensionId is a paid mutator transaction binding the contract method 0xaa5032c6.
//
// Solidity: function setExtensionId() returns()
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderSession) SetExtensionId() (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.SetExtensionId(&_MyExtensionInstructionSender.TransactOpts)
}

// SetExtensionId is a paid mutator transaction binding the contract method 0xaa5032c6.
//
// Solidity: function setExtensionId() returns()
func (_MyExtensionInstructionSender *MyExtensionInstructionSenderTransactorSession) SetExtensionId() (*types.Transaction, error) {
	return _MyExtensionInstructionSender.Contract.SetExtensionId(&_MyExtensionInstructionSender.TransactOpts)
}
