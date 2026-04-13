# Extension Registration Hardening — Design Spec

**Date:** 2026-04-09
**Scope:** Harden the extension registration pipeline (Step 2) against edge cases R1–R8 documented in `EXTENSION-DEPLOYMENT-EDGE-CASES.md`. Extends the verification tool and skill from the Step 1 hardening.

**Edge cases reference:** `EXTENSION-DEPLOYMENT-EDGE-CASES.md` — STEP 2: Extension Registration (R1–R8)
**Depends on:** `2026-04-09-deployment-hardening-design.md` (Step 1 hardening — already implemented)

---

## Goals

1. **Make `SetupExtension()` fully idempotent.** Re-running after a partial failure resumes from where it left off, logging "already done, skipping" for completed steps.
2. **Prevent silent data corruption.** Validate event parsing outputs (extension ID non-zero, receipt log length) so wrong IDs are never silently propagated.
3. **Extend the verification tool** with `--step register` checks for deep post-registration diagnostics (R1–R7).
4. **Update the Claude Code skill** with R1–R8 interpretation guidance.

**Audience:** Developer-grade error messages (same as Step 1).

---

## Track 1: Harden `SetupExtension()` and `register-extension`

### 1.1 Idempotent SetupExtension()

Each of the 4 sub-steps becomes check-before-act:

**Step 1 — Register Extension:**

New helper `findExistingExtension(s *Support, instructionsSenderAddress common.Address) (*big.Int, error)`:
- Calls `s.TeeExtensionRegistry.ExtensionsCounter(callOpts)` to get count
- Loops `0..count-1`, calling `s.TeeExtensionRegistry.GetTeeExtensionInstructionsSender(callOpts, i)` for each
- If match found, returns the extension ID
- If no match, returns nil

In `SetupExtension()`:
```go
extensionID, err := findExistingExtension(s, instructionsSenderAddress)
if err != nil {
    return nil, err
}
if extensionID != nil {
    logger.Infof("Extension already registered with ID %s for instruction sender %s, skipping registration",
        extensionID.String(), instructionsSenderAddress.Hex())
} else {
    extRegistered, _, err := registerExtension(s, opts, instructionsSenderAddress, stateVerifierAddress)
    if err != nil {
        return nil, err
    }
    extensionID = extRegistered.ExtensionId
    logger.Infof("Extension registered with ID: %s", extensionID.String())
}
```

**Step 2 — Allow TEE Machine Owners:**

```go
deployerAddr := crypto.PubkeyToAddress(s.Prv.PublicKey)
isOwner, err := s.TeeOwnerAllowlist.IsAllowedTeeMachineOwner(callOpts, extensionID, deployerAddr)
if err != nil {
    return nil, err
}
if isOwner {
    logger.Infof("Deployer %s already allowed as TEE machine owner for extension %s, skipping",
        deployerAddr.Hex(), extensionID.String())
} else {
    _, err = allowTeeMachineOwners(s, opts, extensionID, []common.Address{deployerAddr})
    if err != nil {
        return nil, errors.Errorf("failed adding TEE machine owners (extension exists as ID %s but owners not set): %s",
            extensionID.String(), err)
    }
    logger.Infof("TEE machine owners allowed for extension %s", extensionID.String())
}
```

**Step 3 — Allow Wallet Project Owners:**

Same pattern as Step 2 — use `s.TeeOwnerAllowlist.IsAllowedTeeWalletProjectOwner(callOpts, extensionID, deployerAddr)` to check before calling `AddAllowedTeeWalletProjectOwners`. Log "already allowed as wallet project owner, skipping" if true.

**Step 4 — Add Supported Key Types:**

Change the current error return to a skip log:
```go
if isKeyTypeSupported {
    logger.Infof("EVM key type already supported for extension %s, skipping", extensionID.String())
    // Continue instead of returning error
} else {
    logger.Infof("Adding key type %s to extension %s", wallets.EVMType, extensionID)
    err = AddSupportedKeyTypes(s, extensionID, []common.Hash{wallets.EVMType})
    if err != nil {
        return nil, err
    }
}
```

### 1.2 Event Parsing Validation

**Log length check in `registerExtension()`:**

After `bind.WaitMined`, before accessing `receipt.Logs`:
```go
if receipt.Status != 1 {
    return nil, nil, errors.Errorf("Register() transaction failed (receipt status %d)", receipt.Status)
}
if len(receipt.Logs) < 2 {
    return nil, nil, errors.Errorf(
        "expected at least 2 logs from Register() transaction, got %d — "+
            "the registry contract may have changed or be behind a proxy",
        len(receipt.Logs),
    )
}
```

**Extension ID non-zero warning:**

After parsing `TeeExtensionRegistered` event:
```go
if extensionRegistered.ExtensionId == nil || extensionRegistered.ExtensionId.Sign() == 0 {
    logger.Warnf("WARNING: extension ID is 0 — this may cause issues with setExtensionId() sentinel logic")
}
```

**Separate status check from parse error in all helpers:**

Current pattern (confusing):
```go
if err != nil || receipt.Status != 1 {
    return nil, errors.Errorf("error %s, or receipt status not 1", err)
}
```

Fixed pattern (clear):
```go
if receipt.Status != 1 {
    return nil, errors.Errorf("transaction failed (receipt status %d)", receipt.Status)
}
if len(receipt.Logs) == 0 {
    return nil, errors.Errorf("no logs in transaction — unexpected")
}
parsed, err := s.TeeOwnerAllowlist.ParseAllowedTeeMachineOwnersAdded(*receipt.Logs[0])
if err != nil {
    return nil, errors.Errorf("failed to parse event: %s", err)
}
```

