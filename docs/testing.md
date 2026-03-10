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

## How the scaffold test works

The scaffold's test sends `{"name": "World"}` via `sendSayHello` and verifies the greeting response. Here's how each part works — when you build your own extension, you'll replace these with your own types, payloads, and assertions.

### 1. Define your message and response types

The scaffold defines `SayHelloResponse` at the top of the test file, mirroring the type from `pkg/types/types.go`:

```go
type SayHelloResponse struct {
    Greeting       string `json:"greeting"`
    GreetingNumber int    `json:"greetingNumber"`
}
```

Replace this with structs matching your extension's response types. These are defined separately in the test file because the test tool module is independent from the main extension module.

### 2. Send your instructions

The scaffold builds a JSON payload and sends it through the contract:

```go
payload, _ := json.Marshal(map[string]interface{}{
    "name": "World",
})
instructionId, _, err := instrutils.SendInstruction(s, addr, payload)
```

Replace the payload with whatever your contract function expects. If your Solidity contract has multiple send functions, you'll need to add corresponding Go helpers in `tools/pkg/utils/instructions.go` and call them here.

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

The generic status checks are already in `verifyResult`. The scaffold validates the SAY_HELLO response like this:

```go
var resp SayHelloResponse
err = json.Unmarshal(actionResult.Data, &resp)
if err != nil {
    return errors.Errorf("failed to unmarshal response: %s", err)
}

if resp.Greeting == "" {
    return errors.New("expected non-empty Greeting")
}
if resp.GreetingNumber < 1 {
    return errors.Errorf("expected GreetingNumber >= 1, got %d", resp.GreetingNumber)
}
```

Replace the response type, unmarshal target, and field assertions with your own.

### 4. Add more test cases

The scaffold shows a single send+verify pair. For a real extension, add multiple test cases covering:

- Each op type your extension supports
- Success cases with valid inputs
- Edge cases (empty fields, boundary values)
- Error cases (invalid payloads that should return `status == 0`)

### Matching op types between Solidity and Go

Your Solidity contract defines op types as `bytes32` constants:

```solidity
bytes32 constant OP_TYPE_SAY_HELLO = bytes32("SAY_HELLO");
```

Your Go extension's `processAction` routes on the same value:

```go
case dataFixed.OPType == teeutils.ToHash(config.OPTypeSayHello):
    return e.processSayHello(action, dataFixed)
```

The test sends instructions through the contract function that uses that op type (`sendSayHello`), and verifies the response matches what `processSayHello` returns.

## What you need to change (summary)

| Step | What to change | Where |
|------|---------------|-------|
| Response types | Define structs for your extension's responses | `tools/cmd/run-test/main.go` (top of file) |
| Message payloads | Create the JSON your contract function expects | `main()` in `run-test/main.go` |
| Send instructions | Call your contract's specific function(s) | `main()` in `run-test/main.go` |
| Validate responses | Unmarshal `Data` and assert your fields | `verifyResult()` in `run-test/main.go` |
| Add test scenarios | Add more send+verify pairs for each op type | `main()` in `run-test/main.go` |
