# Deployment Hardening & Verification Tool — Design Spec

**Date:** 2026-04-09
**Scope:** Harden the contract deployment pipeline (Step 1) against the edge cases documented in `EXTENSION-DEPLOYMENT-EDGE-CASES.md`. Design verification architecture for all 6 steps, implement Step 1.

**Edge cases reference:** `EXTENSION-DEPLOYMENT-EDGE-CASES.md` — STEP 1: Contract Deployment (D1–D8)

---

## Goals

1. **Silent failures become loud failures.** Every deployment error surfaces a clear, actionable message with the root cause and fix.
2. **Pre-flight validation catches mistakes before on-chain mutations.** Bad addresses, unfunded keys, wrong networks — caught before gas is spent.
3. **A standalone verification tool** provides comprehensive diagnostics across the full deployment lifecycle, usable by the end client to self-diagnose.
4. **A Claude Code skill** wraps the verification tool with interpretive guidance, cross-referencing the edge cases doc and deployment logs.

**Audience:** Developer-grade — errors reference addresses, revert reasons, and chain state directly.

---

## Architecture: Three Layers

```
Layer 1: Script Hardening (pre-build.sh)
  └─ Catches: missing env vars, malformed outputs, suppressed errors

Layer 2: Go Command Hardening (deploy-contract, register-extension)
  └─ Catches: zero/EOA addresses, unfunded keys, wrong chain, timeouts

Layer 3: Standalone Verification (cmd/verify-deploy) + Claude Code Skill
  └─ Catches: everything above + cross-step consistency, stale config, network mismatches
```

Each layer is independent — if one is bypassed (e.g., deploying via Foundry skips Layers 1+2), the others still provide protection.

---

## Layer 1: Script Hardening (`scripts/pre-build.sh`)

### 1.1 Stop Suppressing Stderr

**Current (lines 79, 84):**
```bash
INSTRUCTION_SENDER=$(go run ./cmd/deploy-contract -a "$ADDRESSES_FILE" -c "$CHAIN_URL" 2>/dev/null | tail -1)
EXTENSION_ID=$(go run ./cmd/register-extension ... 2>/dev/null | tail -1)
```

**Change:** Redirect stderr to `config/deploy.log`, display on failure:
```bash
LOG_FILE="$PROJECT_DIR/config/deploy.log"

INSTRUCTION_SENDER=$(go run ./cmd/deploy-contract -a "$ADDRESSES_FILE" -c "$CHAIN_URL" 2>"$LOG_FILE" | tail -1) || {
    echo -e "${RED}Deploy failed. Logs:${NC}" >&2
    cat "$LOG_FILE" >&2
    die "Deploy failed — see output above"
}
```

Same pattern for register-extension (append to same log file with `2>>"$LOG_FILE"`).

### 1.2 Validate Captured Values

After each capture, validate the format:
```bash
[[ "$INSTRUCTION_SENDER" =~ ^0x[0-9a-fA-F]{40}$ ]] || \
    die "deploy-contract returned invalid address: '$INSTRUCTION_SENDER' (expected 0x + 40 hex chars). Check $LOG_FILE"

[[ "$EXTENSION_ID" =~ ^0x[0-9a-fA-F]{64}$ ]] || \
    die "register-extension returned invalid ID: '$EXTENSION_ID'. Check $LOG_FILE"
```

### 1.3 Pre-flight Check Before Deploying

Add a Step 0 that calls `deploy-contract --preflight-only`:
```bash
step 0 "Pre-flight check"
go run ./cmd/deploy-contract -a "$ADDRESSES_FILE" -c "$CHAIN_URL" --preflight-only 2>&1 || die "Pre-flight check failed"
```

This prints deployer address, balance, chain ID, and validates registry addresses — then exits without deploying.

**Edge cases addressed:** D5 (key fallback hidden), D7 (tail -1 captures garbage), D8 (partial — balance check).

---

## Layer 2: Go Command Hardening

