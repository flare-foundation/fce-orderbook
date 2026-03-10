# Extension Development Guide

This guide explains how the extension scaffold works and how to implement your own logic.

## How an Extension Works

An extension is an HTTP server that runs inside a Trusted Execution Environment (TEE). It receives instructions from the blockchain, processes them, and returns results. The full lifecycle:

```
1. User calls your Solidity contract (on-chain)
2. Contract emits a TeeInstructionsSent event via TeeExtensionRegistry
3. TEE proxy picks up the instruction from the chain
4. TEE node fetches the instruction from the proxy
5. TEE node forwards it as POST /action to your extension server
6. Your extension processes the action and returns a result
7. TEE node sends the result back to the proxy
8. Caller polls the proxy for the result
```

Your extension controls steps 1 (the contract) and 6 (the action handler). Everything else is handled by the TEE infrastructure.

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│  YOUR CODE (what you customize)                     │
│                                                     │
│  contracts/InstructionSender.sol    On-chain entry   │
│  internal/config/config.go         OPType constants  │
│  internal/extension/extension.go   Action handlers   │
│  pkg/types/types.go                Request/response  │
│  tools/cmd/run-test/main.go        E2E tests        │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  INFRASTRUCTURE (do not modify)                     │
│                                                     │
│  cmd/main.go                       Server entry      │
│  pkg/server/server.go              Server wrapper    │
│  buildResult()                     Result builder    │
│  actionHandler()                   HTTP handler      │
│  scripts/*                         Build/deploy      │
│  tools/cmd/deploy-contract/        Deployment        │
│  tools/cmd/register-*/             Registration      │
└─────────────────────────────────────────────────────┘
```

## The Files You Modify

### 1. `internal/config/config.go` — Operation Type Constants

This file defines the string constants for your operation types. Each constant is hashed to `bytes32` at runtime using `teeutils.ToHash()` and compared against the `OPType` field in incoming actions.

**What to do:** Add one constant per operation your extension supports. The scaffold defines one:

```go
const (
    OPTypeSayHello = "SAY_HELLO"
)
```

These strings must exactly match the `bytes32` constants in your Solidity contract:

```solidity
bytes32 constant OP_TYPE_SAY_HELLO = bytes32("SAY_HELLO");
```

### 2. `pkg/types/types.go` — Request and Response Types

This file defines the JSON structures for your extension's inputs and outputs.

**Request types** are decoded from the instruction's `OriginalMessage` field. This is the payload the user sent through your Solidity contract.

**Response types** are what your extension returns in `ActionResult.Data`. The caller receives this when polling the proxy.

**State type** represents your extension's observable state, returned by `GET /state`. The TEE infrastructure uses this for state synchronization.

The scaffold defines:

```go
// What the user sends (via the Solidity contract)
type SayHelloRequest struct {
    Name string `json:"name"`
}

// What your extension returns
type SayHelloResponse struct {
    Greeting       string `json:"greeting"`
    GreetingNumber int    `json:"greetingNumber"`
}

