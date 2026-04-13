# Deployment Hardening — Comprehensive Test Plan

**Goal:** Verify that all deployment hardening (Steps 1–6) works correctly, especially revert reason decoding, smart contract error handling, idempotent registration, state file resume, and edge case detection — without breaking existing functionality.

**Approach:** Three layers of testing:
1. **Unit tests** — Pure Go tests for isolated functions (revert decoding, state file I/O, env parsing, report formatting)
2. **On-chain integration tests** — Against a local Hardhat/Anvil node to verify revert decoding, contract validation, and idempotent flows actually work
3. **Script/CLI smoke tests** — Bash-level tests ensuring scripts handle errors correctly

---

## Layer 1: Unit Tests (No Chain Required)

### 1.1 Revert Reason Decoding (`tools/pkg/fccutils/revert_test.go`)

These test the pure decoding logic in `revert.go` without needing an RPC node.

**Test: `TestDecodeRevertHex_StandardError`**
- Input: ABI-encoded `Error(string)` with message "Extension ID already set."
- Encode: `abi.PackRevert("Extension ID already set.")`, hex-encode
- Expected: returns `"Extension ID already set."`

**Test: `TestDecodeRevertHex_WithPrefix`**
- Same as above but with `0x` prefix
- Expected: same decoded string

**Test: `TestDecodeRevertHex_CustomError`**
- Input: 4-byte selector + encoded data (not Error(string))
- Expected: returns `"0x" + rawHex` (falls back to hex)

**Test: `TestDecodeRevertHex_TooShort`**
- Input: "abcd" (2 bytes, not 4)
- Expected: returns `""` (too short to be a valid revert)

**Test: `TestDecodeRevertHex_Empty`**
- Input: ""
- Expected: returns `""` 

**Test: `TestDecodeRevertHex_InvalidHex`**
- Input: "not-valid-hex"
- Expected: returns `""`

**Test: `TestDecodeRevertReason_NilError`**
- Input: `nil`
- Expected: returns `""`

**Test: `TestDecodeRevertReason_PlainError`**
- Input: `errors.New("something")`
- Expected: returns `""` (no ErrorData interface)

**Test: `TestDecodeRevertReason_WithDataError`**
- Create a mock error implementing `ErrorData() interface{}`
- Return ABI-encoded revert hex string from `ErrorData()`
- Expected: returns decoded reason string

**Test: `TestDecodeRevertReason_WithNilData`**
- Mock error where `ErrorData()` returns nil
- Expected: returns `""`

**Test: `TestDecodeRevertReason_WithNonStringData`**
- Mock error where `ErrorData()` returns `42` (int, not string)
- Expected: returns `""`

### 1.2 Revert Decoding in support.go (`tools/pkg/support/support_test.go`)

**Test: `TestDecodeRevertFromError_StandardRevert`**
- Create a mock error implementing `ErrorData() interface{}` 
- Set ErrorData to return hex-encoded `Error(string)` ABI data
- Expected: returns the decoded reason

**Test: `TestDecodeRevertFromError_NoInterface`**
- Input: `errors.New("plain error")`
- Expected: returns `""`

**Test: `TestDecodeRevertFromError_InvalidHex`**
- Mock error with ErrorData returning "zzzz"
- Expected: returns `""` (hex decode fails)

**Test: `TestDecodeRevertFromError_ShortData`**
- Mock error with ErrorData returning "0x1234" (2 bytes)
- Expected: returns `""` (too short for UnpackRevert)

### 1.3 State File I/O (`tools/pkg/fccutils/registration_test.go`)

**Test: `TestLoadState_NoFile`**
- Path: non-existent file
- Expected: returns empty `registrationState{}`, no error

**Test: `TestLoadState_ValidFile`**
- Write JSON: `{"completed_steps":"r","tee_attest_instruction_id":"0xabc..."}`
- Expected: `CompletedSteps == "r"`, instruction ID matches

**Test: `TestLoadState_InvalidJSON`**
- Write: `{invalid json`
- Expected: returns error mentioning "parse state file"

**Test: `TestLoadState_EmptyFile`**
- Write: ``
- Expected: returns error (empty JSON is invalid)

**Test: `TestSaveState_WritesCorrectly`**
- Save state with `CompletedSteps: "rRa"` and instruction IDs
- Read back file, unmarshal, verify fields match

**Test: `TestSaveState_OverwritesExisting`**
- Save state with "r", then save again with "rR"
- Read back: should see "rR" (not "rrR")

**Test: `TestSaveState_ReadOnlyDir`**
- Attempt to save to a read-only directory
- Expected: returns error mentioning "write state file"

