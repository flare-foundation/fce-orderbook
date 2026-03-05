# Testing

After post-build completes, you can send instructions to your extension and verify the results:

```bash
./scripts/test.sh
```

Or run everything in one shot:

```bash
./scripts/full-setup.sh --test
```

## What the test does

The test runner (`tools/cmd/run-test/main.go`) executes this lifecycle:

```
1. SetExtensionId()         ← Generic: tells the contract its extension ID (idempotent)
2. Send instruction          ← YOUR CODE: call your contract function with your payload
3. Wait for TEE processing   ← Generic: time.Sleep(5s)
4. Poll for result            ← Generic: utils.ActionResult() polls proxy (15 retries, 2s apart)
5. Validate response          ← YOUR CODE: unmarshal Data into your type, check your fields
```

Steps 1, 3, and 4 are the same for every extension. Steps 2 and 5 are what you customize.

## How to write tests for your extension

The scaffold's test sends `{"name": "World"}` via `SendSayHello` and logs the response. For your extension, you need to change both what you send and how you verify the result.

### 1. Define your message and response types

The scaffold has a placeholder `SayHelloResponse`. Replace it with structs matching your extension:

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

### 2. Send your instructions

Replace the `SendInstruction()` call with your own contract function. The scaffold calls `SendSayHello(bytes)`, but your contract will have different functions:

```go
// Scaffold example (replace this):
payload, _ := json.Marshal(map[string]string{"name": "World"})
instructionId, _, err := instrutils.SendInstruction(s, addr, payload)

// Your extension (something like):
payload, _ := json.Marshal(TransferRequest{From: "...", To: "...", Amount: 100})
instructionId, _, err := instrutils.SendInstruction(s, addr, payload)
```

If your Solidity contract has multiple send functions (e.g. `sendTransfer()`, `sendSwap()`), you'll need to add corresponding Go helpers in `tools/pkg/utils/instructions.go` and call them here.

### 3. Validate your responses

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

### 4. Add more test cases

The scaffold shows a single send+verify pair. For a real extension, add multiple test cases covering:

- Each op type your extension supports
- Success cases with valid inputs
- Edge cases (empty fields, boundary values)
- Error cases (invalid payloads that should return `status == 0`)

### Matching op types between Solidity and Go

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

## What you need to change (summary)

| Step | What to change | Where |
|------|---------------|-------|
| Response types | Define structs for your extension's responses | `tools/cmd/run-test/main.go` (top of file) |
| Message payloads | Create the JSON your contract function expects | `main()` in `run-test/main.go` |
| Send instructions | Call your contract's specific function(s) | `main()` in `run-test/main.go` |
| Validate responses | Unmarshal `Data` and assert your fields | `verifyResult()` in `run-test/main.go` |
| Add test scenarios | Add more send+verify pairs for each op type | `main()` in `run-test/main.go` |