Apply this pattern to:
- `registerExtension()` (lines 113-123)
- `allowTeeMachineOwners()` (lines 137-140)
- `allowTeeProjectManagerOwners()` (lines 155-160)

### 1.3 Harden `register-extension/main.go`

Add pre-flight validation after creating `DefaultSupport`:

```go
if err := validate.AddressHasCode(testSupport.ChainClient, instructionSenderAddress, "InstructionSender"); err != nil {
    fccutils.FatalWithCause(err)
}
```

After `SetupExtension()` returns, validate extension ID:
```go
if extensionID == nil || extensionID.Sign() <= 0 {
    logger.Warnf("WARNING: extension ID is %v — verify this is expected", extensionID)
}
```

---

## Track 2: Extend `verify-deploy` with `--step register`

### 2.1 New Check Function

```go
func RegisterRegistrationChecks(
    r *Report,
    client *ethclient.Client,
    key *ecdsa.PrivateKey,
    registry *teeextensionregistry.TeeExtensionRegistry,
    allowlist *teeownerallowlist.TeeOwnerAllowlist,
    extensionEnvPath string,
)
```

Reads `EXTENSION_ID` and `INSTRUCTION_SENDER` from `extensionEnvPath`. If the file doesn't exist, all checks SKIP with "run pre-build.sh first".

### 2.2 Check Table

| Check ID | What it verifies | Status on Failure | Fix Message |
|----------|-----------------|-------------------|-------------|
| R1 | `extensionsCounter()` > 0 | WARN | "registry has 0 extensions — first registration will get ID 0 which may conflict with sentinel logic" |
| R2 | On-chain instruction sender for this extension ID matches INSTRUCTION_SENDER in extension.env | FAIL | "extension ID X has instruction sender 0xABC on-chain, but extension.env says 0xDEF — config is stale or wrong contract deployed" |
| R3 | Deployer is in allowed TEE machine owners for this extension | FAIL | "deployer 0x... is not an allowed TEE machine owner for extension X. Re-run pre-build.sh (it will skip completed steps)" |
| R4 | EVM key type is supported for this extension | FAIL | "EVM key type not supported for extension X. Re-run pre-build.sh" |
| R5 | Composite: extension exists + owners set + key type set | WARN | "extension X is partially configured — re-run pre-build.sh to complete setup" |
| R7 | INSTRUCTION_SENDER is not registered for multiple extensions | WARN | "instruction sender 0x... registered for multiple extensions (IDs: X, Y). setExtensionId() will always resolve to first (ID X)" |

### 2.3 Wire Up in `verify-deploy/main.go`

The `verify-deploy` CLI currently creates `support.DefaultSupport()` which gives us `s.TeeExtensionRegistry` and `s.TeeOwnerAllowlist`. Pass these through:

```go
if runStep("register") {
    validate.RegisterRegistrationChecks(report, s.ChainClient, s.Prv,
        s.TeeExtensionRegistry, s.TeeOwnerAllowlist, *configPath)
}
```

Replaces the current `RegisterStubChecks(report, "register", "R1-R8", ...)` call.

### 2.4 Update `/verify-deploy` Skill

Add R1-R8 interpretation to the skill's "Interpreting Check Results" section:

- **R1** (extensions counter is 0): Registry is empty. If this is a fresh deployment that's expected. If not, wrong ADDRESSES_FILE or CHAIN_URL.
- **R2** (instruction sender mismatch): extension.env is stale — deployed contract doesn't match what's registered. Delete extension.env and re-run pre-build.sh.
- **R3** (deployer not allowed owner): Registration partially failed. Re-run pre-build.sh — it's now idempotent and will skip completed steps.
- **R4** (key type not supported): Same — partial failure, re-run pre-build.sh.
- **R5** (partially configured): Composite detection of R3/R4. Re-run pre-build.sh.
- **R7** (duplicate instruction sender): The instruction sender contract was registered more than once. setExtensionId() will always pick the first one. If the second was intended, deploy a new instruction sender contract.

---

## Files Changed / Created

**Modified:**
- `tools/pkg/fccutils/extension.go` — idempotent `SetupExtension()`, `findExistingExtension()` helper, event validation, separated error checks
- `tools/cmd/register-extension/main.go` — pre-flight `AddressHasCode` check, extension ID warning
- `tools/pkg/validate/checks.go` — new `RegisterRegistrationChecks()` function (replaces stub)
- `tools/pkg/validate/checks_test.go` — tests for new registration checks
- `tools/cmd/verify-deploy/main.go` — wire up registration checks with registry/allowlist contracts
- `.claude/skills/verify-deploy/SKILL.md` — add R1-R8 interpretation

**No new files** — everything extends existing infrastructure from Step 1.

---

## Out of Scope

- R1 contract fix (`setExtensionId` / `_getExtensionId` sentinel value) — extension ID 0 won't occur in practice. Defensive warning added.
- R8 (zero state verifier) — intentional for scaffold, low severity.
- Event parsing by topic hash scan — contracts won't be upgraded, unnecessary complexity.
- Changes to `pre-build.sh` — the script doesn't need modification for Step 2 (it already calls `register-extension` correctly, and the Go command handles idempotency internally).