### 1.4 Extension Env Parsing (already partially tested, add coverage)

**Test: `TestParseExtensionEnv_CommentsOnly`**
- File with only comments and blank lines
- Expected: both return empty strings, no error

**Test: `TestParseExtensionEnv_ExtraKeys`**
- File with EXTENSION_ID, INSTRUCTION_SENDER, and EXTRA_KEY
- Expected: parses the two known keys, ignores EXTRA_KEY

**Test: `TestParseExtensionEnv_NoEquals`**
- File with line "EXTENSION_ID_NO_EQUALS"
- Expected: skips line, returns empty for EXTENSION_ID

**Test: `TestCheckExtensionEnvFormat_UpperLowerCase`**
- INSTRUCTION_SENDER with lowercase hex `0xabcdef...` (valid)
- Expected: PASS

**Test: `TestCheckExtensionEnvFormat_MixedCase`**
- EIP-55 checksum address format
- Expected: PASS (regex `[0-9a-fA-F]` allows mixed case)

### 1.5 Report Formatting (already tested, verify edge cases)

**Test: `TestReport_EmptyReport`**
- No results added
- `Summary()` should return "0 passed, 0 warning, 0 failed, 0 skipped"
- `HasFailures()` should return false
- `PrintJSON()` should produce valid JSON with empty results array

**Test: `TestReport_AllStatuses`**
- Add one of each: PASS, WARN, FAIL, SKIP
- `HasFailures()` = true
- Summary: "1 passed, 1 warning, 1 failed, 1 skipped"

### 1.6 Validation Primitives (add edge case coverage to `tools/pkg/validate/validate_test.go`)

**Test: `TestAddressNotZero_PartialZero`**
- Address: `0x0000000000000000000000000000000000000001`
- Expected: no error (not zero)

**Test: `TestKeyHasFunds_NilKey`**
- Nil private key, non-nil client
- Expected: error or panic protection (verify current behavior)

---

## Layer 2: On-Chain Integration Tests (Hardhat/Anvil)

These tests require a running local Ethereum node. They verify the full flow including RPC calls, transaction submission, and revert reason extraction.

### Prerequisites
- Hardhat or Anvil running on `http://127.0.0.1:8545`
- Deployed registry contracts (from `deployed-addresses.json`)
- Funded deployer account (Hardhat default key is fine for local)

### 2.1 Contract Constructor Validation (`tools/integration_test/constructor_test.go`)

**Test: `TestDeploy_ZeroRegistryAddress`**
- Deploy `InstructionSender` with zero address for `_teeExtensionRegistry`
- Expected: transaction reverts with "TeeExtensionRegistry cannot be zero address"
- Verify: decoded revert reason matches exactly (tests full revert decoding chain)

**Test: `TestDeploy_ZeroMachineRegistryAddress`**
- Deploy with valid extension registry but zero machine registry
- Expected: reverts with "TeeMachineRegistry cannot be zero address"

**Test: `TestDeploy_EOAAsRegistry`**
- Deploy with a regular EOA (externally owned account) address as registry
- Expected: reverts with "TeeExtensionRegistry has no code"
- This tests the `code.length > 0` check

**Test: `TestDeploy_ValidAddresses`**
- Deploy with correct registry addresses from deployed-addresses.json
- Expected: succeeds, returns valid contract address

### 2.2 setExtensionId Error Handling

**Test: `TestSetExtensionId_NotRegistered`**
- Deploy a fresh InstructionSender (valid addresses)
- Call `setExtensionId()` WITHOUT registering the extension first
- Expected: reverts with "Extension ID not found."
- Verify: Go code (`instrutils.SetExtensionId`) returns error containing this message

**Test: `TestSetExtensionId_AlreadySet`**
- Deploy, register extension, call `setExtensionId()` once (success)
- Call `setExtensionId()` again
- Expected: reverts with "Extension ID already set."
- Verify: Go `run-test` logic catches "already set" and continues

**Test: `TestSetExtensionId_RevertReasonDecoded`**
- Same as above, but specifically verify that:
  - `DecodeRevertReason(err)` returns the revert string, OR
  - `SimulateAndDecodeRevert()` returns it as fallback
- This is the KEY test for the revert decoding fix

### 2.3 Instruction Sending Error Handling

**Test: `TestSendSayHello_NoExtensionId`**
- Deploy InstructionSender but don't call `setExtensionId()`
- Call `sendSayHello()` with valid message
- Expected: reverts with "Extension ID is not set."
- Verify: error message is decoded correctly (not garbled binary)

