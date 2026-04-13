# Deployment Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the contract deployment pipeline so every failure produces a clear, actionable error, and provide a standalone verification tool + Claude Code skill for comprehensive deployment diagnostics.

**Architecture:** Three-layer defense — (1) script hardening in `pre-build.sh`, (2) Go pre-flight validation in deploy commands via a shared `validate` package, (3) standalone `verify-deploy` CLI + Claude Code skill. Solidity constructor gets last-line-of-defense checks.

**Tech Stack:** Go (standard lib + go-ethereum), Bash, Solidity 0.8.27

**Spec:** `docs/superpowers/specs/2026-04-09-deployment-hardening-design.md`

---

## File Structure

**New files:**
- `tools/pkg/validate/validate.go` — shared validation primitives (`AddressHasCode`, `KeyHasFunds`, `AddressNotZero`)
- `tools/pkg/validate/validate_test.go` — unit tests for validation primitives
- `tools/pkg/validate/report.go` — `CheckResult`, `Report` types, terminal + JSON output
- `tools/pkg/validate/report_test.go` — unit tests for report formatting
- `tools/pkg/validate/checks.go` — composable per-step check functions
- `tools/pkg/validate/checks_test.go` — unit tests for check functions
- `tools/cmd/verify-deploy/main.go` — standalone verification CLI
- `.claude/skills/verify-deploy/SKILL.md` — Claude Code skill

**Modified files:**
- `tools/pkg/support/support.go` — remove key leak (line 93), stderr for warnings, WaitMined timeout
- `tools/cmd/deploy-contract/main.go` — pre-flight checks, `--preflight-only` flag
- `tools/pkg/utils/instructions.go` — WaitMined timeout in `DeployInstructionSender`
- `contracts/InstructionSender.sol` — constructor validation
- `scripts/pre-build.sh` — stderr capture, regex validation, preflight step

---

## Task 1: Shared Validation Primitives (`tools/pkg/validate/validate.go`)

**Files:**
- Create: `tools/pkg/validate/validate.go`
- Create: `tools/pkg/validate/validate_test.go`

- [ ] **Step 1: Write failing tests for `AddressNotZero`**

Create `tools/pkg/validate/validate_test.go`:

```go
package validate

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestAddressNotZero_ZeroAddress(t *testing.T) {
	err := AddressNotZero(common.Address{}, "TeeExtensionRegistry")
	if err == nil {
		t.Fatal("expected error for zero address")
	}
	if got := err.Error(); got != "TeeExtensionRegistry is the zero address (0x0000000000000000000000000000000000000000) in the addresses file" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestAddressNotZero_NonZeroAddress(t *testing.T) {
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	err := AddressNotZero(addr, "TeeExtensionRegistry")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd tools && go test ./pkg/validate/ -v -run TestAddressNotZero`
Expected: Compilation error — package doesn't exist yet.

- [ ] **Step 3: Implement `validate.go` with all primitives**

Create `tools/pkg/validate/validate.go`:

```go
package validate

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// MinDeployBalance is the minimum balance required to deploy a contract and register
// an extension on Coston2 (~0.01 ETH). Callers can override if needed.
var MinDeployBalance = big.NewInt(10_000_000_000_000_000) // 0.01 ETH

// AddressNotZero checks that the address is not the zero address.
func AddressNotZero(addr common.Address, label string) error {
	if addr == (common.Address{}) {
		return fmt.Errorf("%s is the zero address (%s) in the addresses file", label, addr.Hex())
	}
	return nil
}

// AddressHasCode checks that the address has deployed bytecode on-chain.
// This catches both wrong addresses and EOAs passed where contracts are expected.
func AddressHasCode(client *ethclient.Client, addr common.Address, label string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return fmt.Errorf("failed to query code at %s (%s): %w", label, addr.Hex(), err)
	}
	if len(code) == 0 {
		return fmt.Errorf(
			"%s at %s has no deployed code. "+
				"This address will be set as immutable in the contract constructor and cannot be changed. "+
				"Check your deployed-addresses.json file — are you on the right network?",
			label, addr.Hex(),
		)
	}
	return nil
}

// KeyHasFunds checks that the key's account has at least minWei balance.
func KeyHasFunds(client *ethclient.Client, key *ecdsa.PrivateKey, minWei *big.Int) error {
	addr := crypto.PubkeyToAddress(key.PublicKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return fmt.Errorf("failed to query balance for deployer %s: %w", addr.Hex(), err)
	}
	if balance.Cmp(minWei) < 0 {
		return fmt.Errorf(
			"deployer %s has insufficient funds (balance: %s wei, minimum required: %s wei). "+
				"Fund this account before deploying",
			addr.Hex(), balance.String(), minWei.String(),
		)
	}
	return nil
}

// IsUsingDevKey checks if PRIV_KEY env var is unset (meaning the hardcoded dev key is being used).
// Returns true if using the dev key, false if a real key is configured.
func IsUsingDevKey() bool {
	return os.Getenv("PRIV_KEY") == ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd tools && go test ./pkg/validate/ -v -run TestAddressNotZero`
Expected: PASS

- [ ] **Step 5: Add tests for `AddressHasCode` and `KeyHasFunds`**

These require an ethclient, so we test the error-formatting path by using a mock-like approach — test the functions that don't need a live chain, and add integration-style tests that verify error message format with nil client (expecting connection errors):

Append to `tools/pkg/validate/validate_test.go`:

