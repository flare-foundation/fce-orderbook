# Extension Scaffold

A template repository for building Flare Confidential Compute (FCC) extensions. This scaffold provides the on-chain contracts, deployment tooling, and registration scripts you need to get a new extension running on the Flare TEE infrastructure.

## Setup (First Steps)

**You must rename the placeholder names before running any scripts.** The scaffold ships with generic names (`MyExtensionInstructionSender`, `myextension`, `MY_ACTION`) that need to be replaced with your extension's actual names. The build scripts will refuse to run until you do this.

If you're using [Claude Code](https://claude.ai/code), you can run `/rename-scaffold` to do all of this automatically — just tell it your extension name and it will handle steps 1-5. You can also ask Claude to help you customize the Solidity contract with your own action types and send functions. #TODO: Generalize this to other coding agents

The manual steps are below, using "Orderbook" / "orderbook" as an example — substitute your own name.

### 1. Rename the Solidity contract

**File:** `contracts/InstructionSender.sol`

```solidity
// Before:
contract MyExtensionInstructionSender {
    bytes32 constant OP_TYPE_MY_ACTION = bytes32("MY_ACTION");

// After:
contract OrderbookInstructionSender {
    bytes32 constant OP_TYPE_PLACE_ORDER = bytes32("PLACE_ORDER");
```

### 2. Update `generate-bindings.sh` config

**File:** `scripts/generate-bindings.sh`

```bash
# Before:
CONTRACT_NAME="MyExtensionInstructionSender"
GO_PKG="myextension"

# After:
CONTRACT_NAME="OrderbookInstructionSender"
GO_PKG="orderbook"
```

### 3. Rename the Go bindings directory

```bash
mv tools/pkg/contracts/myextension tools/pkg/contracts/orderbook
```

### 4. Update the `go:generate` directive

**File:** `tools/pkg/contracts/orderbook/myextension.go` (rename this file too)

```bash
mv tools/pkg/contracts/orderbook/myextension.go tools/pkg/contracts/orderbook/orderbook.go
```

Update the directive inside:
```go
// Before:
//go:generate abigen --abi MyExtensionInstructionSender.abi --bin MyExtensionInstructionSender.bin --pkg myextension --type MyExtensionInstructionSender --out autogen.go

// After:
//go:generate abigen --abi OrderbookInstructionSender.abi --bin OrderbookInstructionSender.bin --pkg orderbook --type OrderbookInstructionSender --out autogen.go
```

### 5. Update Go imports

**File:** `tools/pkg/utils/instructions.go`
```go
// Before:
import "your-module/pkg/contracts/myextension"
// ... myextension.DeployMyExtensionInstructionSender(...)

// After:
import "your-module/pkg/contracts/orderbook"
// ... orderbook.DeployOrderbookInstructionSender(...)
```

**File:** `tools/cmd/deploy-contract/main.go`
```go
// Before:
import "your-module/pkg/contracts/myextension"

// After:
import "your-module/pkg/contracts/orderbook"
```

### Summary checklist

| # | What | File | Change |
|---|------|------|--------|
| 1 | Rename the Solidity contract | `contracts/InstructionSender.sol` | `MyExtensionInstructionSender` → `YourNameInstructionSender` |
| 2 | Update script config | `scripts/generate-bindings.sh` | `CONTRACT_NAME` and `GO_PKG` variables |
| 3 | Rename Go bindings directory | `tools/pkg/contracts/myextension/` | Rename to match `GO_PKG` |
| 4 | Update `go:generate` directive | `tools/pkg/contracts/<yourpkg>/*.go` | `--abi`, `--bin`, `--pkg`, `--type` flags |
| 5 | Update Go imports | `tools/pkg/utils/instructions.go`, `tools/cmd/deploy-contract/main.go` | Import path + type names |

## Prerequisites

- **Go 1.25.1+**
- **Foundry** (`forge`) — compiles the Solidity contract
- **abigen** (go-ethereum) — generates Go bindings from compiled contract
- **jq** — extracts ABI/bytecode from Foundry output
- **Running local infrastructure** — Hardhat node with deployed Flare contracts (via `docker compose up` from the `e2e/` repo)

## Quick Start