// Your extension's cumulative state
type State struct {
    GreetingCount int    `json:"greetingCount"`
    LastGreeting  string `json:"lastGreeting"`
}
```

### 3. `internal/extension/extension.go` — Action Handlers

This is the main file. It contains:

- **Extension struct** — your in-memory state fields
- **processAction()** — routes incoming actions to handlers based on OPType
- **Your handler functions** — the actual business logic

#### The Extension Struct

Add fields to hold your extension's state. Always protect state access with the `mu` mutex. The scaffold tracks greeting count and the last greeting sent:

```go
type Extension struct {
    mu     sync.RWMutex
    Server *http.Server

    greetingCount int
    lastGreeting  string
}
```

#### processAction() — The Router

This function receives every action and routes it. Add a `case` for each operation type:

```go
func (e *Extension) processAction(action teetypes.Action) (int, []byte) {
    dataFixed, err := processorutils.Parse[instruction.DataFixed](action.Data.Message)
    if err != nil {
        return http.StatusBadRequest, []byte(fmt.Sprintf("decoding fixed data: %v", err))
    }

    switch {
    case dataFixed.OPType == teeutils.ToHash(config.OPTypeSayHello):
        ar := e.processSayHello(action, dataFixed)
        b, _ := json.Marshal(ar)
        return http.StatusOK, b

    default:
        return http.StatusNotImplemented, []byte("unsupported op type")
    }
}
```

#### Handler Functions

Each handler follows the same 4-step pattern. Here's the scaffold's `processSayHello`:

```go
func (e *Extension) processSayHello(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
    // 1. Decode the incoming message
    var req types.SayHelloRequest
    dec := json.NewDecoder(bytes.NewReader(df.OriginalMessage))
    dec.DisallowUnknownFields()
    err := dec.Decode(&req)
    if err != nil {
        return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
    }

    // 2. Validate
    if req.Name == "" {
        return buildResult(action, df, nil, 0, fmt.Errorf("name must not be empty"))
    }

    // 3. Execute business logic
    e.mu.Lock()
    e.greetingCount++
    greetingNumber := e.greetingCount
    greeting := fmt.Sprintf("Hello, %s! Welcome to Flare Confidential Compute.", req.Name)
    e.lastGreeting = greeting
    e.mu.Unlock()

    // 4. Build response
    resp := types.SayHelloResponse{
        Greeting:       greeting,
        GreetingNumber: greetingNumber,
    }
    data, _ := json.Marshal(resp)

    return buildResult(action, df, data, 1, nil)
}
```

**`buildResult` parameters:**
- `status = 0` → error. The `err` parameter is logged.
- `status = 1` → success. The `data` parameter is returned to the caller.

### 4. `contracts/InstructionSender.sol` — On-Chain Entry Point

This contract is the only address allowed to submit instructions to your extension. You define `bytes32` constants for each operation type (matching your Go constants) and add one send function per operation type. After modifying, run `./scripts/generate-bindings.sh` to regenerate Go bindings.

See the **[InstructionSender Contract Guide](instruction-sender.md)** for a full walkthrough and examples.

### 5. `tools/cmd/run-test/main.go` — E2E Tests

The test runner sends instructions through the full pipeline (contract → TEE → proxy) and verifies results. You define test payloads, send them, and assert on your response fields.

See the **[Testing Guide](testing.md)** for details on writing and running tests.

## How the Pieces Connect

The critical link between your Solidity contract and Go code is the **OPType string**. It must be identical in three places:

| Location | Example |
|----------|---------|
| Solidity contract | `bytes32 constant OP_TYPE_SAY_HELLO = bytes32("SAY_HELLO");` |
| Go config | `OPTypeSayHello = "SAY_HELLO"` |
| Go router | `case dataFixed.OPType == teeutils.ToHash(config.OPTypeSayHello):` |

If these don't match, the action will fall through to the `default` case and return "unsupported op type".

## Data Flow Through the Extension

```
Solidity contract
    │
    │  _message (raw bytes, typically JSON)
    ▼
TeeExtensionRegistry.sendInstructions()
    │
    │  wraps into DataFixed{OPType, OPCommand, OriginalMessage}
    ▼
TEE node → POST /action → actionHandler()
    │
    │  decodes teetypes.Action from request body
    ▼
processAction()
    │
    │  parses DataFixed from action.Data.Message
    │  routes based on dataFixed.OPType
    ▼
processSayHello()
    │
    │  decodes SayHelloRequest from df.OriginalMessage
    │  executes greeting logic, increments counter
    │  returns ActionResult with SayHelloResponse in Data field
    ▼
buildResult() → JSON response → TEE node → proxy → caller
```

Key types in the flow:
- `teetypes.Action` — the envelope from the TEE node (contains `Data.Message`, `Data.ID`, etc.)
- `instruction.DataFixed` — parsed from `Action.Data.Message` (contains `OPType`, `OPCommand`, `OriginalMessage`)
- `df.OriginalMessage` — the raw `_message` bytes from the Solidity contract (your JSON payload)
- `teetypes.ActionResult` — what you return (contains `Status`, `Data`, `Log`)

## Using the TEE Signing Port

Extensions can request the TEE to sign data or perform cryptographic operations through the sign port (default: 9090, configurable via `SIGN_PORT`). This is useful for extensions that need to:

- Sign transactions on behalf of the TEE
- Generate verifiable attestations
- Encrypt/decrypt data using TEE-managed keys

The sign port is available at `localhost:{SIGN_PORT}` from within the extension.

## Step-by-Step: Adding a New Operation

1. **Add the OPType constant** in `internal/config/config.go`
2. **Define request/response types** in `pkg/types/types.go`
3. **Add a case** in `processAction()` in `internal/extension/extension.go`
4. **Write the handler function** following the 4-step pattern
5. **Add the Solidity constant and send function** in `contracts/InstructionSender.sol`
6. **Regenerate bindings**: `./scripts/generate-bindings.sh`
7. **Update the Go tooling** if the send function has a different signature (see `tools/pkg/utils/instructions.go`)
8. **Add a test case** in `tools/cmd/run-test/main.go`

## Common Patterns

### Returning errors to the caller

Use `status = 0` in `buildResult`. The error message goes into `ActionResult.Log`:

```go
if req.Name == "" {
    return buildResult(action, df, nil, 0, fmt.Errorf("name must not be empty"))
}
```

### Maintaining state across actions

Add fields to the `Extension` struct and protect with the mutex:

```go
e.mu.Lock()
e.greetingCount++
e.lastGreeting = greeting
e.mu.Unlock()
```

Return state in `stateHandler()` via the `types.State` struct.

### Multiple operation types

Each operation type gets its own handler function. They share the same `Extension` struct, so they can read/write the same state.
