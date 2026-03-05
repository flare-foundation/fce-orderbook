// Package types contains types that could be useful to other apps when interacting with this extension.
package types

import "github.com/ethereum/go-ethereum/common"

// --- CUSTOMIZE: Define your request/response types below. ---
//
// Request types: the JSON payload that users send via the Solidity contract.
// These get decoded from DataFixed.OriginalMessage in your action handler.
//
// Response types: the JSON payload your extension returns in ActionResult.Data.
// These are what the caller receives when polling the proxy for results.

// MyActionRequest is a placeholder for incoming instruction payloads.
// Replace with your own fields (e.g., From, To, Amount for a transfer extension).
type MyActionRequest struct {
	// TODO: Add your request fields here.
	// Example:
	//   From   string `json:"from"`
	//   To     string `json:"to"`
	//   Amount uint64 `json:"amount"`
}

// MyActionResponse is a placeholder for outgoing result payloads.
// Replace with your own fields (e.g., TxHash, Status for a transfer extension).
type MyActionResponse struct {
	// TODO: Add your response fields here.
	// Example:
	//   TxHash string `json:"txHash"`
	//   Status string `json:"status"`
}

// --- CUSTOMIZE: Define your extension's observable state. ---
//
// State is returned by GET /state and represents your extension's current state.
// The TEE infrastructure uses this for state synchronization across machines.
// Add fields that represent the cumulative state of your extension.

// State holds the extension's observable state.
type State struct {
	// TODO: Add your state fields here.
	// Example:
	//   OrderCount int    `json:"orderCount"`
	//   LastOrder  string `json:"lastOrder"`
}

// --- DO NOT MODIFY below this line. ---

// StateResponse is the envelope returned by GET /state.
type StateResponse struct {
	StateVersion common.Hash `json:"stateVersion"`
	State        State       `json:"state"`
}