After completing the [Setup](#setup-first-steps) steps above, with local infrastructure running (`docker compose up` from `e2e/`):

```bash
./scripts/pre-build.sh
```

This does four things:
1. Compiles `contracts/InstructionSender.sol` and generates Go bindings
2. Deploys your `InstructionSender` contract to the local chain
3. Registers the extension on the `TeeExtensionRegistry`
4. Writes `config/extension.env` with `EXTENSION_ID` and `INSTRUCTION_SENDER`

The script auto-detects the deployed contract addresses file. To override:

```bash
ADDRESSES_FILE=/path/to/deployed-addresses.json ./scripts/pre-build.sh
```

For a non-default chain RPC:

```bash
CHAIN_URL=http://your-node:8545 ./scripts/pre-build.sh
```

### Verify output

```bash
cat config/extension.env
```

You should see:
```
EXTENSION_ID=0x0000000000000000000000000000000000000000000000000000000000000001
INSTRUCTION_SENDER=0x1234...abcd
```

These values are used by Docker Compose (extension TEE + proxy) and the post-build script.

### 2. Run post-build

After Docker Compose brings up the extension TEE, proxy, and Redis:

```bash
./scripts/post-build.sh
```

This does two things:
1. Registers the TEE's code hash as an allowed version on the extension registry
2. Registers the extension TEE machine on-chain and brings it to production

The script waits for both the extension proxy and normal proxy to be ready before proceeding.

To override defaults:

```bash
EXT_PROXY_URL=http://localhost:6664 \
NORMAL_PROXY_URL=http://localhost:6662 \
./scripts/post-build.sh
```

## Repository Structure

```
├── contracts/
│   └── InstructionSender.sol          # Your extension's on-chain entry point
├── config/
│   └── extension.env                  # Generated by pre-build (gitignored)
├── foundry.toml                       # Foundry config for compiling contracts
├── scripts/
│   ├── pre-build.sh                   # Compile + deploy + register → writes config
│   ├── post-build.sh                  # Allow TEE version + register TEE on-chain
│   ├── test.sh                        # Send instructions + verify results
│   ├── full-setup.sh                  # Chains pre-build → post-build → test
│   └── generate-bindings.sh           # Compile contract → generate Go bindings
└── tools/
    ├── go.mod
    ├── cmd/
    │   ├── deploy-contract/main.go    # Deploys InstructionSender to chain
    │   ├── register-extension/main.go # Registers extension on TeeExtensionRegistry
    │   ├── allow-tee-version/main.go  # Registers TEE code hash as allowed version
    │   ├── register-tee/main.go       # Registers extension TEE machine on-chain
    │   └── run-test/main.go           # Sends instructions and verifies results
    └── pkg/
        ├── utils/instructions.go      # Deploy, SetExtensionId, SendInstruction helpers
        └── contracts/myextension/
            ├── myextension.go         # go:generate directive for abigen
            └── autogen.go             # Generated by generate-bindings.sh (gitignored)
```

## Running Individual Steps

You can run the deploy and register steps independently:

```bash
# Deploy only
cd tools && go run ./cmd/deploy-contract -a /path/to/deployed-addresses.json

# Register with a specific contract address
cd tools && go run ./cmd/register-extension \
  -a /path/to/deployed-addresses.json \
  --instructionSender 0xYourContractAddress
```

You can also run binding generation standalone:

```bash
./scripts/generate-bindings.sh
```

Post-build steps can also be run individually:

```bash
# Allow TEE version
cd tools && go run ./cmd/allow-tee-version \
  -a /path/to/deployed-addresses.json \
  -p http://localhost:6664

# Register TEE machine
cd tools && go run ./cmd/register-tee \
  -a /path/to/deployed-addresses.json \
  -p http://localhost:6664 \
  -ep http://localhost:6662 \
  -l
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDRESSES_FILE` | auto-detected | Path to `deployed-addresses.json` |
| `CHAIN_URL` | `http://127.0.0.1:8545` | Chain RPC endpoint |
| `PRIV_KEY` | Hardhat dev key | Funded private key for transactions |
| `EXT_PROXY_URL` | `http://localhost:6664` | Extension proxy URL (post-build, test) |
| `NORMAL_PROXY_URL` | `http://localhost:6662` | Normal/FTDC proxy URL (post-build) |
| `EXTENSION_OWNER_KEY` | (empty, falls back to `PRIV_KEY`) | Private key override for AddTeeVersion |
| `TEE_VERSION` | `v0.1.0` | Version string for TEE registration |
| `LOCAL_MODE` | `true` | Skip attestation in local dev |
| `WAIT_TIMEOUT` | `120` | Service wait timeout in seconds |
| `INSTRUCTION_SENDER` | from `config/extension.env` | InstructionSender contract address (test) |

## Testing

After post-build completes, you can send instructions to your extension and verify the results:

```bash
./scripts/test.sh
```

Or run everything in one shot:

```bash
./scripts/full-setup.sh --test
```

### What the test does

The test runner (`tools/cmd/run-test/main.go`) executes this lifecycle:

```
1. SetExtensionId()         ← Generic: tells the contract its extension ID (idempotent)
2. Send instruction          ← YOUR CODE: call your contract function with your payload
3. Wait for TEE processing   ← Generic: time.Sleep(5s)
4. Poll for result            ← Generic: utils.ActionResult() polls proxy (15 retries, 2s apart)
5. Validate response          ← YOUR CODE: unmarshal Data into your type, check your fields
```

Steps 1, 3, and 4 are the same for every extension. Steps 2 and 5 are what you customize.

### How to write tests for your extension

The scaffold's test sends `{"message": "hello"}` via `SendMyInstruction` and logs the response. For your extension, you need to change both what you send and how you verify the result.

#### 1. Define your message and response types

The scaffold has a placeholder `MyActionResponse`. Replace it with structs matching your extension:

```go
// What you send (JSON-encoded as the instruction payload)
type TransferRequest struct {
    From   string `json:"from"`
    To     string `json:"to"`
    Amount uint64 `json:"amount"`
}

// What your extension returns (in ActionResult.Data)
type TransferResponse struct {
    TxHash string `json:"txHash"`
    Status string `json:"status"`
}
```

#### 2. Send your instructions

Replace the `SendInstruction()` call with your own contract function. The scaffold calls `SendMyInstruction(bytes)`, but your contract will have different functions:

```go
// Scaffold example (replace this):
payload, _ := json.Marshal(map[string]string{"message": "hello"})
instructionId, _, err := instrutils.SendInstruction(s, addr, payload)

// Your extension (something like):
payload, _ := json.Marshal(TransferRequest{From: "...", To: "...", Amount: 100})
instructionId, _, err := instrutils.SendInstruction(s, addr, payload)
```

If your Solidity contract has multiple send functions (e.g. `sendTransfer()`, `sendSwap()`), you'll need to add corresponding Go helpers in `tools/pkg/utils/instructions.go` and call them here.

#### 3. Validate your responses

The `verifyResult` function receives the raw response from the proxy. The response envelope is always the same:

```json
{
  "result": {
    "id": "0x...",
    "status": 1,
    "log": "",
    "opType": "0x...",
    "opCommand": "0x...",
    "data": "<your extension's JSON response>"
  }
}
```

- `status`: `0` = failed, `1` = success, `2` = pending
- `log`: error message when `status == 0`
- `data`: your extension's response bytes — this is whatever your `processAction` handler returned via `buildResult`

The generic status checks are already in `verifyResult`. You customize the part that unmarshals and validates `data`:

```go
// ★ CUSTOM: unmarshal into YOUR response type
var resp TransferResponse
err = json.Unmarshal(actionResult.Data, &resp)
if err != nil {
    return errors.Errorf("failed to unmarshal response: %s", err)
}

// ★ CUSTOM: validate YOUR specific fields
if resp.TxHash == "" {
    return errors.New("expected non-empty TxHash")
}
if resp.Status != "confirmed" {
    return errors.Errorf("expected status 'confirmed', got %q", resp.Status)
}
```

#### 4. Add more test cases

The scaffold shows a single send+verify pair. For a real extension, add multiple test cases covering:

- Each op type your extension supports
- Success cases with valid inputs
- Edge cases (empty fields, boundary values)
- Error cases (invalid payloads that should return `status == 0`)

#### Matching op types between Solidity and Go

Your Solidity contract defines op types as `bytes32` constants:

```solidity
bytes32 constant OP_TYPE_PLACE_ORDER = bytes32("PLACE_ORDER");
```

Your Go extension's `processAction` routes on the same value:

```go
case dataFixed.OPType == teeutils.ToHash("PLACE_ORDER"):
    return e.handlePlaceOrder(action, dataFixed)
```

The test sends instructions through the contract function that uses that op type, and verifies the response matches what `handlePlaceOrder` returns.

### What you need to change (summary)

| Step | What to change | Where |
|------|---------------|-------|
| Response types | Define structs for your extension's responses | `tools/cmd/run-test/main.go` (top of file) |
| Message payloads | Create the JSON your contract function expects | `main()` in `run-test/main.go` |
| Send instructions | Call your contract's specific function(s) | `main()` in `run-test/main.go` |
| Validate responses | Unmarshal `Data` and assert your fields | `verifyResult()` in `run-test/main.go` |
| Add test scenarios | Add more send+verify pairs for each op type | `main()` in `run-test/main.go` |