**Test: `TestSendSayHello_InsufficientFee`**
- Deploy, register, set extension ID
- Call `sendSayHello()` with `value: 0` (no fee)
- Expected: reverts (registry requires fee)
- Verify: revert reason mentions payment/fee issue

**Test: `TestSendSayHello_Success`**
- Full happy path: deploy, register, set ID, send with correct fee
- Expected: transaction succeeds
- Verify: `receipt.Logs` has at least 1 log, `TeeInstructionsSent` event parsed correctly

### 2.4 CheckTx Revert Reason Chain (`tools/integration_test/checktx_test.go`)

**Test: `TestCheckTx_SuccessfulTx`**
- Submit a simple successful transaction
- Expected: returns receipt with Status == 1

**Test: `TestCheckTx_FailedTx_DecodesReason`**
- Submit a transaction that will fail (e.g., call setExtensionId on unregistered contract)
- Expected: error message contains the revert reason, NOT hex garbage
- This is the CRITICAL test — verifies the fix from garbled `string(rawBytes)` to `abi.UnpackRevert()`

**Test: `TestCheckTx_FailedTx_FallbackToHex`**
- Submit a transaction that fails with a custom error (no Error(string) selector)
- Expected: error contains `0x` hex representation (not binary garbage)

### 2.5 Idempotent Registration Flow (`tools/integration_test/registration_test.go`)

**Test: `TestSetupExtension_FirstTime`**
- Call `SetupExtension()` on a clean chain
- Expected: registers extension, returns valid ID, logs registration

**Test: `TestSetupExtension_AlreadyRegistered`**
- Call `SetupExtension()` twice with same instruction sender
- Expected: second call returns same ID, logs "already registered, skipping"
- Verify: no duplicate transactions submitted

**Test: `TestSetupExtension_PartialFailure_OwnersNotSet`**
- Register extension but simulate owner step failure (hard to do without mocking)
- Alternative: verify error message mentions "extension exists as ID X but owners not set"

**Test: `TestFindExistingExtension_NoExtensions`**
- Fresh chain with no extensions
- Expected: returns nil, nil

**Test: `TestFindExistingExtension_WrongSender`**
- Register an extension, then search for a different sender address
- Expected: returns nil, nil

**Test: `TestFindExistingExtension_MatchFound`**
- Register an extension, then search for its sender
- Expected: returns the correct extension ID

### 2.6 Pre-Flight Validation Against Chain

**Test: `TestAddressHasCode_DeployedContract`**
- Check registry address from deployed-addresses.json
- Expected: no error (contract has code)

**Test: `TestAddressHasCode_RandomEOA`**
- Check a random generated address
- Expected: error mentioning "no code"

**Test: `TestKeyHasFunds_FundedAccount`**
- Hardhat default account (10000 ETH)
- Expected: no error

**Test: `TestKeyHasFunds_EmptyAccount`**
- Generate a random private key (unfunded)
- Expected: error mentioning insufficient balance, showing actual (0) vs required

---

## Layer 3: Script & CLI Smoke Tests

These verify that scripts and commands handle error conditions correctly.

### 3.1 deploy-contract Pre-flight (`scripts/test-hardening/test_deploy_preflight.sh`)

**Test: Preflight-only mode**
```bash
cd tools && go run ./cmd/deploy-contract --preflight-only \
  -a <addresses-file> -c http://127.0.0.1:8545
# Expected: exits 0, prints "Pre-flight checks passed"
```

**Test: Wrong chain URL**
```bash
cd tools && go run ./cmd/deploy-contract --preflight-only \
  -a <addresses-file> -c http://127.0.0.1:9999
# Expected: exits 1, error about connection refused
```

### 3.2 verify-deploy All Steps

**Test: All checks with valid config**
```bash
cd tools && go run ./cmd/verify-deploy \
  -a <addresses-file> -c http://127.0.0.1:8545 \
  --config ../config/extension.env
# Expected: exits 0 or 1 depending on state, but produces readable output
```

**Test: JSON output format**
```bash
cd tools && go run ./cmd/verify-deploy \
  -a <addresses-file> -c http://127.0.0.1:8545 \
  --json | python3 -m json.tool
# Expected: valid JSON, parseable by python
```

**Test: Single step mode**
```bash
cd tools && go run ./cmd/verify-deploy \
  -a <addresses-file> -c http://127.0.0.1:8545 \
  --step deploy
# Expected: only shows deploy-step checks (D1-D7)
```

**Test: Invalid step name**
```bash
cd tools && go run ./cmd/verify-deploy --step invalid
# Expected: exits 1, "Unknown step" error with valid step list
```

### 3.3 Script Error Handling