```go
func TestAddressHasCode_NilClient(t *testing.T) {
	// Without a live chain, verify the function returns an error (not panic)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	err := AddressHasCode(nil, addr, "TestRegistry")
	if err == nil {
		t.Fatal("expected error with nil client")
	}
}

func TestKeyHasFunds_NilClient(t *testing.T) {
	key, _ := crypto.GenerateKey()
	err := KeyHasFunds(nil, key, big.NewInt(1))
	if err == nil {
		t.Fatal("expected error with nil client")
	}
}

func TestIsUsingDevKey_WhenSet(t *testing.T) {
	t.Setenv("PRIV_KEY", "abc123")
	if IsUsingDevKey() {
		t.Fatal("expected false when PRIV_KEY is set")
	}
}

func TestIsUsingDevKey_WhenUnset(t *testing.T) {
	t.Setenv("PRIV_KEY", "")
	if !IsUsingDevKey() {
		t.Fatal("expected true when PRIV_KEY is empty")
	}
}
```

- [ ] **Step 6: Run all validate tests**

Run: `cd tools && go test ./pkg/validate/ -v`
Expected: All tests PASS (nil client tests will fail with runtime errors — adjust the functions to handle nil client gracefully if needed, or accept the error path).

Note: `AddressHasCode` and `KeyHasFunds` will panic on nil client since `ethclient` methods dereference the receiver. Wrap with a nil check at the top of each:

```go
// At top of AddressHasCode:
if client == nil {
	return fmt.Errorf("cannot check %s: no chain client connected", label)
}

// At top of KeyHasFunds:
if client == nil {
	return fmt.Errorf("cannot check deployer funds: no chain client connected")
}
```

- [ ] **Step 7: Run tests again after nil-guard fix**

Run: `cd tools && go test ./pkg/validate/ -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add tools/pkg/validate/validate.go tools/pkg/validate/validate_test.go
git commit -m "feat: add shared validation primitives for deployment checks

AddressNotZero, AddressHasCode, KeyHasFunds, IsUsingDevKey — reusable
validators for pre-flight deployment checks."
```

---

## Task 2: Report Types (`tools/pkg/validate/report.go`)

**Files:**
- Create: `tools/pkg/validate/report.go`
- Create: `tools/pkg/validate/report_test.go`

- [ ] **Step 1: Write failing tests for Report**

Create `tools/pkg/validate/report_test.go`:

```go
package validate

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestReport_Summary(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{Step: "deploy", ID: "D1", Name: "registry has code", Status: PASS, Message: "ok"})
	r.Add(CheckResult{Step: "deploy", ID: "D5", Name: "key source", Status: WARN, Message: "using dev key", Fix: "set PRIV_KEY"})
	r.Add(CheckResult{Step: "register", ID: "R1", Name: "stub", Status: SKIP, Message: "not implemented"})

	summary := r.Summary()
	expected := "1 passed, 1 warning, 0 failed, 1 skipped"
	if summary != expected {
		t.Fatalf("expected %q, got %q", expected, summary)
	}
}

func TestReport_HasFailures(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{Status: PASS})
	r.Add(CheckResult{Status: WARN})
	if r.HasFailures() {
		t.Fatal("expected no failures")
	}

	r.Add(CheckResult{Status: FAIL})
	if !r.HasFailures() {
		t.Fatal("expected failures")
	}
}

func TestReport_PrintJSON(t *testing.T) {
	r := &Report{}
	r.Add(CheckResult{Step: "deploy", ID: "D1", Name: "test", Status: PASS, Message: "ok"})

	var buf bytes.Buffer
	r.FprintJSON(&buf)

	var parsed struct {
		Results []struct {
			Step   string `json:"step"`
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"results"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(parsed.Results))
	}
	if parsed.Results[0].Status != "PASS" {
		t.Fatalf("expected PASS, got %s", parsed.Results[0].Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd tools && go test ./pkg/validate/ -v -run TestReport`
Expected: Compilation error — `Report`, `CheckResult`, etc. not defined yet.

- [ ] **Step 3: Implement `report.go`**

Create `tools/pkg/validate/report.go`:

```go
package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Status represents the outcome of a verification check.
type Status int

const (
	PASS Status = iota
	WARN
	FAIL
	SKIP
)

func (s Status) String() string {
	switch s {
	case PASS:
		return "PASS"
	case WARN:
		return "WARN"
	case FAIL:
		return "FAIL"
	case SKIP:
		return "SKIP"
	default:
		return "UNKNOWN"
	}
}

// CheckResult represents the outcome of a single verification check.
type CheckResult struct {
	Step    string // "deploy", "register", "services", "tee-version", "tee-machine", "test"
	ID      string // "D1", "D2", etc. — maps to edge cases doc
	Name    string // human-readable check name
	Status  Status
	Message string // what was found
	Fix     string // how to fix (only on FAIL/WARN)
}

// Report collects CheckResults and produces formatted output.
type Report struct {
	Results []CheckResult
}

// Add appends a check result to the report.
func (r *Report) Add(result CheckResult) {
	r.Results = append(r.Results, result)
}

// HasFailures returns true if any check has FAIL status.
func (r *Report) HasFailures() bool {
	for _, result := range r.Results {
		if result.Status == FAIL {
			return true
		}
	}
	return false
}

// Summary returns a one-line summary like "7 passed, 1 warning, 0 failed, 5 skipped".
func (r *Report) Summary() string {
	counts := map[Status]int{}
	for _, result := range r.Results {
		counts[result.Status]++
	}
	return fmt.Sprintf("%d passed, %d warning, %d failed, %d skipped",
		counts[PASS], counts[WARN], counts[FAIL], counts[SKIP])
}

// Fprint writes the colored terminal report to w.
func (r *Report) Fprint(w io.Writer) {
	red := "\033[0;31m"
	green := "\033[0;32m"
	yellow := "\033[0;33m"
	cyan := "\033[0;36m"
	gray := "\033[0;90m"
	nc := "\033[0m"

	fmt.Fprintf(w, "\n%s=== Deployment Verification Report ===%s\n", cyan, nc)

	currentStep := ""
	for _, result := range r.Results {
		if result.Step != currentStep {
			currentStep = result.Step
			fmt.Fprintf(w, "\n%sStep: %s%s\n", cyan, stepLabel(currentStep), nc)
		}

		var color string
		switch result.Status {
		case PASS:
			color = green
		case WARN:
			color = yellow
		case FAIL:
			color = red
		case SKIP:
			color = gray
		}

		idStr := ""
		if result.ID != "" {
			idStr = result.ID + "  "
		}

		fmt.Fprintf(w, "  %s[%s]%s %s%s\n", color, result.Status, nc, idStr, result.Name)
		if result.Message != "" && result.Status != PASS {
			fmt.Fprintf(w, "         %s\n", result.Message)
		}
		if result.Fix != "" {
			fmt.Fprintf(w, "         Fix: %s\n", result.Fix)
		}
	}

	fmt.Fprintf(w, "\n%s\n", r.Summary())
}

// Print writes the report to stdout.
func (r *Report) Print() {
	r.Fprint(os.Stdout)
}

// jsonReport is the JSON serialization format.
type jsonReport struct {
	Results []jsonResult `json:"results"`
	Summary string       `json:"summary"`
}

type jsonResult struct {
	Step    string `json:"step"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

