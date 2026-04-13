# Registration Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make extension registration idempotent, validate event parsing, and extend verify-deploy with `--step register` checks (R1-R7).

**Architecture:** Two tracks — (1) harden `SetupExtension()` and `register-extension` in-place, (2) extend existing verify-deploy CLI and skill with registration checks.

**Tech Stack:** Go (standard lib + go-ethereum), existing `validate` package from Step 1 hardening

**Spec:** `docs/superpowers/specs/2026-04-09-registration-hardening-design.md`

---

## File Structure

**Modified:**
- `tools/pkg/fccutils/extension.go` — idempotent `SetupExtension()`, `findExistingExtension()` helper, event validation, cleaner error handling
- `tools/cmd/register-extension/main.go` — pre-flight `AddressHasCode`, extension ID warning
- `tools/pkg/validate/checks.go` — new `RegisterRegistrationChecks()`, helper `parseExtensionEnv()`
- `tools/pkg/validate/checks_test.go` — tests for registration checks
- `tools/cmd/verify-deploy/main.go` — wire up registration checks
- `.claude/skills/verify-deploy/SKILL.md` — R1-R8 interpretation

---

## Task 1: Make `SetupExtension()` Idempotent + Fix Event Parsing

**Files:**
- Modify: `tools/pkg/fccutils/extension.go`

This is the largest task — it rewrites the core of `SetupExtension()` and its helpers.

- [ ] **Step 1: Read the current file**

Read `tools/pkg/fccutils/extension.go` to confirm current content matches expected state.

- [ ] **Step 2: Add `findExistingExtension()` helper**

Add this function after the existing `SetupExtension()` function:

```go
// findExistingExtension checks if an extension already exists for the given instruction sender.
// Returns the extension ID if found, nil if not found.
func findExistingExtension(s *support.Support, instructionsSenderAddress common.Address) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: context.Background()}

	count, err := s.TeeExtensionRegistry.ExtensionsCounter(callOpts)
	if err != nil {
		return nil, errors.Errorf("failed to query extensions counter: %s", err)
	}

	for i := int64(0); i < count.Int64(); i++ {
		id := big.NewInt(i)
		sender, err := s.TeeExtensionRegistry.GetTeeExtensionInstructionsSender(callOpts, id)
		if err != nil {
			continue // skip unreadable extensions
		}
		if sender == instructionsSenderAddress {
			return id, nil
		}
	}

	return nil, nil
}
```

Add `"context"` to the import block if not already present.

- [ ] **Step 3: Rewrite `SetupExtension()` to be idempotent**

Replace the entire `SetupExtension()` function (lines 20-66) with:

```go
func SetupExtension(
	s *support.Support,
	governanceHash common.Hash,
	instructionsSenderAddress, stateVerifierAddress common.Address,
) (*big.Int, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{
		From:    crypto.PubkeyToAddress(s.Prv.PublicKey),
		Context: context.Background(),
	}
	deployerAddr := crypto.PubkeyToAddress(s.Prv.PublicKey)

	// Step 1: Register extension (or find existing)
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

	// Step 2: Allow TEE machine owners
	isTeeMachineOwner, err := s.TeeOwnerAllowlist.IsAllowedTeeMachineOwner(callOpts, extensionID, deployerAddr)
	if err != nil {
		return nil, errors.Errorf("failed to check TEE machine owner status: %s", err)
	}
	if isTeeMachineOwner {
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

	// Step 3: Allow wallet project owners
	isWalletProjectOwner, err := s.TeeOwnerAllowlist.IsAllowedTeeWalletProjectOwner(callOpts, extensionID, deployerAddr)
	if err != nil {
		return nil, errors.Errorf("failed to check wallet project owner status: %s", err)
	}
	if isWalletProjectOwner {
		logger.Infof("Deployer %s already allowed as wallet project owner for extension %s, skipping",
			deployerAddr.Hex(), extensionID.String())
	} else {
		_, err = allowTeeProjectManagerOwners(s, opts, extensionID, []common.Address{deployerAddr})
		if err != nil {
			return nil, errors.Errorf("failed adding wallet project owners (extension ID %s, machine owners set, but project owners failed): %s",
				extensionID.String(), err)
		}
		logger.Infof("Wallet project owners allowed for extension %s", extensionID.String())
	}

	// Step 4: Add supported key types
	isKeyTypeSupported, err := IsKeyTypeSupported(s, extensionID, wallets.EVMType)
	if err != nil {
		return nil, err
	}
	if isKeyTypeSupported {
		logger.Infof("EVM key type already supported for extension %s, skipping", extensionID.String())
	} else {
		logger.Infof("Adding key type %s to extension %s", wallets.EVMType, extensionID)
		err = AddSupportedKeyTypes(s, extensionID, []common.Hash{wallets.EVMType})
		if err != nil {
			return nil, err
		}
	}

	return extensionID, nil
}
```

