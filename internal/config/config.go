// Package config contains configuration values and defaults used by the extension.
package config

import (
	"os"
	"strconv"
	"time"
)

const (
	Version = "0.1.0"

	// --- CUSTOMIZE: Define your operation type constants here. ---
	// Each constant must match a bytes32 value in your Solidity contract.
	// Example: if your contract has `bytes32 constant OP_TYPE_PLACE_ORDER = bytes32("PLACE_ORDER")`,
	// add `OPTypePlaceOrder = "PLACE_ORDER"` here.
	//
	// The scaffold ships with one placeholder. Replace or extend it with your own.
	OPTypeMyAction = "MY_ACTION"
	// OPTypeAnotherAction = "ANOTHER_ACTION"

	TimeoutShutdown = 5 * time.Second
)

// Defaults.
var (
	ExtensionPort = 8080
	SignPort      = 9090
)

// Environment variables override defaults.
func init() {
	ep := os.Getenv("EXTENSION_PORT")
	sp := os.Getenv("SIGN_PORT")

	if ep != "" {
		if v, err := strconv.Atoi(ep); err == nil {
			ExtensionPort = v
		}
	}
	if sp != "" {
		if v, err := strconv.Atoi(sp); err == nil {
			SignPort = v
		}
	}
}