### 2.1 Shared Validation Package (`tools/pkg/validate/`)

New package with reusable check primitives:

**`tools/pkg/validate/validate.go`:**

```go
package validate

// AddressHasCode checks that the address has deployed bytecode.
// Returns a descriptive error naming the address and label on failure.
func AddressHasCode(client *ethclient.Client, addr common.Address, label string) error

// KeyHasFunds checks that the key's account has at least minWei balance.
// Error includes the address and current balance.
func KeyHasFunds(client *ethclient.Client, key *ecdsa.PrivateKey, minWei *big.Int) error

// AddressNotZero checks that the address is not 0x000...000.
func AddressNotZero(addr common.Address, label string) error

// ChainIDMatches checks the connected chain's ID. Used for informational logging,
// not hard failures (the user may intentionally target any chain).
func ChainIDMatches(client *ethclient.Client, expected *big.Int) error
```

`MinDeployBalance` is a package-level constant, suggested default `0.01 ETH` (enough for a contract deployment + registration on Coston2). Can be overridden by callers if needed.

Error messages are specific and actionable:
- `AddressHasCode`: `"TeeExtensionRegistry at 0xABC...123 has no deployed code. This address will be set as immutable in the contract constructor and cannot be changed. Check your deployed-addresses.json file."`
- `KeyHasFunds`: `"deployer 0xDEF...456 has 0 ETH on chain 114 (balance: 0 wei, minimum required: 100000000000000 wei). Fund this account before deploying."`
- `AddressNotZero`: `"TeeExtensionRegistry is the zero address (0x000...000) in the addresses file."`

### 2.2 Hardening `deploy-contract/main.go`

Add pre-flight validation before `DeployInstructionSender()`:

```go
// New --preflight-only flag
preflightOnly := flag.Bool("preflight-only", false, "run validation checks and exit without deploying")

// After creating Support:
deployer := crypto.PubkeyToAddress(testSupport.Prv.PublicKey)
logger.Infof("Deployer:             %s", deployer.Hex())
logger.Infof("Chain ID:             %s", testSupport.ChainID.String())
logger.Infof("TeeExtensionRegistry: %s", testSupport.Addresses.TeeExtensionRegistry.Hex())
logger.Infof("TeeMachineRegistry:   %s", testSupport.Addresses.TeeMachineRegistry.Hex())

// Validate constructor parameter addresses
if err := validate.AddressNotZero(testSupport.Addresses.TeeExtensionRegistry, "TeeExtensionRegistry"); err != nil {
    fccutils.FatalWithCause(err)
}
if err := validate.AddressNotZero(testSupport.Addresses.TeeMachineRegistry, "TeeMachineRegistry"); err != nil {
    fccutils.FatalWithCause(err)
}
if err := validate.AddressHasCode(testSupport.ChainClient, testSupport.Addresses.TeeExtensionRegistry, "TeeExtensionRegistry"); err != nil {
    fccutils.FatalWithCause(err)
}
if err := validate.AddressHasCode(testSupport.ChainClient, testSupport.Addresses.TeeMachineRegistry, "TeeMachineRegistry"); err != nil {
    fccutils.FatalWithCause(err)
}
if err := validate.KeyHasFunds(testSupport.ChainClient, testSupport.Prv, validate.MinDeployBalance); err != nil {
    fccutils.FatalWithCause(err)
}

if *preflightOnly {
    logger.Infof("Pre-flight checks passed. Exiting without deploying.")
    return
}
```

### 2.3 Fix `DefaultPrivateKey()` in `support.go`

Two changes:

1. **Remove private key leak** — delete `fmt.Printf("privKeyString: %s\n", privKeyString)` (line 93)
2. **Move fallback warning to stderr:**

```go
if privKeyString == "" {
    fmt.Fprintln(os.Stderr, "WARNING: PRIV_KEY not set — using hardcoded Hardhat dev key")
    fmt.Fprintln(os.Stderr, "         This key only has funds on local devnets (Hardhat/Anvil)")
    return configs.PrvWithFunds, nil
}
```