- [ ] **Step 4: Fix `registerExtension()` — validate log length and separate error checks**

Replace the `registerExtension()` function (lines 98-124) with:

```go
func registerExtension(
	s *support.Support, opts *bind.TransactOpts, instructionsSenderAddress, stateVerifierAddress common.Address,
) (
	*teeextensionregistry.TeeExtensionRegistryTeeExtensionRegistered, *teeextensionregistry.TeeExtensionRegistryTeeExtensionContractsSet, error,
) {
	tx, err := s.TeeExtensionRegistry.Register(opts, stateVerifierAddress, instructionsSenderAddress)
	if err != nil {
		return nil, nil, errors.Errorf("TeeExtensionRegistry.Register failed: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return nil, nil, errors.Errorf("failed waiting for Register transaction: %s", err)
	}

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

	extensionRegistered, err := s.TeeExtensionRegistry.ParseTeeExtensionRegistered(*receipt.Logs[0])
	if err != nil {
		return nil, nil, errors.Errorf("failed to parse TeeExtensionRegistered event: %s", err)
	}

	if extensionRegistered.ExtensionId == nil || extensionRegistered.ExtensionId.Sign() == 0 {
		logger.Warnf("WARNING: extension ID is 0 — this may cause issues with setExtensionId() sentinel logic")
	}

	extensionContractsSet, err := s.TeeExtensionRegistry.ParseTeeExtensionContractsSet(*receipt.Logs[1])
	if err != nil {
		return nil, nil, errors.Errorf("failed to parse TeeExtensionContractsSet event: %s", err)
	}

	return extensionRegistered, extensionContractsSet, nil
}
```

- [ ] **Step 5: Fix `allowTeeMachineOwners()` — validate log length and separate error checks**

Replace the `allowTeeMachineOwners()` function (lines 126-143) with:

```go
func allowTeeMachineOwners(s *support.Support, opts *bind.TransactOpts, extensionId *big.Int, owners []common.Address) (*teeownerallowlist.TeeOwnerAllowlistAllowedTeeMachineOwnersAdded, error) {
	tx, err := s.TeeOwnerAllowlist.AddAllowedTeeMachineOwners(opts, extensionId, owners)
	if err != nil {
		return nil, errors.Errorf("AddAllowedTeeMachineOwners failed: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return nil, errors.Errorf("failed waiting for AddAllowedTeeMachineOwners transaction: %s", err)
	}

	if receipt.Status != 1 {
		return nil, errors.Errorf("AddAllowedTeeMachineOwners transaction failed (receipt status %d)", receipt.Status)
	}

	if len(receipt.Logs) == 0 {
		return nil, errors.New("no logs in AddAllowedTeeMachineOwners transaction — unexpected")
	}

	ownersAdded, err := s.TeeOwnerAllowlist.ParseAllowedTeeMachineOwnersAdded(*receipt.Logs[0])
	if err != nil {
		return nil, errors.Errorf("failed to parse AllowedTeeMachineOwnersAdded event: %s", err)
	}

	return ownersAdded, nil
}
```

