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

The scaffold's test sends instructions via `SendSayHello` and `SendSayGoodbye` and verifies the responses. Here's how each part works — when you build your own extension, you'll replace these with your own types, payloads, and assertions.

### 1. Define your message and response types

The scaffold defines `SayHelloResponse` and `SayGoodbyeResponse` at the top of the test file, mirroring the types from `pkg/types/types.go`:

```go
type SayHelloResponse struct {
    Greeting       string `json:"greeting"`
    GreetingNumber int    `json:"greetingNumber"`
}

type SayGoodbyeResponse struct {
    Farewell       string `json:"farewell"`
    FarewellNumber int    `json:"farewellNumber"`
}
```

Replace these with structs matching your extension's response types. These are defined separately in the test file because the test tool module is independent from the main extension module.

### 2. Send your instructions

The scaffold builds JSON payloads and sends them through the contract. The scaffold includes two test cases:

```go
// SAY_HELLO test case
payload, _ := json.Marshal(map[string]interface{}{
    "name": "World",
})
instructionId, _, err := instrutils.SendSayHello(s, addr, payload)

// SAY_GOODBYE test case
payload, _ = json.Marshal(map[string]interface{}{
    "name": "World",
})
instructionId, _, err = instrutils.SendSayGoodbye(s, addr, payload)
```

Replace the payloads with whatever your contract functions expect. If your Solidity contract has multiple send functions, you'll need to add corresponding Go helpers in `tools/pkg/utils/instructions.go` and call them here.

### 3. Validate your responses

The `verifyHelloResult` and `verifyGoodbyeResult` functions each receive the raw response from the proxy. The response envelope is always the same:

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

The generic status checks are already in `verifyHelloResult` and `verifyGoodbyeResult`. The scaffold validates the SAY_HELLO response like this:

```go
// verifyHelloResult
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

And the SAY_GOODBYE response like this:

```go
// verifyGoodbyeResult
var resp SayGoodbyeResponse
err = json.Unmarshal(actionResult.Data, &resp)
if err != nil {
    return errors.Errorf("failed to unmarshal response: %s", err)
}

if resp.Farewell == "" {
    return errors.New("expected non-empty Farewell")
}
if resp.FarewellNumber < 1 {
    return errors.Errorf("expected FarewellNumber >= 1, got %d", resp.FarewellNumber)
}
```

Replace the response types, unmarshal targets, and field assertions with your own.

### 4. Add more test cases

The scaffold shows two send+verify pairs (SAY_HELLO and SAY_GOODBYE). For a real extension, add multiple test cases covering:

- Each op type your extension supports
- Success cases with valid inputs
- Edge cases (empty fields, boundary values)
- Error cases (invalid payloads that should return `status == 0`)

### Matching op types between Solidity and Go

Your Solidity contract defines op types and op commands as `bytes32` constants:

```solidity
bytes32 constant OP_TYPE_GREETING  = bytes32("GREETING");
bytes32 constant OP_COMMAND_SAY_HELLO   = bytes32("SAY_HELLO");
bytes32 constant OP_COMMAND_SAY_GOODBYE = bytes32("SAY_GOODBYE");
```

Your Go extension's `processAction` routes on the same values:

```go
case dataFixed.OPType == teeutils.ToHash(config.OPTypeGreeting) &&
    dataFixed.OPCommand == teeutils.ToHash(config.OPCommandSayHello):
    return e.processSayHello(action, dataFixed)
case dataFixed.OPType == teeutils.ToHash(config.OPTypeGreeting) &&
    dataFixed.OPCommand == teeutils.ToHash(config.OPCommandSayGoodbye):
    return e.processSayGoodbye(action, dataFixed)
```

The test sends instructions through the contract functions (`sendSayHello`, `sendSayGoodbye`) which set the OPType to `GREETING` and the corresponding OPCommand, then verifies the response matches what `processSayHello` or `processSayGoodbye` returns.

## What you need to change (summary)

| Step | What to change | Where |
|------|---------------|-------|
| Response types | Define structs for your extension's responses | `tools/cmd/run-test/main.go` (top of file) |
| Message payloads | Create the JSON your contract function expects | `main()` in `run-test/main.go` |
| Send instructions | Call your contract's specific function(s) (e.g. `SendSayHello`, `SendSayGoodbye`) | `main()` in `run-test/main.go` |
| Validate responses | Unmarshal `Data` and assert your fields | `verifyHelloResult()` / `verifyGoodbyeResult()` in `run-test/main.go` |
| Op type + command routing | Match `OPTypeGreeting` + `OPCommandSayHello` / `OPCommandSayGoodbye` | `internal/config/config.go` and `processAction` |
| Add test scenarios | Add more send+verify pairs for each op command | `main()` in `run-test/main.go` |