### 2.4 Add Timeout to `WaitMined`

In `support.go` `CheckTx()` and in `instructions.go` `DeployInstructionSender()`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
if err != nil {
    return nil, errors.Errorf("transaction not mined after 2 minutes — check chain RPC at %s: %s", chainURL, err)
}
```

**Edge cases addressed:** D1 (wrong addresses), D2 (EOAs), D3 (zero addresses), D5 (key leak + fallback), D6 (chain ID logged), D8 (timeout).

---

## Layer 3: Standalone Verification Tool

### 3.1 Report Types (`tools/pkg/validate/report.go`)

```go
type Status int
const (
    PASS Status = iota
    WARN
    FAIL
    SKIP
)

type CheckResult struct {
    Step    string // "deploy", "register", "services", "tee-version", "tee-machine", "test"
    ID      string // "D1", "D2", etc. — maps to edge cases doc
    Name    string // human-readable check name
    Status  Status
    Message string // what was found
    Fix     string // how to fix (only on FAIL/WARN)
}

type Report struct {
    Results []CheckResult
}

func (r *Report) Add(result CheckResult)
func (r *Report) Print()           // colored terminal output
func (r *Report) PrintJSON()       // JSON for programmatic use
func (r *Report) HasFailures() bool
func (r *Report) Summary() string  // "7 passed, 1 warning, 0 failed, 5 skipped"
```

### 3.2 CLI Interface (`tools/cmd/verify-deploy/main.go`)

```bash
# Check everything
go run ./cmd/verify-deploy -a deployed-addresses.json -c http://127.0.0.1:8545

# Check specific step
go run ./cmd/verify-deploy --step deploy

# Post-deployment check (also validates extension.env)
go run ./cmd/verify-deploy --config ../config/extension.env

# JSON output
go run ./cmd/verify-deploy --json
```

Flags:
- `-a` — addresses file path (same as deploy-contract)
- `-c` — chain URL (same as deploy-contract)
- `--step` — `deploy`, `register`, `services`, `tee-version`, `tee-machine`, `test`, or `all` (default)
- `--config` — path to `extension.env` for post-deployment validation
- `--json` — output as JSON instead of colored terminal

### 3.3 Step 1 Checks (Implemented)

Check registration follows a simple pattern per step:

```go
func RegisterDeployChecks(r *Report, s *support.Support, cfg *VerifyConfig) {
    r.Add(checkRegistryHasCode(s, "TeeExtensionRegistry", s.Addresses.TeeExtensionRegistry))
    r.Add(checkRegistryHasCode(s, "TeeMachineRegistry", s.Addresses.TeeMachineRegistry))
    r.Add(checkNotZeroAddress("TeeExtensionRegistry", s.Addresses.TeeExtensionRegistry))
    r.Add(checkNotZeroAddress("TeeMachineRegistry", s.Addresses.TeeMachineRegistry))
    r.Add(checkDeployerBalance(s))
    r.Add(checkDeployerKeySource())
    r.Add(checkChainID(s))
    if cfg.ExtensionEnvPath != "" {
        r.Add(checkExtensionEnvFormat(cfg.ExtensionEnvPath))
        r.Add(checkInstructionSenderExists(s, cfg.ExtensionEnvPath))
    }
}
```

Full check table:

| Check ID | Checks | Status on Failure | Fix Message |
|----------|--------|-------------------|-------------|
| D1 | `TeeExtensionRegistry` has code on-chain | FAIL | "address 0x... has no code — wrong network or wrong addresses file?" |
| D1 | `TeeMachineRegistry` has code on-chain | FAIL | Same |
| D2 | Both addresses are contracts (not EOAs) | FAIL | "address 0x... is an EOA, expected a contract" |
| D3 | Neither address is zero | FAIL | "TeeExtensionRegistry is 0x000...000 in addresses file" |
| D4 | If extension.env exists, INSTRUCTION_SENDER has code | WARN | "stale extension.env — INSTRUCTION_SENDER at 0x... has no code. Re-run pre-build.sh?" |
| D5 | `PRIV_KEY` env var is set | WARN (non-local) | "PRIV_KEY not set — using Hardhat dev key which has no funds on Coston2" |
| D5 | Deployer balance ≥ minimum | FAIL | "deployer 0x... has 0 ETH — fund this account" |
| D6 | Chain ID logged | INFO | "connected to chain ID 31337 (Hardhat)" |
| D7 | extension.env INSTRUCTION_SENDER format | FAIL | "INSTRUCTION_SENDER value is malformed: '...'" |
| D7 | extension.env EXTENSION_ID format | FAIL | "EXTENSION_ID value is malformed" |

### 3.4 Steps 2-6 Stubs (Designed, Not Implemented)

Each step has a `RegisterXxxChecks` function that starts with SKIP results:

```go
func RegisterRegisterChecks(r *Report, s *support.Support, cfg *VerifyConfig) {
    r.Add(CheckResult{Step: "register", ID: "R1-R8", Name: "Extension registration checks", Status: SKIP, Message: "not yet implemented"})
}
// Same for services, tee-version, tee-machine, test
```

Future implementation adds real checks without changing architecture.

### 3.5 Report Output Format

```
=== Deployment Verification Report ===