- [ ] **Step 6: Fix `allowTeeProjectManagerOwners()` — same pattern**

Replace the `allowTeeProjectManagerOwners()` function (lines 145-162) with:

```go
func allowTeeProjectManagerOwners(s *support.Support, opts *bind.TransactOpts, extensionId *big.Int, owners []common.Address) (*teeownerallowlist.TeeOwnerAllowlistAllowedTeeWalletProjectOwnersAdded, error) {
	tx, err := s.TeeOwnerAllowlist.AddAllowedTeeWalletProjectOwners(opts, extensionId, owners)
	if err != nil {
		return nil, errors.Errorf("AddAllowedTeeWalletProjectOwners failed: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return nil, errors.Errorf("failed waiting for AddAllowedTeeWalletProjectOwners transaction: %s", err)
	}

	if receipt.Status != 1 {
		return nil, errors.Errorf("AddAllowedTeeWalletProjectOwners transaction failed (receipt status %d)", receipt.Status)
	}

	if len(receipt.Logs) == 0 {
		return nil, errors.New("no logs in AddAllowedTeeWalletProjectOwners transaction — unexpected")
	}

	ownersAdded, err := s.TeeOwnerAllowlist.ParseAllowedTeeWalletProjectOwnersAdded(*receipt.Logs[0])
	if err != nil {
		return nil, errors.Errorf("failed to parse AllowedTeeWalletProjectOwnersAdded event: %s", err)
	}

	return ownersAdded, nil
}
```

- [ ] **Step 7: Verify it compiles**

Run: `cd tools && go build ./pkg/fccutils/`
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add tools/pkg/fccutils/extension.go
git commit -m "feat: make SetupExtension idempotent with check-before-act

Each of the 4 registration sub-steps now checks on-chain state before
acting and logs 'skipping' if already done. Fixes re-run after partial
failure. Also validates receipt log length and separates status/parse
error checks."
```

---

## Task 2: Harden `register-extension/main.go`

**Files:**
- Modify: `tools/cmd/register-extension/main.go`

- [ ] **Step 1: Read the current file**

Read `tools/cmd/register-extension/main.go` to confirm current content.

- [ ] **Step 2: Add pre-flight validation and extension ID warning**

Add `validate` and `crypto` imports, then add validation after `DefaultSupport` and after `SetupExtension`.

Replace the entire file with:

```go
package main