// FprintJSON writes the report as JSON to w.
func (r *Report) FprintJSON(w io.Writer) {
	jr := jsonReport{Summary: r.Summary()}
	for _, result := range r.Results {
		jr.Results = append(jr.Results, jsonResult{
			Step:    result.Step,
			ID:      result.ID,
			Name:    result.Name,
			Status:  result.Status.String(),
			Message: result.Message,
			Fix:     result.Fix,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(jr)
}

// PrintJSON writes the report as JSON to stdout.
func (r *Report) PrintJSON() {
	r.FprintJSON(os.Stdout)
}

func stepLabel(step string) string {
	labels := map[string]string{
		"deploy":      "Contract Deployment",
		"register":    "Extension Registration",
		"services":    "Service Startup",
		"tee-version": "TEE Version Registration",
		"tee-machine": "TEE Machine Registration",
		"test":        "Testing",
	}
	if label, ok := labels[step]; ok {
		return label
	}
	return strings.Title(step)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd tools && go test ./pkg/validate/ -v -run TestReport`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add tools/pkg/validate/report.go tools/pkg/validate/report_test.go
git commit -m "feat: add Report and CheckResult types for verification output

Supports colored terminal output, JSON output, summary counts, and
PASS/WARN/FAIL/SKIP statuses."
```

---

## Task 3: Step 1 Check Functions (`tools/pkg/validate/checks.go`)

**Files:**
- Create: `tools/pkg/validate/checks.go`
- Create: `tools/pkg/validate/checks_test.go`

- [ ] **Step 1: Write failing tests for check functions**

Create `tools/pkg/validate/checks_test.go`:

```go
package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckExtensionEnvFormat_Valid(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "extension.env")
	os.WriteFile(envFile, []byte(
		"# Auto-generated\n"+
			"EXTENSION_ID=0x00000000000000000000000000000000000000000000000000000000000000db\n"+
			"INSTRUCTION_SENDER=0x32F967bE8F35F73274Bd3d4130073547361A0d75\n",
	), 0644)

	results := CheckExtensionEnvFormat(envFile)
	for _, r := range results {
		if r.Status == FAIL {
			t.Fatalf("unexpected FAIL: %s — %s", r.Name, r.Message)
		}
	}
}

func TestCheckExtensionEnvFormat_Malformed(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "extension.env")
	os.WriteFile(envFile, []byte(
		"EXTENSION_ID=garbage\n"+
			"INSTRUCTION_SENDER=not-an-address\n",
	), 0644)

	results := CheckExtensionEnvFormat(envFile)
	failCount := 0
	for _, r := range results {
		if r.Status == FAIL {
			failCount++
		}
	}
	if failCount != 2 {
		t.Fatalf("expected 2 FAILs, got %d", failCount)
	}
}

func TestCheckExtensionEnvFormat_Missing(t *testing.T) {
	results := CheckExtensionEnvFormat("/nonexistent/extension.env")
	if len(results) != 1 || results[0].Status != SKIP {
		t.Fatalf("expected SKIP for missing file, got %v", results)
	}
}

func TestCheckDeployerKeySource_DevKey(t *testing.T) {
	t.Setenv("PRIV_KEY", "")
	t.Setenv("LOCAL_MODE", "false")
	result := CheckDeployerKeySource()
	if result.Status != WARN {
		t.Fatalf("expected WARN, got %s", result.Status)
	}
}

func TestCheckDeployerKeySource_RealKey(t *testing.T) {
	t.Setenv("PRIV_KEY", "abc123")
	result := CheckDeployerKeySource()
	if result.Status != PASS {
		t.Fatalf("expected PASS, got %s", result.Status)
	}
}

func TestCheckDeployerKeySource_DevKeyLocal(t *testing.T) {
	t.Setenv("PRIV_KEY", "")
	t.Setenv("LOCAL_MODE", "true")
	result := CheckDeployerKeySource()
	if result.Status != PASS {
		t.Fatalf("expected PASS for local mode, got %s", result.Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd tools && go test ./pkg/validate/ -v -run "TestCheck"`
Expected: Compilation error — `CheckExtensionEnvFormat`, `CheckDeployerKeySource` not defined.

- [ ] **Step 3: Implement `checks.go`**

Create `tools/pkg/validate/checks.go`:

```go
package validate

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	addressRegex     = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	extensionIDRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
)

// RegisterDeployChecks adds all Step 1 (Contract Deployment) checks to the report.
func RegisterDeployChecks(r *Report, client *ethclient.Client, key *ecdsa.PrivateKey, addresses map[string]common.Address, extensionEnvPath string) {
	// D3: Zero address checks
	for label, addr := range addresses {
		if err := AddressNotZero(addr, label); err != nil {
			r.Add(CheckResult{
				Step: "deploy", ID: "D3", Name: fmt.Sprintf("%s is not zero address", label),
				Status: FAIL, Message: err.Error(),
				Fix: fmt.Sprintf("Check your deployed-addresses.json — %s is missing or unpopulated", label),
			})
		} else {
			r.Add(CheckResult{
				Step: "deploy", ID: "D3", Name: fmt.Sprintf("%s is not zero address", label),
				Status: PASS, Message: addr.Hex(),
			})
		}
	}

	// D1/D2: Code existence checks
	for label, addr := range addresses {
		if addr == (common.Address{}) {
			continue // already reported as D3
		}
		if err := AddressHasCode(client, addr, label); err != nil {
			r.Add(CheckResult{
				Step: "deploy", ID: "D1", Name: fmt.Sprintf("%s has code", label),
				Status: FAIL, Message: err.Error(),
				Fix: "Check CHAIN_URL and ADDRESSES_FILE — they may point to different networks",
			})
		} else {
			r.Add(CheckResult{
				Step: "deploy", ID: "D1", Name: fmt.Sprintf("%s has code at %s", label, addr.Hex()),
				Status: PASS, Message: "contract found",
			})
		}
	}

	// D5: Key source check
	r.Add(CheckDeployerKeySource())

	// D5: Balance check
	if key != nil && client != nil {
		if err := KeyHasFunds(client, key, MinDeployBalance); err != nil {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			r.Add(CheckResult{
				Step: "deploy", ID: "D5", Name: "deployer has sufficient funds",
				Status: FAIL, Message: err.Error(),
				Fix: fmt.Sprintf("Fund account %s before deploying", addr.Hex()),
			})
		} else {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			r.Add(CheckResult{
				Step: "deploy", ID: "D5", Name: fmt.Sprintf("deployer %s has sufficient funds", addr.Hex()),
				Status: PASS, Message: "balance OK",
			})
		}
	}

	// D6: Chain ID (informational)
	if client != nil {
		r.Add(checkChainID(client))
	}

	// D7: extension.env format
	if extensionEnvPath != "" {
		for _, result := range CheckExtensionEnvFormat(extensionEnvPath) {
			r.Add(result)
		}
	}
}

// CheckDeployerKeySource checks whether PRIV_KEY is set or falling back to the dev key.
func CheckDeployerKeySource() CheckResult {
	if !IsUsingDevKey() {
		return CheckResult{
			Step: "deploy", ID: "D5", Name: "deployer key source",
			Status: PASS, Message: "PRIV_KEY is set",
		}
	}

	localMode := os.Getenv("LOCAL_MODE")
	if localMode == "true" || localMode == "" {
		return CheckResult{
			Step: "deploy", ID: "D5", Name: "deployer key source",
			Status: PASS, Message: "using Hardhat dev key (LOCAL_MODE)",
		}
	}

	return CheckResult{
		Step: "deploy", ID: "D5", Name: "deployer key source",
		Status: WARN,
		Message: "PRIV_KEY not set — using Hardhat dev key which has no funds on Coston2",
		Fix:     "Set PRIV_KEY in .env to a funded account on the target network",
	}
}

// CheckExtensionEnvFormat validates the format of values in extension.env.
func CheckExtensionEnvFormat(path string) []CheckResult {
	f, err := os.Open(path)
	if err != nil {
		return []CheckResult{{
			Step: "deploy", ID: "D7", Name: "extension.env exists",
			Status: SKIP, Message: fmt.Sprintf("file not found: %s", path),
		}}
	}
	defer f.Close()

	var results []CheckResult
	values := map[string]string{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}

	// Check INSTRUCTION_SENDER format
	if sender, ok := values["INSTRUCTION_SENDER"]; ok {
		if addressRegex.MatchString(sender) {
			results = append(results, CheckResult{
				Step: "deploy", ID: "D7", Name: "INSTRUCTION_SENDER format valid",
				Status: PASS, Message: sender,
			})
		} else {
			results = append(results, CheckResult{
				Step: "deploy", ID: "D7", Name: "INSTRUCTION_SENDER format valid",
				Status: FAIL,
				Message: fmt.Sprintf("INSTRUCTION_SENDER value is malformed: '%s' (expected 0x + 40 hex chars)", sender),
				Fix:     "Delete config/extension.env and re-run scripts/pre-build.sh",
			})
		}
	} else {
		results = append(results, CheckResult{
			Step: "deploy", ID: "D7", Name: "INSTRUCTION_SENDER present",
			Status: FAIL, Message: "INSTRUCTION_SENDER not found in extension.env",
			Fix: "Re-run scripts/pre-build.sh",
		})
	}

	// Check EXTENSION_ID format
	if extID, ok := values["EXTENSION_ID"]; ok {
		if extensionIDRegex.MatchString(extID) {
			results = append(results, CheckResult{
				Step: "deploy", ID: "D7", Name: "EXTENSION_ID format valid",
				Status: PASS, Message: extID,
			})
		} else {
			results = append(results, CheckResult{
				Step: "deploy", ID: "D7", Name: "EXTENSION_ID format valid",
				Status: FAIL,
				Message: fmt.Sprintf("EXTENSION_ID value is malformed: '%s' (expected 0x + 64 hex chars)", extID),
				Fix:     "Delete config/extension.env and re-run scripts/pre-build.sh",
			})
		}
	} else {
		results = append(results, CheckResult{
			Step: "deploy", ID: "D7", Name: "EXTENSION_ID present",
			Status: FAIL, Message: "EXTENSION_ID not found in extension.env",
			Fix: "Re-run scripts/pre-build.sh",
		})
	}

	return results
}

// checkChainID returns an informational check with the connected chain ID.
func checkChainID(client *ethclient.Client) CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return CheckResult{
			Step: "deploy", ID: "D6", Name: "chain connectivity",
			Status: FAIL, Message: fmt.Sprintf("failed to get chain ID: %v", err),
			Fix: "Check CHAIN_URL — is the RPC endpoint reachable?",
		}
	}

	knownChains := map[string]string{
		"31337": "Hardhat/Anvil (local devnet)",
		"114":   "Coston2 (testnet)",
		"14":    "Flare (mainnet)",
	}

	chainName := knownChains[chainID.String()]
	if chainName == "" {
		chainName = "unknown network"
	}

	return CheckResult{
		Step: "deploy", ID: "D6", Name: "chain ID",
		Status: PASS, Message: fmt.Sprintf("chain ID %s (%s)", chainID.String(), chainName),
	}
}

// RegisterStubChecks adds SKIP placeholders for steps not yet implemented.
func RegisterStubChecks(r *Report, step, idRange, name string) {
	r.Add(CheckResult{
		Step: step, ID: idRange, Name: name,
		Status: SKIP, Message: "not yet implemented",
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd tools && go test ./pkg/validate/ -v -run "TestCheck"`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add tools/pkg/validate/checks.go tools/pkg/validate/checks_test.go
git commit -m "feat: add Step 1 check functions for deployment verification

RegisterDeployChecks composes D1-D8 checks. CheckExtensionEnvFormat
validates extension.env values. Stub checks for Steps 2-6."
```

---

## Task 4: Verification CLI (`tools/cmd/verify-deploy/main.go`)

**Files:**
- Create: `tools/cmd/verify-deploy/main.go`

- [ ] **Step 1: Implement the CLI entry point**

Create `tools/cmd/verify-deploy/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/support"
	"extension-scaffold/tools/pkg/validate"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	configPath := flag.String("config", "", "path to extension.env for post-deploy validation")
	step := flag.String("step", "all", "step to check: deploy, register, services, tee-version, tee-machine, test, all")
	jsonOutput := flag.Bool("json", false, "output as JSON")
	flag.Parse()

	s, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		fmt.Fprintf(os.Stderr, "Hint: check -a (addresses file) and -c (chain URL) flags\n")
		os.Exit(1)
	}

	report := &validate.Report{}

	addresses := map[string]common.Address{
		"TeeExtensionRegistry": s.Addresses.TeeExtensionRegistry,
		"TeeMachineRegistry":   s.Addresses.TeeMachineRegistry,
	}

	runStep := func(name string) bool {
		return *step == "all" || *step == name
	}

	if runStep("deploy") {
		validate.RegisterDeployChecks(report, s.ChainClient, s.Prv, addresses, *configPath)
	}

	if runStep("register") {
		validate.RegisterStubChecks(report, "register", "R1-R8", "Extension registration checks")
	}

	if runStep("services") {
		validate.RegisterStubChecks(report, "services", "S1-S11", "Service startup checks")
	}

	if runStep("tee-version") {
		validate.RegisterStubChecks(report, "tee-version", "V1-V6", "TEE version registration checks")
	}

	if runStep("tee-machine") {
		validate.RegisterStubChecks(report, "tee-machine", "T1-T10", "TEE machine registration checks")
	}

	if runStep("test") {
		validate.RegisterStubChecks(report, "test", "E1-E9", "Testing checks")
	}

	if *jsonOutput {
		report.PrintJSON()
	} else {
		report.Print()
	}

	if report.HasFailures() {
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd tools && go build ./cmd/verify-deploy/`
Expected: Build succeeds with no errors.

- [ ] **Step 3: Commit**

```bash
git add tools/cmd/verify-deploy/main.go
git commit -m "feat: add verify-deploy CLI for deployment diagnostics

Runs all Step 1 checks with --step deploy, stubs for Steps 2-6.
Supports --json output and --config for post-deploy validation."
```

---

## Task 5: Harden `support.go` — Fix Key Leak and Add Timeout

**Files:**
- Modify: `tools/pkg/support/support.go:88-109` (DefaultPrivateKey)
- Modify: `tools/pkg/support/support.go:179-193` (CheckTx)

- [ ] **Step 1: Remove private key print and fix fallback warning**

In `tools/pkg/support/support.go`, replace the `DefaultPrivateKey` function (lines 88-109):

Replace:
```go
func DefaultPrivateKey() (*ecdsa.PrivateKey, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
	}
	privKeyString := os.Getenv("PRIV_KEY")
	fmt.Printf("privKeyString: %s\n", privKeyString)

	if privKeyString == "" {
		fmt.Println("Warning: PRIV_KEY not set, falling back to hardcoded dev key (only works on local devnet)")
		return configs.PrvWithFunds, nil
	} else {
```

With:
```go
func DefaultPrivateKey() (*ecdsa.PrivateKey, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error loading .env file: %v\n", err)
	}
	privKeyString := os.Getenv("PRIV_KEY")

	if privKeyString == "" {
		fmt.Fprintln(os.Stderr, "WARNING: PRIV_KEY not set — using hardcoded Hardhat dev key")
		fmt.Fprintln(os.Stderr, "         This key only has funds on local devnets (Hardhat/Anvil)")
		return configs.PrvWithFunds, nil
	} else {
```

- [ ] **Step 2: Add timeout to `CheckTx`**

In `tools/pkg/support/support.go`, replace the `CheckTx` function (lines 179-193):

Replace:
```go
func CheckTx(tx *types.Transaction, client *ethclient.Client) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}
```

With:
```go
func CheckTx(tx *types.Transaction, client *ethclient.Client) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, errors.Errorf("transaction not mined within 2 minutes (tx: %s): %s", tx.Hash().Hex(), err)
	}
```

- [ ] **Step 3: Add `time` import if not already present**

Check imports in `support.go` — if `time` is not imported, add it. Current imports include `context` but not `time`.

In the import block, add `"time"`.

- [ ] **Step 4: Verify it compiles**

Run: `cd tools && go build ./...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add tools/pkg/support/support.go
git commit -m "fix: remove private key leak from stdout, add WaitMined timeout

- Remove fmt.Printf that printed PRIV_KEY to stdout (captured by tail -1)
- Move fallback warnings to stderr so they don't pollute stdout capture
- Add 2-minute timeout to CheckTx WaitMined calls"
```

---

## Task 6: Harden `deploy-contract` — Pre-flight Checks and `--preflight-only`

**Files:**
- Modify: `tools/cmd/deploy-contract/main.go`

- [ ] **Step 1: Add pre-flight checks and `--preflight-only` flag**

Replace the entire `tools/cmd/deploy-contract/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	"extension-scaffold/tools/pkg/validate"
	instrutils "extension-scaffold/tools/pkg/utils"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	outFile := flag.String("o", "", "write deployed address to this file (optional)")
	preflightOnly := flag.Bool("preflight-only", false, "run validation checks and exit without deploying")
	flag.Parse()

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Pre-flight validation ---
	deployer := crypto.PubkeyToAddress(testSupport.Prv.PublicKey)
	logger.Infof("Deployer:             %s", deployer.Hex())
	logger.Infof("Chain ID:             %s", testSupport.ChainID.String())
	logger.Infof("TeeExtensionRegistry: %s", testSupport.Addresses.TeeExtensionRegistry.Hex())
	logger.Infof("TeeMachineRegistry:   %s", testSupport.Addresses.TeeMachineRegistry.Hex())

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

	// --- Deploy ---
	logger.Infof("Deploying InstructionSender contract...")
	address, _, err := instrutils.DeployInstructionSender(testSupport)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("InstructionSender deployed at: %s", address.Hex())

	// Optionally write address to file for script consumption
	if *outFile != "" {
		os.MkdirAll(filepath.Dir(*outFile), 0755)
		os.WriteFile(*outFile, []byte(address.Hex()), 0644)
	}

	// Machine-readable output on stdout (for scripts)
	fmt.Println(address.Hex())
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd tools && go build ./cmd/deploy-contract/`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add tools/cmd/deploy-contract/main.go
git commit -m "feat: add pre-flight validation to deploy-contract

Checks registry addresses have code, deployer has funds, addresses not
zero before deploying. New --preflight-only flag runs checks and exits."
```

---

## Task 7: Add WaitMined Timeout to `DeployInstructionSender`

**Files:**
- Modify: `tools/pkg/utils/instructions.go:31-34`

- [ ] **Step 1: Add timeout context to WaitMined in DeployInstructionSender**

In `tools/pkg/utils/instructions.go`, replace lines 31-34:

Replace:
```go
	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed waiting for deployment: %s", err)
	}
```

With:
```go
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("deployment tx not mined within 2 minutes (tx: %s): %s", tx.Hash().Hex(), err)
	}
```

- [ ] **Step 2: Add `time` to imports**

Add `"time"` to the import block in `instructions.go`.

- [ ] **Step 3: Verify it compiles**

Run: `cd tools && go build ./...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add tools/pkg/utils/instructions.go
git commit -m "fix: add 2-minute timeout to DeployInstructionSender WaitMined

Prevents indefinite hang when chain RPC is unresponsive."
```

---

## Task 8: Solidity Constructor Hardening

**Files:**
- Modify: `contracts/InstructionSender.sol:39-45`

- [ ] **Step 1: Add constructor validation**

In `contracts/InstructionSender.sol`, replace the constructor (lines 39-45):

Replace:
```solidity
    constructor(
        ITeeExtensionRegistry _teeExtensionRegistry,
        ITeeMachineRegistry _teeMachineRegistry
    ) {
        TEE_EXTENSION_REGISTRY = _teeExtensionRegistry;
        TEE_MACHINE_REGISTRY = _teeMachineRegistry;
    }
```

With:
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

- [ ] **Step 2: Regenerate Go bindings**

Run: `./scripts/generate-bindings.sh`
Expected: Bindings regenerated successfully.

- [ ] **Step 3: Verify everything compiles**

Run: `cd tools && go build ./...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add contracts/InstructionSender.sol tools/pkg/contracts/
git commit -m "fix: add constructor validation to InstructionSender

Rejects zero addresses and addresses without deployed code at deploy
time. Last line of defense regardless of deployment tooling."
```

---

## Task 9: Harden `pre-build.sh`

**Files:**
- Modify: `scripts/pre-build.sh`

- [ ] **Step 1: Add stderr capture, regex validation, and preflight step**

Replace the entire `scripts/pre-build.sh`:

```bash
#!/usr/bin/env bash
# pre-build.sh — Deploy InstructionSender contract and register extension on-chain.
#
# Inputs (env vars):
#   ADDRESSES_FILE  — path to deployed-addresses.json (auto-detected if unset)
#   CHAIN_URL       — chain RPC URL (default: http://127.0.0.1:8545)
#   PRIV_KEY        — funded private key (default: Hardhat account)
#
# Outputs:
#   config/extension.env — EXTENSION_ID and INSTRUCTION_SENDER
#   config/deploy.log    — stderr from Go deploy commands
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${GREEN}[pre-build]${NC} $*"; }
step() { echo -e "\n${CYAN}=== Step $1: $2 ===${NC}"; }
die()  { echo -e "${RED}[pre-build] ERROR:${NC} $*" >&2; exit 1; }

# --- Load .env from project root (if present) ---
if [[ -f "$PROJECT_DIR/.env" ]]; then
    set -a
    source "$PROJECT_DIR/.env"
    set +a
fi

ADDRESSES_FILE="${ADDRESSES_FILE:-}"
# Resolve relative paths against PROJECT_DIR (not caller's cwd)
if [[ -n "$ADDRESSES_FILE" && "$ADDRESSES_FILE" != /* ]]; then
    ADDRESSES_FILE="$PROJECT_DIR/$ADDRESSES_FILE"
fi
CHAIN_URL="${CHAIN_URL:-http://127.0.0.1:8545}"
CONFIG_OUTPUT="$PROJECT_DIR/config/extension.env"
LOG_FILE="$PROJECT_DIR/config/deploy.log"

# Auto-detect addresses file
if [[ -z "$ADDRESSES_FILE" ]]; then
    LOCAL_MODE="${LOCAL_MODE:-true}"
    if [[ "$LOCAL_MODE" != "true" ]]; then
        # Non-local mode: use coston2 deployed addresses
        candidate="$PROJECT_DIR/config/coston2/deployed-addresses.json"
        if [[ -f "$candidate" ]]; then
            ADDRESSES_FILE="$(cd "$(dirname "$candidate")" && pwd)/$(basename "$candidate")"
        fi
    fi

    # Fall back to sim_dump candidates (local devnet)
    if [[ -z "$ADDRESSES_FILE" ]]; then
        for candidate in \
            "$PROJECT_DIR/../../e2e/docker/sim_dump/deployed-addresses.json" \
            "$PROJECT_DIR/../docker/sim_dump/deployed-addresses.json" \
            "$PROJECT_DIR/../../docker/sim_dump/deployed-addresses.json" \
            "$PROJECT_DIR/../../../docker/sim_dump/deployed-addresses.json"; do
            if [[ -f "$candidate" ]]; then
                ADDRESSES_FILE="$(cd "$(dirname "$candidate")" && pwd)/$(basename "$candidate")"
                break
            fi
        done
    fi

    [[ -n "$ADDRESSES_FILE" ]] || die "Cannot find deployed-addresses.json. Set ADDRESSES_FILE."
fi

[[ -f "$ADDRESSES_FILE" ]] || die "Addresses file not found: $ADDRESSES_FILE"

# Resolve to absolute path so it works after cd into tools/
ADDRESSES_FILE="$(cd "$(dirname "$ADDRESSES_FILE")" && pwd)/$(basename "$ADDRESSES_FILE")"

log "Chain URL:      $CHAIN_URL"
log "Addresses file: $ADDRESSES_FILE"

# --- Step 0: Pre-flight check ---
step 0 "Pre-flight check"
cd "$PROJECT_DIR/tools"
if ! go run ./cmd/deploy-contract -a "$ADDRESSES_FILE" -c "$CHAIN_URL" --preflight-only 2>&1; then
    die "Pre-flight check failed — fix the issues above before deploying"
fi

# --- Step 1: Generate Go bindings from Solidity contract ---
step 1 "Generate Go bindings"
"$SCRIPT_DIR/generate-bindings.sh" || die "Binding generation failed"

# --- Step 2: Deploy InstructionSender ---
step 2 "Deploy InstructionSender contract"
cd "$PROJECT_DIR/tools"
: > "$LOG_FILE"  # truncate log file
INSTRUCTION_SENDER=$(go run ./cmd/deploy-contract -a "$ADDRESSES_FILE" -c "$CHAIN_URL" 2>"$LOG_FILE" | tail -1) || {
    echo -e "${RED}Deploy failed. Logs:${NC}" >&2
    cat "$LOG_FILE" >&2
    die "Deploy failed — see output above"
}

# Validate captured address
[[ "$INSTRUCTION_SENDER" =~ ^0x[0-9a-fA-F]{40}$ ]] || {
    echo -e "${RED}deploy-contract output was not a valid address. Logs:${NC}" >&2
    cat "$LOG_FILE" >&2
    die "deploy-contract returned invalid address: '$INSTRUCTION_SENDER' (expected 0x + 40 hex chars)"
}

log "InstructionSender deployed at: $INSTRUCTION_SENDER"

# --- Step 3: Register extension ---
step 3 "Register extension on-chain"
EXTENSION_ID=$(go run ./cmd/register-extension -a "$ADDRESSES_FILE" -c "$CHAIN_URL" --instructionSender "$INSTRUCTION_SENDER" 2>>"$LOG_FILE" | tail -1) || {
    echo -e "${RED}Registration failed. Logs:${NC}" >&2
    cat "$LOG_FILE" >&2
    die "Registration failed — see output above"
}

# Validate captured extension ID
[[ "$EXTENSION_ID" =~ ^0x[0-9a-fA-F]{64}$ ]] || {
    echo -e "${RED}register-extension output was not a valid ID. Logs:${NC}" >&2
    cat "$LOG_FILE" >&2
    die "register-extension returned invalid ID: '$EXTENSION_ID' (expected 0x + 64 hex chars)"
}

log "Extension ID: $EXTENSION_ID"

# --- Step 4: Write config ---
step 4 "Write config"
mkdir -p "$(dirname "$CONFIG_OUTPUT")"
cat > "$CONFIG_OUTPUT" <<EOF
# Auto-generated by pre-build.sh — do not edit manually
EXTENSION_ID=$EXTENSION_ID
INSTRUCTION_SENDER=$INSTRUCTION_SENDER
EOF

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} Pre-build complete${NC}"
echo -e "${GREEN}========================================${NC}"
echo "  EXTENSION_ID         $EXTENSION_ID"
echo "  INSTRUCTION_SENDER   $INSTRUCTION_SENDER"
echo "  Config file          $CONFIG_OUTPUT"
echo "  Deploy log           $LOG_FILE"
```

- [ ] **Step 2: Verify script syntax**

Run: `bash -n scripts/pre-build.sh`
Expected: No syntax errors.

- [ ] **Step 3: Commit**

```bash
git add scripts/pre-build.sh
git commit -m "fix: harden pre-build.sh with stderr capture and validation

- Redirect Go command stderr to config/deploy.log instead of /dev/null
- Display log contents on failure for immediate diagnosis
- Regex-validate captured INSTRUCTION_SENDER and EXTENSION_ID
- Add Step 0 pre-flight check before deploying"
```

---

## Task 10: Claude Code Skill (`.claude/skills/verify-deploy/SKILL.md`)

**Files:**
- Create: `.claude/skills/verify-deploy/SKILL.md`

- [ ] **Step 1: Create the skill file**

Create `.claude/skills/verify-deploy/SKILL.md`:

```markdown
# Verify Deploy

Runs deployment verification checks and helps diagnose deployment failures.

## When to Use

The user wants to check their deployment setup, diagnose a failed deployment, or verify everything is correctly configured. They may say things like:
- "verify my deployment"
- "check my setup"
- "why is deploy failing"
- "is my config correct"
- "pre-flight check"
- "/verify-deploy"

## Steps to Execute

### Step 1: Determine deployment state

Read these files to understand where the user is in the deployment lifecycle:

1. `.env` — check CHAIN_URL, PRIV_KEY (set or not), LOCAL_MODE, ADDRESSES_FILE
2. `config/extension.env` — if it exists, deployment has been run before
3. `config/deploy.log` — if it exists, contains stderr from the last deployment attempt

### Step 2: Run the verification tool

```bash
cd tools && go run ./cmd/verify-deploy \
  -a <ADDRESSES_FILE> \
  -c <CHAIN_URL> \
  --config ../config/extension.env
```

Use the values from `.env` for ADDRESSES_FILE and CHAIN_URL. If `.env` doesn't set them, use the defaults:
- ADDRESSES_FILE: auto-detected (same logic as pre-build.sh)
- CHAIN_URL: http://127.0.0.1:8545

To check only a specific step: add `--step deploy` (or register, services, tee-version, tee-machine, test).

### Step 3: Interpret results

For each FAIL or WARN, explain what it means and how to fix it:

**D1/D2 — Registry address has no code:**
The addresses in deployed-addresses.json don't point to contracts on this chain.
Likely causes:
- Wrong CHAIN_URL (pointing to a different network than the addresses file)
- Wrong ADDRESSES_FILE (using coston2 addresses on local devnet or vice versa)
Fix: Check that CHAIN_URL and ADDRESSES_FILE in .env match the same network.

**D3 — Zero address in config:**
A required contract address is 0x000...000 in the addresses file.
Fix: Check deployed-addresses.json — a required entry is missing or unpopulated.

**D4 — Stale extension.env:**
The INSTRUCTION_SENDER from a previous deploy no longer has code on-chain.
Fix: Delete config/extension.env and re-run `scripts/pre-build.sh`.

**D5 — Deployer key/balance issues:**
Either PRIV_KEY is not set (using Hardhat dev key) or the account has no funds.
Fix: Set PRIV_KEY in .env to a funded account on the target network.

**D6 — Unexpected chain ID:**
Connected to a chain that may not match your intent.
Fix: Verify CHAIN_URL points to the intended network (Coston2: chain ID 114, local: 31337).

**D7 — Malformed extension.env values:**
The INSTRUCTION_SENDER or EXTENSION_ID in extension.env is not a valid hex value. This usually means a previous deploy command printed unexpected output that got captured by the script.
Fix: Delete config/extension.env and re-run `scripts/pre-build.sh`.

### Step 4: Check deployment logs for historical failures

If the user reports a failed deployment, or if verify-deploy shows FAILs:

1. Read `config/deploy.log` — this captures stderr from the Go deploy commands during `pre-build.sh`
2. Look for these patterns:
   - **Revert reasons** → on-chain rejection — likely wrong addresses or permissions (D1/D2)
   - **"insufficient funds"** → wrong key or unfunded account (D5)
   - **Connection errors** → wrong CHAIN_URL or RPC endpoint down (D6)
   - **Panic/stack traces** → bug in tooling (report to developer)
   - **"WARNING: PRIV_KEY not set"** → using dev key on non-local network (D5)
3. Cross-reference the log content with check IDs from the verification output
4. If `config/deploy.log` doesn't exist, the user hasn't run `pre-build.sh` yet — suggest running the verification tool with `--step deploy` as a preflight check

### Step 5: Detect compound misconfigurations

Check for these cross-cutting problems:
- **Coston2 addresses + localhost CHAIN_URL** = network mismatch (everything will fail silently)
- **LOCAL_MODE=false + no PRIV_KEY** = will use Hardhat key which has no Coston2 funds
- **extension.env exists + INSTRUCTION_SENDER has no code** = stale deployment, needs re-run

## Edge Cases Reference

Full edge case documentation: `EXTENSION-DEPLOYMENT-EDGE-CASES.md`
Design spec: `docs/superpowers/specs/2026-04-09-deployment-hardening-design.md`

## Important Notes

- The `.env` file contains secrets — do NOT print PRIV_KEY values. Only report whether it is set or not.
- Always read files before suggesting edits to them.
- The verification tool exits with code 1 if any check FAILs — this is expected, not a bug.
```

- [ ] **Step 2: Verify the skill file is well-formed**

Run: `head -5 .claude/skills/verify-deploy/SKILL.md`
Expected: Shows the skill header.

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/verify-deploy/SKILL.md
git commit -m "feat: add /verify-deploy Claude Code skill

Wraps cmd/verify-deploy with interpretive guidance, deployment log
analysis, and compound misconfiguration detection."
```

---

## Task 11: Final Verification

- [ ] **Step 1: Build everything**

Run: `cd tools && go build ./...`
Expected: All packages compile.

- [ ] **Step 2: Run all tests**

Run: `cd tools && go test ./pkg/validate/ -v`
Expected: All tests PASS.

- [ ] **Step 3: Verify pre-build.sh syntax**

Run: `bash -n scripts/pre-build.sh`
Expected: No syntax errors.

- [ ] **Step 4: Verify Solidity compiles (if forge is available)**

Run: `forge build` (if foundry is installed)
Expected: Compilation succeeds.

- [ ] **Step 5: Final commit if any cleanup needed**

Only if previous steps revealed issues that needed fixing.