Step: Contract Deployment
  [PASS] D1  TeeExtensionRegistry has code at 0xABCD...1234
  [PASS] D1  TeeMachineRegistry has code at 0xEFGH...5678
  [PASS] D3  No zero addresses in config
  [WARN] D5  Using Hardhat dev key — only works on local devnet
  [PASS] D5  Deployer 0x1234...5678 has 100 ETH
  [PASS] D6  Chain ID 31337 (local devnet)
  [PASS] D7  extension.env values are well-formed

Step: Extension Registration
  [SKIP]     Not yet implemented

Step: Service Startup
  [SKIP]     Not yet implemented

Step: TEE Version Registration
  [SKIP]     Not yet implemented

Step: TEE Machine Registration
  [SKIP]     Not yet implemented

Step: Testing
  [SKIP]     Not yet implemented

Summary: 7 passed, 1 warning, 0 failed, 5 skipped
```

Exit code: 0 if no FAIL, 1 if any FAIL (enables CI integration).

---

## Claude Code Skill (`.claude/skills/verify-deploy/SKILL.md`)

### Purpose

Wraps `cmd/verify-deploy`, interprets results, suggests fixes, and cross-references the edge cases doc. Also reads deployment logs for historical failure diagnosis.

### When to Use

- User says "verify my deployment", "check my setup", "why is deploy failing"
- Before running pre-build.sh on a new network
- After a failed deployment to diagnose what went wrong
- `/verify-deploy`

### Behavior

1. Read `.env` and `config/extension.env` to determine deployment state
2. Run `cd tools && go run ./cmd/verify-deploy -a <ADDRESSES_FILE> -c <CHAIN_URL> --config ../config/extension.env`
3. For each FAIL/WARN, reference the edge case doc by ID and provide the specific fix
4. If all PASS, confirm deployment looks healthy

### Diagnosing Past Failures

If the user reports a failed deployment, or if verify-deploy shows FAILs:

1. Read `config/deploy.log` — this captures stderr from the Go deploy commands during `pre-build.sh`
2. Look for:
   - Revert reasons (indicates on-chain rejection — likely wrong addresses or permissions)
   - "insufficient funds" (wrong key or unfunded account — D5)
   - Connection errors (wrong CHAIN_URL — D6)
   - Panic/stack traces (bug in tooling)
   - "WARNING: PRIV_KEY not set" (using dev key on non-local network — D5)
3. Cross-reference the log content with check IDs from verification output
4. If `config/deploy.log` doesn't exist, the user hasn't run `pre-build.sh` yet — suggest running `verify-deploy --step deploy` as a preflight check first

### Interpreting Check Results

Each check ID maps to an edge case in `EXTENSION-DEPLOYMENT-EDGE-CASES.md`:

- **D1/D2** (registry has no code): Wrong CHAIN_URL or wrong ADDRESSES_FILE — they point to different networks
- **D3** (zero address): A required contract address is missing in the addresses file
- **D4** (stale extension.env): The INSTRUCTION_SENDER from a previous deploy no longer exists — re-run pre-build.sh
- **D5** (key/balance): Set PRIV_KEY in .env to a funded account on the target network
- **D6** (chain ID): Verify CHAIN_URL points to the intended network
- **D7** (malformed values): Go command printed unexpected output — delete extension.env and re-run pre-build.sh

### Cross-Check Patterns

The skill should detect compound misconfigurations:
- Coston2 addresses + localhost CHAIN_URL = network mismatch
- LOCAL_MODE=false + no PRIV_KEY = will use Hardhat key with no Coston2 funds
- extension.env exists + INSTRUCTION_SENDER has no code = stale deployment, re-run needed

---

## Solidity Constructor Hardening

Add validation to `contracts/InstructionSender.sol` constructor:

```solidity
constructor(
    ITeeExtensionRegistry _teeExtensionRegistry,
    ITeeMachineRegistry _teeMachineRegistry
) {
    require(address(_teeExtensionRegistry) != address(0), "TeeExtensionRegistry cannot be zero address");
    require(address(_teeMachineRegistry) != address(0), "TeeMachineRegistry cannot be zero address");
    require(address(_teeExtensionRegistry).code.length > 0, "TeeExtensionRegistry has no code");
    require(address(_teeMachineRegistry).code.length > 0, "TeeMachineRegistry has no code");
    TEE_EXTENSION_REGISTRY = _teeExtensionRegistry;
    TEE_MACHINE_REGISTRY = _teeMachineRegistry;
}
```

Catches D1, D2, D3 at the EVM level — last line of defense regardless of tooling.

---

## Files Changed / Created

**Modified:**
- `scripts/pre-build.sh` — stderr handling, regex validation, preflight step
- `tools/cmd/deploy-contract/main.go` — pre-flight checks, `--preflight-only` flag
- `tools/pkg/support/support.go` — remove key leak, stderr for warnings, WaitMined timeout
- `tools/pkg/utils/instructions.go` — WaitMined timeout
- `contracts/InstructionSender.sol` — constructor validation

**Created:**
- `tools/pkg/validate/validate.go` — shared validation primitives
- `tools/pkg/validate/report.go` — CheckResult type, report builder
- `tools/pkg/validate/checks.go` — composable check functions per step
- `tools/cmd/verify-deploy/main.go` — standalone verification CLI
- `.claude/skills/verify-deploy/SKILL.md` — Claude Code skill

---

## Out of Scope

- Extension ID 0 sentinel bug (R1) — the contract says "DO NOT MODIFY" on `setExtensionId()` and `_getExtensionId()`. The verification tool can warn about it but we don't change the contract logic.
- Steps 2-6 check implementation — architecture supports them, stubs are in place, implemented later.
- `SetupExtension()` idempotency — that's a Step 2 concern.
- Changes to other scripts (`start-services.sh`, `post-build.sh`, `test.sh`) — future work following the same patterns.

---

## Known Limitation: Extension ID 0

The verification tool should include a check (under Step 2 stubs, or as a deploy-time warning) that queries `TeeExtensionRegistry.extensionsCounter()`. If the counter is 0, warn:

> "The next registered extension will get ID 0. The current InstructionSender contract uses `_extensionId != 0` as a sentinel for 'not set', which means extension ID 0 will cause `_getExtensionId()` to always revert with 'Extension ID is not set'. See edge case R1."

This is informational only — we don't modify the contract's `setExtensionId`/`_getExtensionId` functions.