import (
	"flag"
	"fmt"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	"extension-scaffold/tools/pkg/validate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	instructionSenderF := flag.String("instructionSender", "", "InstructionSender contract address (required)")
	governanceHashF := flag.String("governanceHash", "", "governance hash (optional)")
	flag.Parse()

	if *instructionSenderF == "" {
		logger.Fatal("--instructionSender flag is required")
	}

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	instructionSenderAddress := common.HexToAddress(*instructionSenderF)

	// Pre-flight: verify instruction sender has code on-chain
	if err := validate.AddressHasCode(testSupport.ChainClient, instructionSenderAddress, "InstructionSender"); err != nil {
		fccutils.FatalWithCause(err)
	}

	governanceHash := common.HexToHash(*governanceHashF)

	logger.Infof("Registering extension with InstructionSender %s...", instructionSenderAddress.Hex())
	extensionID, err := fccutils.SetupExtension(testSupport, governanceHash, instructionSenderAddress, common.Address{})
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// Validate extension ID
	if extensionID == nil || extensionID.Sign() <= 0 {
		logger.Warnf("WARNING: extension ID is %v — verify this is expected", extensionID)
	}

	extensionIDHex := fmt.Sprintf("0x%064x", extensionID)
	logger.Infof("Extension registered with ID: %s", extensionIDHex)

	// Machine-readable output on stdout
	fmt.Println(extensionIDHex)
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd tools && go build ./cmd/register-extension/`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add tools/cmd/register-extension/main.go
git commit -m "feat: add pre-flight validation to register-extension

Validates InstructionSender has code on-chain before registration.
Warns if extension ID is zero or negative."
```

---

## Task 3: Add Registration Checks to `validate` Package

**Files:**
- Modify: `tools/pkg/validate/checks.go`
- Modify: `tools/pkg/validate/checks_test.go`

- [ ] **Step 1: Add a `parseExtensionEnv()` helper to `checks.go`**

This helper is needed by both deploy and register checks. Add it after the existing `CheckExtensionEnvFormat` function:

```go
// parseExtensionEnv reads EXTENSION_ID and INSTRUCTION_SENDER from an extension.env file.
// Returns the parsed values and any error. If the file doesn't exist, returns empty strings and nil error.
func parseExtensionEnv(path string) (extensionID string, instructionSender string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", nil // file doesn't exist — not an error
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "EXTENSION_ID":
			extensionID = val
		case "INSTRUCTION_SENDER":
			instructionSender = val
		}
	}
	return extensionID, instructionSender, nil
}
```

- [ ] **Step 2: Add `RegisterRegistrationChecks()` function**

Add these imports to the import block in `checks.go`:

```go
"math/big"

"github.com/ethereum/go-ethereum/accounts/abi/bind"
"github.com/ethereum/go-ethereum/crypto"
"github.com/flare-foundation/go-flare-common/pkg/contracts/teeextensionregistry"
"github.com/flare-foundation/go-flare-common/pkg/contracts/teeownerallowlist"
"github.com/flare-foundation/tee-node/pkg/wallets"
```

Then add the function:

```go
// RegisterRegistrationChecks adds Step 2 (Extension Registration) checks to a Report.
// Requires extension.env to exist with EXTENSION_ID and INSTRUCTION_SENDER.
func RegisterRegistrationChecks(
	r *Report,
	client *ethclient.Client,
	key *ecdsa.PrivateKey,
	registry *teeextensionregistry.TeeExtensionRegistry,
	allowlist *teeownerallowlist.TeeOwnerAllowlist,
	extensionEnvPath string,
) {
	if extensionEnvPath == "" {
		r.Add(CheckResult{
			Step: "register", ID: "R1-R7", Name: "extension registration checks",
			Status: SKIP, Message: "no --config path provided",
		})
		return
	}

	extIDStr, instrSenderStr, err := parseExtensionEnv(extensionEnvPath)
	if err != nil || (extIDStr == "" && instrSenderStr == "") {
		r.Add(CheckResult{
			Step: "register", ID: "R1-R7", Name: "extension registration checks",
			Status: SKIP, Message: "config/extension.env not found — run pre-build.sh first",
		})
		return
	}

	callOpts := &bind.CallOpts{Context: context.Background()}

	// R1: Check extensions counter > 0
	counter, err := registry.ExtensionsCounter(callOpts)
	if err != nil {
		r.Add(CheckResult{
			Step: "register", ID: "R1", Name: "extensions counter",
			Status: FAIL, Message: fmt.Sprintf("failed to query extensions counter: %v", err),
			Fix: "Check chain connectivity and registry address",
		})
		return // can't continue without counter
	}
	if counter.Sign() == 0 {
		r.Add(CheckResult{
			Step: "register", ID: "R1", Name: "extensions counter",
			Status: WARN,
			Message: "registry has 0 extensions — first registration will get ID 0 which may conflict with sentinel logic",
			Fix:     "This is expected on a fresh registry. Proceed with caution.",
		})
	} else {
		r.Add(CheckResult{
			Step: "register", ID: "R1", Name: "extensions counter",
			Status: PASS, Message: fmt.Sprintf("%s extensions registered", counter.String()),
		})
	}

	// Parse extension ID for remaining checks
	if !extensionIDRegex.MatchString(extIDStr) {
		r.Add(CheckResult{
			Step: "register", ID: "R2-R7", Name: "extension ID format",
			Status: FAIL, Message: fmt.Sprintf("EXTENSION_ID in extension.env is malformed: %q", extIDStr),
			Fix: "Delete config/extension.env and re-run pre-build.sh",
		})
		return
	}
	extensionID := new(big.Int)
	extensionID.SetString(extIDStr[2:], 16) // strip 0x prefix

	// R2: Check on-chain instruction sender matches extension.env
	if addressRegex.MatchString(instrSenderStr) {
		onChainSender, err := registry.GetTeeExtensionInstructionsSender(callOpts, extensionID)
		if err != nil {
			r.Add(CheckResult{
				Step: "register", ID: "R2", Name: "instruction sender matches on-chain",
				Status: FAIL, Message: fmt.Sprintf("failed to query instruction sender for extension %s: %v", extensionID.String(), err),
				Fix: "Extension ID may not exist on-chain. Delete config/extension.env and re-run pre-build.sh",
			})
		} else {
			envSender := common.HexToAddress(instrSenderStr)
			if onChainSender == envSender {
				r.Add(CheckResult{
					Step: "register", ID: "R2", Name: "instruction sender matches on-chain",
					Status: PASS, Message: fmt.Sprintf("extension %s points to %s", extensionID.String(), onChainSender.Hex()),
				})
			} else {
				r.Add(CheckResult{
					Step: "register", ID: "R2", Name: "instruction sender matches on-chain",
					Status: FAIL,
					Message: fmt.Sprintf("extension %s has instruction sender %s on-chain, but extension.env says %s",
						extensionID.String(), onChainSender.Hex(), envSender.Hex()),
					Fix: "Config is stale or wrong contract was deployed. Delete config/extension.env and re-run pre-build.sh",
				})
			}
		}
	}

	// R3: Check deployer is allowed TEE machine owner
	if key != nil {
		deployerAddr := crypto.PubkeyToAddress(key.PublicKey)
		isOwner, err := allowlist.IsAllowedTeeMachineOwner(callOpts, extensionID, deployerAddr)
		if err != nil {
			r.Add(CheckResult{
				Step: "register", ID: "R3", Name: "deployer is allowed TEE machine owner",
				Status: FAIL, Message: fmt.Sprintf("failed to check owner status: %v", err),
				Fix: "Check chain connectivity",
			})
		} else if isOwner {
			r.Add(CheckResult{
				Step: "register", ID: "R3", Name: "deployer is allowed TEE machine owner",
				Status: PASS, Message: fmt.Sprintf("deployer %s is allowed", deployerAddr.Hex()),
			})
		} else {
			r.Add(CheckResult{
				Step: "register", ID: "R3", Name: "deployer is allowed TEE machine owner",
				Status: FAIL,
				Message: fmt.Sprintf("deployer %s is NOT an allowed TEE machine owner for extension %s",
					deployerAddr.Hex(), extensionID.String()),
				Fix: "Registration partially failed. Re-run pre-build.sh (it will skip completed steps)",
			})
		}
	}

	// R4: Check EVM key type is supported
	evmType := wallets.EVMType
	var evmTypeBytes [32]byte
	copy(evmTypeBytes[:], evmType[:])
	isSupported, err := registry.IsKeyTypeSupported(callOpts, extensionID, evmTypeBytes)
	if err != nil {
		r.Add(CheckResult{
			Step: "register", ID: "R4", Name: "EVM key type supported",
			Status: FAIL, Message: fmt.Sprintf("failed to check key type support: %v", err),
			Fix: "Check chain connectivity",
		})
	} else if isSupported {
		r.Add(CheckResult{
			Step: "register", ID: "R4", Name: "EVM key type supported",
			Status: PASS, Message: "EVM key type is supported",
		})
	} else {
		r.Add(CheckResult{
			Step: "register", ID: "R4", Name: "EVM key type supported",
			Status: FAIL,
			Message: fmt.Sprintf("EVM key type not supported for extension %s", extensionID.String()),
			Fix: "Registration partially failed. Re-run pre-build.sh",
		})
	}

	// R7: Check for duplicate instruction sender registrations
	if addressRegex.MatchString(instrSenderStr) && counter.Int64() > 0 {
		envSender := common.HexToAddress(instrSenderStr)
		var matchingIDs []string
		for i := int64(0); i < counter.Int64(); i++ {
			id := big.NewInt(i)
			sender, err := registry.GetTeeExtensionInstructionsSender(callOpts, id)
			if err != nil {
				continue
			}
			if sender == envSender {
				matchingIDs = append(matchingIDs, id.String())
			}
		}
		if len(matchingIDs) > 1 {
			r.Add(CheckResult{
				Step: "register", ID: "R7", Name: "no duplicate instruction sender",
				Status: WARN,
				Message: fmt.Sprintf("instruction sender %s is registered for multiple extensions (IDs: %s). "+
					"setExtensionId() will always resolve to first (ID %s)",
					envSender.Hex(), strings.Join(matchingIDs, ", "), matchingIDs[0]),
				Fix: "If the latest registration was intended, deploy a new instruction sender contract",
			})
		} else {
			r.Add(CheckResult{
				Step: "register", ID: "R7", Name: "no duplicate instruction sender",
				Status: PASS, Message: "instruction sender is unique",
			})
		}
	}
}
```

- [ ] **Step 3: Add tests for registration checks**

Append to `tools/pkg/validate/checks_test.go`:

```go
func TestParseExtensionEnv_Valid(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "extension.env")
	os.WriteFile(envFile, []byte(
		"# Auto-generated\n"+
			"EXTENSION_ID=0x00000000000000000000000000000000000000000000000000000000000000db\n"+
			"INSTRUCTION_SENDER=0x32F967bE8F35F73274Bd3d4130073547361A0d75\n",
	), 0644)

	extID, instrSender, err := parseExtensionEnv(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extID != "0x00000000000000000000000000000000000000000000000000000000000000db" {
		t.Fatalf("unexpected EXTENSION_ID: %s", extID)
	}
	if instrSender != "0x32F967bE8F35F73274Bd3d4130073547361A0d75" {
		t.Fatalf("unexpected INSTRUCTION_SENDER: %s", instrSender)
	}
}

func TestParseExtensionEnv_Missing(t *testing.T) {
	extID, instrSender, err := parseExtensionEnv("/nonexistent/extension.env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extID != "" || instrSender != "" {
		t.Fatalf("expected empty strings for missing file, got %q and %q", extID, instrSender)
	}
}

func TestRegisterRegistrationChecks_NoConfig(t *testing.T) {
	r := &Report{}
	RegisterRegistrationChecks(r, nil, nil, nil, nil, "")
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	if r.Results[0].Status != SKIP {
		t.Fatalf("expected SKIP, got %s", r.Results[0].Status)
	}
}

func TestRegisterRegistrationChecks_MissingFile(t *testing.T) {
	r := &Report{}
	RegisterRegistrationChecks(r, nil, nil, nil, nil, "/nonexistent/extension.env")
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	if r.Results[0].Status != SKIP {
		t.Fatalf("expected SKIP, got %s", r.Results[0].Status)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd tools && go test ./pkg/validate/ -v`
Expected: All tests PASS (new + existing).

- [ ] **Step 5: Commit**

```bash
git add tools/pkg/validate/checks.go tools/pkg/validate/checks_test.go
git commit -m "feat: add registration checks for verify-deploy --step register

RegisterRegistrationChecks implements R1-R7 checks: extensions counter,
instruction sender mismatch, owner permissions, key type support, and
duplicate sender detection."
```

---

## Task 4: Wire Up Registration Checks in `verify-deploy`

**Files:**
- Modify: `tools/cmd/verify-deploy/main.go`

- [ ] **Step 1: Replace the register stub with real checks**

In `tools/cmd/verify-deploy/main.go`, replace both occurrences of:

```go
validate.RegisterStubChecks(report, "register", "R1-R8", "Extension registration checks")
```

With:

```go
validate.RegisterRegistrationChecks(report, s.ChainClient, s.Prv, s.TeeExtensionRegistry, s.TeeOwnerAllowlist, *configPath)
```

There are two occurrences — one in `case "register":` (line 44) and one in `case "all":` (line 55).

- [ ] **Step 2: Verify it compiles**

Run: `cd tools && go build ./cmd/verify-deploy/`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add tools/cmd/verify-deploy/main.go
git commit -m "feat: wire up registration checks in verify-deploy --step register

Replaces R1-R8 stub with real checks for extensions counter, instruction
sender mismatch, owner permissions, key type support, and duplicate
sender detection."
```

---

## Task 5: Update `/verify-deploy` Skill with R1-R8 Interpretation

**Files:**
- Modify: `.claude/skills/verify-deploy/SKILL.md`

- [ ] **Step 1: Add R1-R8 interpretation section**

After the D7 entry in the "Interpret results" section (after line 69), add:

```markdown

**R1 — Extensions counter is zero:**
The registry has no extensions registered. If this is a fresh deployment, that's expected — the first call to pre-build.sh will register one. If you expected extensions to exist, check CHAIN_URL and ADDRESSES_FILE.

**R2 — Instruction sender mismatch:**
The instruction sender address in extension.env doesn't match what's registered on-chain for this extension ID. The config is stale or a different contract was deployed.
Fix: Delete config/extension.env and re-run `scripts/pre-build.sh`.

**R3 — Deployer not allowed as TEE machine owner:**
The registration partially failed — the extension was created but the deployer wasn't added as a TEE machine owner. This means TEE machine registration will fail later.
Fix: Re-run `scripts/pre-build.sh` — it's now idempotent and will skip completed steps.

**R4 — EVM key type not supported:**
The registration partially failed — the extension was created but the EVM key type wasn't enabled.
Fix: Re-run `scripts/pre-build.sh`.

**R5 — Partially configured extension:**
Composite detection of R3/R4. The extension exists but is missing owner permissions or key type support.
Fix: Re-run `scripts/pre-build.sh`.

**R7 — Duplicate instruction sender:**
The same instruction sender contract is registered for multiple extensions. `setExtensionId()` in the Solidity contract will always resolve to the first (lowest ID) extension.
Fix: If the latest registration was intended, deploy a new instruction sender contract and re-run pre-build.sh.
```

- [ ] **Step 2: Add registration-specific compound misconfiguration**

Add to the "Detect compound misconfigurations" section:

```markdown
- **extension.env has EXTENSION_ID but R3/R4 fail** = partial registration, re-run pre-build.sh (now idempotent)
```

- [ ] **Step 3: Update the edge cases reference**

After the existing edge cases reference line, add:

```markdown
Registration hardening spec: `docs/superpowers/specs/2026-04-09-registration-hardening-design.md`
```

- [ ] **Step 4: Commit**

```bash
git add .claude/skills/verify-deploy/SKILL.md
git commit -m "feat: add R1-R8 interpretation to /verify-deploy skill

Covers extensions counter, instruction sender mismatch, owner
permissions, key type support, and duplicate sender detection."
```

---

## Task 6: Final Verification

- [ ] **Step 1: Build everything**

Run: `cd tools && go build ./...`
Expected: All packages compile (ignore pre-existing errors in external deps like tee-proxy).

- [ ] **Step 2: Run all validate tests**

Run: `cd tools && go test ./pkg/validate/ -v`
Expected: All tests PASS.

- [ ] **Step 3: Build specific commands**

Run: `cd tools && go build ./cmd/deploy-contract/ && go build ./cmd/register-extension/ && go build ./cmd/verify-deploy/`
Expected: All three build successfully.

- [ ] **Step 4: Verify skill file is well-formed**

Run: `head -5 .claude/skills/verify-deploy/SKILL.md`
Expected: Shows `# Verify Deploy` header.
