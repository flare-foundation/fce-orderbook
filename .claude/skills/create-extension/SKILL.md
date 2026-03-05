# Create Extension

Guides the developer through implementing their extension's business logic — the core "what does your extension do?" workflow.

## When to Use

The user wants to implement their extension logic: define operations, write handlers, and wire up the Solidity contract. They may say things like:
- "create my extension"
- "add an operation"
- "implement my extension logic"
- "add a new op type"
- "/create-extension"

## Inputs

The skill needs to know what **operation(s)** the extension should support. For each operation, gather:
- **Name** (e.g., "PlaceOrder", "Transfer", "Swap")
- **Description** — what it does
- **Request fields** — what the user sends (JSON payload)
- **Response fields** — what the extension returns

How to determine:
1. **User described it** — use what they said
2. **User is vague** — ask: "What operation(s) should your extension support? For each, describe the name, what it does, and what data it takes/returns."

Before starting, read the current state of the files to understand what's already implemented (the scaffold may already be renamed, or some operations may exist).

## Steps to Execute

All paths are relative to the scaffold root (the directory containing `foundry.toml`).

### Step 1: Add OPType constant(s) in `internal/config/config.go`

Read the file first. Add one constant per operation:

```go
const (
    OPTypePlaceOrder  = "PLACE_ORDER"
    OPTypeCancelOrder = "CANCEL_ORDER"
)
```

Use UPPER_SNAKE_CASE for the string values. These strings must exactly match the `bytes32` constants you'll add in Solidity.

### Step 2: Define request/response/state types in `pkg/types/types.go`

Read the file first. Add structs for each operation's request and response, plus update the State struct:

```go
// Request — decoded from df.OriginalMessage
type PlaceOrderRequest struct {
    Symbol string  `json:"symbol"`
    Side   string  `json:"side"`
    Amount float64 `json:"amount"`
    Price  float64 `json:"price"`
}

// Response — returned in ActionResult.Data
type PlaceOrderResponse struct {
    OrderID string `json:"orderId"`
    Status  string `json:"status"`
}

// State — returned by GET /state
type State struct {
    TotalOrders int    `json:"totalOrders"`
    LastOrderID string `json:"lastOrderId"`
}
```

### Step 3: Add case(s) in `processAction()` router in `internal/extension/extension.go`

Read the file first. Add a `case` in the `switch` block for each new operation:

```go
case dataFixed.OPType == teeutils.ToHash(config.OPTypePlaceOrder):
    ar := e.processPlaceOrder(action, dataFixed)
    b, _ := json.Marshal(ar)
    return http.StatusOK, b
```

### Step 4: Write handler function(s) following the 4-step pattern

Each handler follows this exact pattern:

```go
func (e *Extension) processPlaceOrder(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
    // 1. Decode the incoming message
    var req types.PlaceOrderRequest
    dec := json.NewDecoder(bytes.NewReader(df.OriginalMessage))
    dec.DisallowUnknownFields()
    err := dec.Decode(&req)
    if err != nil {
        return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
    }

    // 2. Validate
    if req.Amount <= 0 {
        return buildResult(action, df, nil, 0, fmt.Errorf("amount must be positive"))
    }

    // 3. Execute business logic
    orderID := generateOrderID()
    // ... your logic here ...

    // 4. Build response
    resp := types.PlaceOrderResponse{OrderID: orderID, Status: "placed"}
    data, _ := json.Marshal(resp)

    e.mu.Lock()
    e.totalOrders++
    e.mu.Unlock()

    return buildResult(action, df, data, 1, nil)
}
```

**`buildResult` status codes:**
- `0` = error — the `err` parameter message goes into `ActionResult.Log`
- `1` = success — the `data` parameter is returned to the caller in `ActionResult.Data`

### Step 5: Update `Extension` struct and `stateHandler()`

Add state fields to the `Extension` struct and wire them into `stateHandler()` via the `types.State` struct. Always protect state access with the `mu` mutex:

```go
e.mu.Lock()
e.orderBook[order.ID] = order
e.totalOrders++
e.mu.Unlock()
```

### Step 6: Add Solidity constant(s) and send function(s) in `contracts/InstructionSender.sol`

Read the file first. Add a `bytes32` constant for each operation and a send function:

```solidity
bytes32 constant OP_TYPE_PLACE_ORDER = bytes32("PLACE_ORDER");

function sendPlaceOrder(bytes calldata _message) external payable {
    _sendInstruction(OP_TYPE_PLACE_ORDER, bytes32(0), _message);
}
```

The OPType string in Solidity (`"PLACE_ORDER"`) must **exactly match** the Go constant (`OPTypePlaceOrder = "PLACE_ORDER"`).

### Step 7: Regenerate bindings

Run from the scaffold root:

```bash
./scripts/generate-bindings.sh
```

This compiles the Solidity contract and generates Go bindings in `tools/pkg/contracts/`.

### Step 8: Update Go tooling in `tools/pkg/utils/instructions.go`

Read the file first. If you added new send functions in Solidity with different signatures, add corresponding Go helper functions that call the new contract methods. The existing `SendInstruction` helper calls the scaffold's default send function — add similar helpers for your new functions if needed.

## Data Flow Reference

```
Solidity contract
    |  _message (raw bytes, typically JSON)
    v
TeeExtensionRegistry.sendInstructions()
    |  wraps into DataFixed{OPType, OPCommand, OriginalMessage}
    v
TEE node -> POST /action -> actionHandler()
    |  decodes teetypes.Action from request body
    v
processAction()
    |  parses DataFixed from action.Data.Message
    |  routes based on dataFixed.OPType
    v
your handler function
    |  decodes YOUR request type from df.OriginalMessage
    |  executes YOUR logic
    |  returns ActionResult with YOUR response in Data field
    v
buildResult() -> JSON response -> TEE node -> proxy -> caller
```

## Verification

After all steps, run from the scaffold root:

```bash
cd tools && go build ./...
```

Then from the root module:

```bash
go build ./...
```

If both succeed, all imports and type references are correct. Report the result to the user.

## Important Notes

- **Do NOT modify infrastructure code** — functions like `buildResult()`, `actionHandler()`, `stateHandler()` (the generic parts), and files in `cmd/main.go`, `pkg/server/` are boilerplate marked "DO NOT MODIFY".
- **Always read each file before editing** to confirm current content.
- **OPType strings must match exactly** across Solidity (`bytes32("PLACE_ORDER")`) and Go (`OPTypePlaceOrder = "PLACE_ORDER"`). If they don't match, actions will fall through to the `default` case and return "unsupported op type".
- **Run `./scripts/generate-bindings.sh`** after any Solidity changes.
- Use `replace_all: true` when replacing identifiers that appear multiple times in a file.
