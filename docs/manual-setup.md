# Manual Setup Guide

**You must rename the placeholder names before running any scripts.** The scaffold ships with generic names (`MyExtensionInstructionSender`, `myextension`, `MY_ACTION`) that need to be replaced with your extension's actual names. The build scripts will refuse to run until you do this.

If you're using [Claude Code](https://claude.ai/code), you can run `/rename-scaffold` to do all of this automatically — just tell it your extension name and it will handle steps 1-5. You can also ask Claude to help you customize the Solidity contract with your own action types and send functions. #TODO: Generalize this to other coding agents

The manual steps are below, using "Orderbook" / "orderbook" as an example — substitute your own name.

## 1. Rename the Solidity contract

**File:** `contracts/InstructionSender.sol`

```solidity
// Before:
contract MyExtensionInstructionSender {
    bytes32 constant OP_TYPE_MY_ACTION = bytes32("MY_ACTION");

// After:
contract OrderbookInstructionSender {
    bytes32 constant OP_TYPE_PLACE_ORDER = bytes32("PLACE_ORDER");
```

## 2. Update `generate-bindings.sh` config

**File:** `scripts/generate-bindings.sh`

```bash
# Before:
CONTRACT_NAME="MyExtensionInstructionSender"
GO_PKG="myextension"

# After:
CONTRACT_NAME="OrderbookInstructionSender"
GO_PKG="orderbook"
```

## 3. Rename the Go bindings directory

```bash
mv tools/pkg/contracts/myextension tools/pkg/contracts/orderbook
```

## 4. Update the `go:generate` directive

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

## 5. Update Go imports

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

## Summary checklist

| # | What | File | Change |
|---|------|------|--------|
| 1 | Rename the Solidity contract | `contracts/InstructionSender.sol` | `MyExtensionInstructionSender` → `YourNameInstructionSender` |
| 2 | Update script config | `scripts/generate-bindings.sh` | `CONTRACT_NAME` and `GO_PKG` variables |
| 3 | Rename Go bindings directory | `tools/pkg/contracts/myextension/` | Rename to match `GO_PKG` |
| 4 | Update `go:generate` directive | `tools/pkg/contracts/<yourpkg>/*.go` | `--abi`, `--bin`, `--pkg`, `--type` flags |
| 5 | Update Go imports | `tools/pkg/utils/instructions.go`, `tools/cmd/deploy-contract/main.go` | Import path + type names |