**Test: pre-build.sh regex validation**
```bash
# Simulate: deploy-contract returns invalid address
# (Requires modifying output temporarily or using a mock)
# Expected: script dies with "invalid INSTRUCTION_SENDER format"
```

**Test: post-build.sh SIMULATED_TEE export**
```bash
bash -n scripts/post-build.sh  # Syntax check
grep "SIMULATED_TEE" scripts/post-build.sh  # Should find export
```

### 3.4 State File Resume

**Test: State file created on partial success**
```bash
# After register-tee completes step "r" but fails on "a":
cat config/register-tee.state
# Expected: {"completed_steps":"r","tee_attest_instruction_id":"0x..."}
```

**Test: Re-run resumes from state**
```bash
# Run register-tee again
# Expected: logs "Pre-registration already completed, skipping (from state file)"
```

---

## Layer 4: Extension Server Tests

### 4.1 OPType/OPCommand Hash Debug Info

**Test: `TestProcessAction_UnknownOPType`**
- Send an action with an unrecognized OPType hash
- Expected: 501 response containing:
  - "unsupported op type"
  - Received hash hex
  - Expected hash hex
  - Expected hash name ("GREETING")

**Test: `TestProcessAction_UnknownOPCommand`**
- Send an action with correct OPType (GREETING) but unknown OPCommand
- Expected: 501 response containing:
  - "unsupported op command"
  - Received hash hex
  - Expected hashes for SAY_HELLO and SAY_GOODBYE with names

**Test: `TestProcessAction_ValidGreeting`**
- Send a valid SAY_HELLO action
- Expected: 200 response with valid ActionResult, greeting in data

**Test: `TestProcessSayHello_EmptyName`**
- Send SAY_HELLO with `{"name": ""}`
- Expected: status 0 with error "name must not be empty"

**Test: `TestProcessSayHello_InvalidJSON`**
- Send SAY_HELLO with `{invalid json`
- Expected: status 0 with error about decoding

---

## Execution Priority

### Must-Test (Critical Path)
1. **2.2 setExtensionId revert decoding** — This was broken before. Must confirm fix.
2. **2.4 CheckTx revert reason chain** — Core fix. Garbled output → readable errors.
3. **1.1 decodeRevertHex all variants** — Foundation for all revert decoding.
4. **2.5 Idempotent registration** — Must not double-register or fail on re-run.
5. **1.3 State file I/O** — Resume support must work correctly.

### Should-Test (Important)
6. **2.1 Constructor validation** — Contract rejects bad addresses.
7. **2.3 Instruction sending errors** — Fee and ID errors decoded properly.
8. **4.1 Extension 501 responses** — Hash mismatch provides useful debug info.
9. **1.4 Env parsing edge cases** — Prevent silent misconfiguration.
10. **3.2 verify-deploy all modes** — CLI tool produces correct output.

### Nice-to-Have (Regression Safety)
11. **2.6 Pre-flight validation** — AddressHasCode and KeyHasFunds against real chain.
12. **3.1 deploy-contract preflight** — CLI flag works.
13. **1.5 Report formatting** — Edge cases in output.
14. **3.3 Script validation** — Bash-level error handling.

---

## Test Infrastructure Needed

### For Layer 1 (unit tests):
- No new dependencies needed
- Mock `dataError` interface for revert decoding tests
- Use `t.TempDir()` for state file tests
- Unexport `decodeRevertHex`, `loadState`, `saveState` test via same-package tests

### For Layer 2 (integration tests):
- Local Hardhat/Anvil node running
- New directory: `tools/integration_test/` or use build tags (`//go:build integration`)
- Reuse existing `support.DefaultSupport()` for chain connection
- Contract deployment helpers from `tools/pkg/utils/`

### For Layer 3 (script tests):
- Hardhat/Anvil running
- Either dedicated test script or manual verification checklist

### For Layer 4 (extension tests):
- Direct Go tests in `internal/extension/` — no external services needed
- Construct `teetypes.Action` structs with specific OPType/OPCommand hashes
- Test `processAction()` directly (it's a method on `Extension`)

---

## Success Criteria

- All Layer 1 tests pass: `cd tools && go test ./pkg/fccutils/ ./pkg/validate/ ./pkg/support/ -v`
- All Layer 2 tests pass: `cd tools && go test -tags integration ./integration_test/ -v` (with local node)
- Layer 4 tests pass: `go test ./internal/extension/ -v`
- Existing tests still pass: `cd tools && go test ./... -v`
- All commands build: `cd tools && go build ./cmd/...`
- Full extension builds: `go build ./cmd/docker/`
- Scripts are syntactically valid: `bash -n scripts/*.sh`
