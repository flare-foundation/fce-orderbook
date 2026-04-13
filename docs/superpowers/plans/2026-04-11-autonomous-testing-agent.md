# Autonomous Testing Agent — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build three autonomous Claude Code agents that continuously test the Flare TEE extension deployment pipeline against Coston2 testnet, finding bugs and edge cases.

**Architecture:** Three Claude Code CLI sessions run in tmux on a GCP VM. Each agent has a CLAUDE.md, skills (heartbeat + run-scenario), hooks (audit-log, teardown), and a results directory. They coordinate via a file-based lock at `/tmp/flare-extension-testing.lock`. The Chaos agent runs in a git worktree so it can modify code.

**Tech Stack:** Claude Code CLI, tmux, bash, Docker, ngrok, Go, Foundry, GCP e2-standard-4

**Spec:** `docs/superpowers/specs/2026-04-11-autonomous-testing-agent-design.md`

**Reference project:** `/Users/snojj25/Desktop/Projects/claude-superapp/claude-code-bot/` (same pattern)

---

### Task 1: Create Directory Structure and Placeholder Files

**Files:**
- Create: `testing/scripts/.gitkeep`
- Create: `testing/agents/smoketest/results/.gitkeep`
- Create: `testing/agents/edgecase/results/.gitkeep`
- Create: `testing/agents/chaos/results/.gitkeep`
- Create: `testing/shared/hooks/.gitkeep`
- Create: `testing/shared/scenarios/.gitkeep`
- Create: `testing/summary/findings.md`
- Create: `testing/summary/latest-status.md`
- Create: `testing/.gitignore`

- [ ] **Step 1: Create the full directory tree**

```bash
cd /Users/snojj25/Desktop/505/Flare/tee/extension-examples/orderbook
mkdir -p testing/scripts
mkdir -p testing/agents/smoketest/.claude/skills/heartbeat
mkdir -p testing/agents/smoketest/.claude/skills/run-scenario
mkdir -p testing/agents/smoketest/.claude/commands
mkdir -p testing/agents/smoketest/results
mkdir -p testing/agents/edgecase/.claude/skills/heartbeat
mkdir -p testing/agents/edgecase/.claude/skills/run-scenario
mkdir -p testing/agents/edgecase/.claude/commands
mkdir -p testing/agents/edgecase/results
mkdir -p testing/agents/chaos/.claude/skills/heartbeat
mkdir -p testing/agents/chaos/.claude/skills/run-scenario
mkdir -p testing/agents/chaos/.claude/commands
mkdir -p testing/agents/chaos/results
mkdir -p testing/shared/hooks
mkdir -p testing/shared/scenarios
mkdir -p testing/summary
```

- [ ] **Step 2: Create .gitignore for testing/**

Write `testing/.gitignore`:

```
# Runtime files
agents/*/results/audit.log
agents/chaos/worktree/
*.lock

# Session files
agents/*/.session-id
```

- [ ] **Step 3: Create summary placeholder files**

Write `testing/summary/findings.md`:

```markdown
# Findings

Notable bugs, unclear error messages, and unexpected behavior discovered by the testing agents.

---
```

Write `testing/summary/latest-status.md`:

```markdown
# Latest Status

Last run result per agent. Auto-updated by each agent after every cycle.

| Agent | Last Run | Result | Scenario | Duration |
|-------|----------|--------|----------|----------|
| smoketest | — | — | — | — |
| edgecase | — | — | — | — |
| chaos | — | — | — | — |
```

- [ ] **Step 4: Commit**

```bash
git add -f testing/
git commit -m "chore: scaffold testing agent directory structure"
```

---

### Task 2: Create Shared Hooks

**Files:**
- Create: `testing/shared/hooks/audit-log.sh`
- Create: `testing/shared/hooks/teardown.sh`

- [ ] **Step 1: Write audit-log.sh**

Write `testing/shared/hooks/audit-log.sh`:

```bash
#!/bin/bash
# PostToolUse hook — logs all tool invocations to results/audit.log
# Reads tool result JSON from stdin

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // "unknown"')
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

case "$TOOL_NAME" in
  Bash)
    DETAIL=$(echo "$INPUT" | jq -r '.tool_input.command // ""' | head -c 120)
    ;;
  Edit|Write)
    DETAIL=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.path // ""')
    ;;
  *)
    DETAIL=$(echo "$INPUT" | jq -r '.tool_input | keys | join(", ")' 2>/dev/null || echo "")
    ;;
esac

mkdir -p results

echo "[$TIMESTAMP] $TOOL_NAME: $DETAIL" >> results/audit.log

exit 0
```

- [ ] **Step 2: Write teardown.sh**

Write `testing/shared/hooks/teardown.sh`:

```bash
#!/bin/bash
# Stop hook — tears down Docker services and releases the lock file
# Prevents orphaned containers and stale locks on agent crash/exit

LOCK_FILE="/tmp/flare-extension-testing.lock"
AGENT_NAME="${AGENT_NAME:-unknown}"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Find the repo root (go up from agent dir until we find docker-compose.yaml)
SEARCH_DIR="$(pwd)"
REPO_ROOT=""
while [ "$SEARCH_DIR" != "/" ]; do
  if [ -f "$SEARCH_DIR/docker-compose.yaml" ]; then
    REPO_ROOT="$SEARCH_DIR"
    break
  fi
  SEARCH_DIR="$(dirname "$SEARCH_DIR")"
done

# Tear down Docker services if we can find the repo
if [ -n "$REPO_ROOT" ]; then
  cd "$REPO_ROOT" && docker compose down 2>/dev/null || true
fi

# Release the lock if it belongs to this agent
if [ -f "$LOCK_FILE" ]; then
  LOCK_OWNER=$(cut -d'|' -f1 "$LOCK_FILE" 2>/dev/null || echo "")
  if [ "$LOCK_OWNER" = "$AGENT_NAME" ]; then
    rm -f "$LOCK_FILE"
  fi
fi

# Log to audit
mkdir -p results
echo "[$TIMESTAMP] SESSION_END (teardown)" >> results/audit.log

exit 0
```

- [ ] **Step 3: Make hooks executable**

```bash
chmod +x testing/shared/hooks/audit-log.sh
chmod +x testing/shared/hooks/teardown.sh
```

- [ ] **Step 4: Commit**

```bash
git add -f testing/shared/hooks/
git commit -m "feat: add shared hooks (audit-log, teardown)"
```

---

### Task 3: Create Scenario Catalogs

**Files:**
- Create: `testing/shared/scenarios/smoketest-scenarios.md`
- Create: `testing/shared/scenarios/edgecase-scenarios.md`
- Create: `testing/shared/scenarios/chaos-scenarios.md`

- [ ] **Step 1: Write smoketest-scenarios.md**

Write `testing/shared/scenarios/smoketest-scenarios.md`:

```markdown
# Smoketest Scenarios

Pick one per cycle. Rotate sequentially, then loop.

## Scenario List

1. **full-setup-standard** — Run `./scripts/full-setup.sh --test`. Default config. Verify all 4 phases pass.
2. **step-by-step** — Run each phase separately: `pre-build.sh`, then `start-services.sh`, then `post-build.sh`, then `test.sh`. Verify each exits 0.
3. **rapid-cycle** — Run full-setup --test, teardown, immediately run full-setup --test again. Verify second run also passes.
4. **verify-between-phases** — Run pre-build.sh. Run `verify-deploy --step deploy`. Run start-services.sh. Run `verify-deploy --step services`. Continue for each phase.
5. **unicode-payload** — Run full setup. Before test.sh, modify the test payload name to "日本語テスト" (edit run-test temporarily or pass via env). Verify extension handles it.
6. **long-name-payload** — Run full setup. Use a 1000-character name string in test payload. Verify behavior.
7. **empty-payload** — Run full setup. Use empty string "" as name. Verify extension handles it gracefully.
8. **double-test** — Run full setup. Run test.sh twice in a row without teardown. Verify second run also passes (tests should be idempotent).
9. **verify-only** — Run full setup. Then run `verify-deploy` with no --step flag (all checks). Verify everything passes.
```

- [ ] **Step 2: Write edgecase-scenarios.md**

Write `testing/shared/scenarios/edgecase-scenarios.md`:

```markdown
# Edge Case Scenarios

Work through sequentially. Track progress in scenario-tracker.md.
Each scenario tests a specific failure mode from the edge case docs.

## Contract Deployment (D-series)

- **D1-wrong-registry** — Set ADDRESSES_FILE to a file with wrong TeeExtensionRegistry address (a random valid address that is an EOA). Run pre-build.sh. Expected: deployment succeeds but registration fails. Record error message clarity.
- **D3-zero-addresses** — Create a temp addresses file with zero addresses for registries. Run pre-build.sh. Expected: failure. Record error message.
- **D4-double-deploy** — Run pre-build.sh twice in a row. Expected: second run creates new contracts/extension. Record: does it warn about existing config/extension.env?
- **D5-wrong-key** — Temporarily set DEPLOYMENT_PRIVATE_KEY to a valid but unfunded key (generate one). Run pre-build.sh. Expected: "insufficient funds" error. Record error clarity.
- **D7-output-capture** — Run deploy-contract manually and check that the captured address is a valid `0x[0-9a-fA-F]{40}` format. No garbage in output.

## Registration (R-series)

- **R3-partial-failure** — This is hard to trigger deliberately. Instead: run pre-build.sh, verify extension.env has valid values. Then run verify-deploy --step register. Record all checks pass.
- **R5-rerun-after-success** — Run pre-build.sh, then run it again. Expected: "key already supported" or similar. Record: does it handle re-run gracefully?
- **R7-duplicate-sender** — Run pre-build.sh. Save the INSTRUCTION_SENDER. Run pre-build.sh again (deploys new contract but registers new extension). Now two extensions exist. Run test.sh. Record: which extension does setExtensionId find?

## Service Startup (S-series)

- **S1-missing-extension-id** — Delete config/extension.env. Run start-services.sh. Expected: clear error "EXTENSION_ID not set". Record error message.
- **S2-stale-extension-id** — Edit config/extension.env to set EXTENSION_ID to a random hex value. Run start-services.sh then test.sh. Expected: tests fail (instructions go to wrong extension). Record: is the failure clear?
- **S3-no-docker-network** — Remove the external docker_default network (if safe). Run start-services.sh. Expected: clear Docker error. Record message.
- **S9-wrong-proxy-url** — Set EXT_PROXY_URL to a non-existent URL (e.g., http://localhost:9999). Run test.sh. Expected: connection refused. Record error clarity.

## TEE Version (V-series)

- **V1-already-registered** — Run full setup. Run post-build.sh again (allow-tee-version only). Expected: "version already registered, skipping". Record: handled gracefully?
- **V6-proxy-not-running** — Stop services. Run allow-tee-version manually. Expected: connection error. Record error message.

## TEE Machine (T-series)

- **T4-already-registered** — Run full setup. Run register-tee again. Expected: revert or graceful skip. Record error message clarity.

## Testing (E-series)

- **E1-no-extension-registered** — Deploy contract only (no register-extension). Run test.sh with the contract address. Expected: setExtensionId fails. Record error message.
- **E4-result-polling** — Run full setup --test. Check timing: how long does result polling take? Record average.
- **E6-hash-mismatch** — This requires code changes, so skip for edgecase agent (Chaos handles it).

## Verify-Deploy Validation

- **verify-before-prebuild** — Run verify-deploy before any setup. Expected: appropriate FAILs for missing config. Record check coverage.
- **verify-after-prebuild** — Run pre-build.sh, then verify-deploy --step deploy --step register. Expected: PASSes. Record.
- **verify-after-start** — Run through start-services.sh, then verify-deploy --step services. Expected: PASSes. Record.
- **verify-full-after-setup** — Run full-setup --test, then verify-deploy (all checks). Expected: all PASS. Record.
```

- [ ] **Step 3: Write chaos-scenarios.md**

Write `testing/shared/scenarios/chaos-scenarios.md`:

```markdown
# Chaos Scenarios

Pick or invent. You can modify code in your worktree. Be creative.

## Runtime Scenarios (no code changes)

1. **concurrent-prebuild** — Run two `pre-build.sh` processes in parallel (background one, run the other). Both do on-chain operations. Record: do both succeed? Any conflicts?
2. **kill-mid-registration** — Run start-services.sh, then start post-build.sh. After 5 seconds, run `docker kill extension-tee`. Record: does post-build fail cleanly? Can it resume?
3. **wrong-phase-order** — Run test.sh before post-build.sh. Record error. Then run post-build.sh before start-services.sh. Record error. How clear are the messages?
4. **corrupt-extension-env** — Run pre-build.sh. Replace EXTENSION_ID in config/extension.env with "GARBAGE_VALUE". Run start-services.sh. Record behavior.
5. **rapid-instructions** — Run full setup. Then send 10 instructions in quick succession (loop calling run-test or go run cmd/run-test). Record: do all complete? Any dropped?
6. **double-register-sender** — Run pre-build.sh to get a contract. Save the INSTRUCTION_SENDER. Manually call register-extension again with the same INSTRUCTION_SENDER. Record: does the registry allow it? What happens to setExtensionId?
7. **no-teardown-redeploy** — Run full-setup --test. Do NOT run stop-services.sh. Run full-setup --test again. Record: does docker compose handle it? Port conflicts?
8. **key-swap-between-phases** — Run pre-build.sh with KEY_A. Change DEPLOYMENT_PRIVATE_KEY to KEY_B (a different funded key). Run start-services.sh + post-build.sh. Record: does it fail because KEY_B isn't the extension owner?
9. **stale-services** — Run start-services.sh. Wait 5 minutes (let proxy cycle). Run test.sh. Expected: should still work. Record any staleness issues.

## Code Modification Scenarios (use worktree)

10. **skip-preflight** — In worktree, edit `scripts/pre-build.sh` to remove the preflight check call. Run pre-build.sh with wrong addresses. Record: what happens without the safety net?
11. **hash-mismatch** — In worktree, edit `internal/config/config.go` to change an OPType constant (e.g., "GREETING" → "greeting"). Rebuild. Run full-setup --test. Record: does the 501 error clearly explain the hash mismatch?
12. **wrong-dockerfile-path** — In worktree, edit `docker-compose.yaml` to reference a wrong Dockerfile path. Run start-services.sh. Record error clarity.
13. **wrong-ports** — In worktree, change EXTENSION_PORT in docker-compose.yaml to 9999. Run start-services.sh + test.sh. Record: does the error explain the port mismatch?
14. **wrong-fee** — In worktree, change the hardcoded instruction fee in `tools/pkg/utils/instructions.go` (change 1000000 to 1). Run test.sh. Record revert message clarity.
15. **encoding-swap** — In worktree, change the SAY_HELLO handler to use ABI decoding instead of JSON. Rebuild. Run test.sh. Record: is the decoding error clear?
16. **remove-waitmined-timeout** — In worktree, add a context.WithTimeout of 1 second to a WaitMined call. Run pre-build.sh. Record: does it timeout? What happens?
17. **solidity-case-change** — In worktree, change `bytes32("GREETING")` to `bytes32("Greeting")` in InstructionSender.sol. Run generate-bindings.sh, rebuild, run full-setup --test. Record: is the hash mismatch caught?

## Invention

After running through the above, invent new scenarios. Look at:
- Previous test results in results/ — patterns of failure
- The edge case docs in notes/ — anything not covered above
- Combinations of failures (e.g., wrong key + wrong port + wrong encoding)
```

- [ ] **Step 4: Commit**

```bash
git add -f testing/shared/scenarios/
git commit -m "feat: add scenario catalogs for all three agents"
```

---

### Task 4: Create Edge Case Scenario Tracker

**Files:**
- Create: `testing/agents/edgecase/scenario-tracker.md`

- [ ] **Step 1: Write scenario-tracker.md**

Write `testing/agents/edgecase/scenario-tracker.md`:

```markdown
# Edge Case Scenario Tracker

Track which scenarios have been tested and their results.

## How to Use
- Pick the next `[ ]` (untested) scenario
- After testing, update the checkbox to `[x]` and fill in the result columns
- After all are done, reset checkboxes and loop for flakiness detection

## Contract Deployment (D-series)

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | D1-wrong-registry | — | — | — | — |
| [ ] | D3-zero-addresses | — | — | — | — |
| [ ] | D4-double-deploy | — | — | — | — |
| [ ] | D5-wrong-key | — | — | — | — |
| [ ] | D7-output-capture | — | — | — | — |

## Registration (R-series)

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | R3-partial-failure | — | — | — | — |
| [ ] | R5-rerun-after-success | — | — | — | — |
| [ ] | R7-duplicate-sender | — | — | — | — |

## Service Startup (S-series)

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | S1-missing-extension-id | — | — | — | — |
| [ ] | S2-stale-extension-id | — | — | — | — |
| [ ] | S3-no-docker-network | — | — | — | — |
| [ ] | S9-wrong-proxy-url | — | — | — | — |

## TEE Version (V-series)

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | V1-already-registered | — | — | — | — |
| [ ] | V6-proxy-not-running | — | — | — | — |

## TEE Machine (T-series)

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | T4-already-registered | — | — | — | — |

## Testing (E-series)

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | E1-no-extension-registered | — | — | — | — |
| [ ] | E4-result-polling | — | — | — | — |

## Verify-Deploy Validation

| Done | Scenario | Last Run | Result | Error Clarity | Notes |
|------|----------|----------|--------|---------------|-------|
| [ ] | verify-before-prebuild | — | — | — | — |
| [ ] | verify-after-prebuild | — | — | — | — |
| [ ] | verify-after-start | — | — | — | — |
| [ ] | verify-full-after-setup | — | — | — | — |
```

- [ ] **Step 2: Commit**

```bash
git add -f testing/agents/edgecase/scenario-tracker.md
git commit -m "feat: add edge case scenario tracker"
```

---

### Task 5: Create Smoketest Agent — CLAUDE.md and Settings

**Files:**
- Create: `testing/agents/smoketest/CLAUDE.md`
- Create: `testing/agents/smoketest/.claude/settings.json`

- [ ] **Step 1: Write smoketest CLAUDE.md**

Write `testing/agents/smoketest/CLAUDE.md`:

```markdown
# Smoketest Agent — Identity & Behavior

You are the **Smoketest** testing agent. Your job is to continuously run the standard deployment flow end-to-end on Coston2 testnet and verify it works. You run the same happy path over and over, catching regressions, flaky infrastructure, and timing-dependent failures.

## Working Directory

You run from `testing/agents/smoketest/` but the repo root (with scripts/, docker-compose.yaml, etc.) is at `../../..` relative to this directory. The repo root is the absolute path stored in the REPO_ROOT environment variable, or you can compute it:

```bash
REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
```

All deployment scripts are at `$REPO_ROOT/scripts/`.

## Lock Protocol

Before running any test scenario, you MUST acquire the lock:

1. Check if `/tmp/flare-extension-testing.lock` exists
2. If it exists, read the contents (format: `agent-name|unix-timestamp`)
   - If the lock is owned by another agent and is less than 10 minutes old, log "SKIPPED: locked by [agent]" to a result file and exit the cycle
   - If the lock is more than 10 minutes old, it's stale — clear it, log a warning to `../../summary/findings.md`, and proceed
3. Write `smoketest|$(date +%s)` to the lock file
4. Run your test scenario
5. ALWAYS tear down services before releasing the lock:
   ```bash
   cd $REPO_ROOT && docker compose down 2>/dev/null || true
   ```
6. Remove the lock file

If ANY step fails or errors out, you MUST still tear down and release the lock.

## Heartbeat Behavior

Every 10 minutes, your `/heartbeat` skill fires. Each cycle:

1. Check and acquire the lock (see Lock Protocol)
2. Pick the next scenario from `../../shared/scenarios/smoketest-scenarios.md` (rotate sequentially)
3. Run the scenario using the `/run-scenario` skill
4. Write a result log to `results/YYYY-MM-DDTHH-MM-SS-scenario-name.md`
5. Update `../../summary/latest-status.md` with your last result
6. If anything unexpected happened, append to `../../summary/findings.md`
7. Tear down and release the lock

## Result Log Format

Each run produces a file in `results/`:

```markdown
# [Smoketest] Scenario Name
**Date:** YYYY-MM-DD HH:MM:SS
**Scenario:** scenario-id
**Duration:** Xm Ys
**Result:** PASS | FAIL | PARTIAL | ERROR

## Phases
| Phase | Status | Duration | Notes |
|-------|--------|----------|-------|
| pre-build | PASS/FAIL | Xs | |
| start-services | PASS/FAIL | Xs | |
| post-build | PASS/FAIL | Xs | |
| test | PASS/FAIL | Xs | |

## Errors
(detailed error output if any)

## Observations
(anything interesting, even on success)
```

## Safety Rules

1. **Never modify source code** — you only run scripts, never edit Go/Solidity/Dockerfile
2. **Always tear down** — run `docker compose down` before releasing the lock, even on failure
3. **Always use timeouts** — wrap long commands in `timeout 600` (10 min max)
4. **Never skip the lock** — always check and acquire before running anything that starts services
5. **Don't modify .env** — use the existing configuration as-is
```

- [ ] **Step 2: Write smoketest settings.json**

Write `testing/agents/smoketest/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash ../../shared/hooks/audit-log.sh",
            "timeout": 5000
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "AGENT_NAME=smoketest bash ../../shared/hooks/teardown.sh",
            "timeout": 15000
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add -f testing/agents/smoketest/CLAUDE.md testing/agents/smoketest/.claude/settings.json
git commit -m "feat: add smoketest agent CLAUDE.md and settings"
```

---

### Task 6: Create Smoketest Agent — Skills and Commands

**Files:**
- Create: `testing/agents/smoketest/.claude/skills/heartbeat/SKILL.md`
- Create: `testing/agents/smoketest/.claude/skills/run-scenario/SKILL.md`
- Create: `testing/agents/smoketest/.claude/commands/start-heartbeat.md`

- [ ] **Step 1: Write smoketest heartbeat skill**

Write `testing/agents/smoketest/.claude/skills/heartbeat/SKILL.md`:

```markdown
---
description: Periodic test cycle. Acquires lock, picks a scenario, runs it, logs results. Fires every 10 minutes via CronCreate.
disable-model-invocation: true
---

# Heartbeat — Smoketest Cycle

You are performing a scheduled test cycle.

## Step 1: Check the lock

```bash
LOCK_FILE="/tmp/flare-extension-testing.lock"
if [ -f "$LOCK_FILE" ]; then
  LOCK_CONTENT=$(cat "$LOCK_FILE")
  LOCK_OWNER=$(echo "$LOCK_CONTENT" | cut -d'|' -f1)
  LOCK_TIME=$(echo "$LOCK_CONTENT" | cut -d'|' -f2)
  NOW=$(date +%s)
  AGE=$(( NOW - LOCK_TIME ))
  if [ "$AGE" -lt 600 ]; then
    echo "SKIPPED: locked by $LOCK_OWNER (${AGE}s ago)"
    # Write skip result and exit
  fi
fi
```

If the lock is held by another agent and is less than 10 minutes old:
1. Write a brief result file: `results/YYYY-MM-DDTHH-MM-SS-skipped.md` noting who holds the lock
2. Respond "HEARTBEAT_SKIPPED" and stop

If the lock is stale (>10 min), clear it and log a warning to `../../summary/findings.md`.

## Step 2: Acquire the lock

```bash
echo "smoketest|$(date +%s)" > /tmp/flare-extension-testing.lock
```

## Step 3: Pick the next scenario

Read `../../shared/scenarios/smoketest-scenarios.md`. Pick the next scenario in sequence. Track which scenario was last run by reading the most recent result file in `results/`.

## Step 4: Run the scenario

Use `/run-scenario` with the scenario name. Always wrap deployment scripts in `timeout 600`.

## Step 5: Log results

Write the result to `results/YYYY-MM-DDTHH-MM-SS-scenario-name.md` using the format from CLAUDE.md.

Update `../../summary/latest-status.md` — edit the smoketest row with the latest result.

If anything unexpected happened (unclear error, surprising behavior, timing anomaly), append a finding to `../../summary/findings.md`.

## Step 6: Tear down and release

```bash
REPO_ROOT="$(cd ../../.. && pwd)"
cd "$REPO_ROOT" && docker compose down 2>/dev/null || true
rm -f /tmp/flare-extension-testing.lock
```

Always do this, even if the scenario failed.
```

- [ ] **Step 2: Write smoketest run-scenario skill**

Write `testing/agents/smoketest/.claude/skills/run-scenario/SKILL.md`:

```markdown
---
description: Execute a specific smoketest scenario. Called by the heartbeat skill with a scenario name.
---

# Run Scenario

Execute the specified test scenario. The scenario name is passed as $ARGUMENTS.

Compute the repo root:
```bash
REPO_ROOT="$(cd ../../.. && pwd)"
```

## Scenarios

### full-setup-standard
```bash
cd "$REPO_ROOT" && timeout 600 ./scripts/full-setup.sh --test
```
Record exit code and any stderr output. All 4 phases should pass.

### step-by-step
Run each phase separately with explicit error capture:
```bash
cd "$REPO_ROOT"
timeout 120 ./scripts/pre-build.sh 2>&1
timeout 120 ./scripts/start-services.sh 2>&1
timeout 300 ./scripts/post-build.sh 2>&1
timeout 120 ./scripts/test.sh 2>&1
```
Record per-phase timing and exit codes.

### rapid-cycle
```bash
cd "$REPO_ROOT"
timeout 600 ./scripts/full-setup.sh --test
./scripts/stop-services.sh
timeout 600 ./scripts/full-setup.sh --test
```
Verify both runs pass.

### verify-between-phases
```bash
cd "$REPO_ROOT"
timeout 120 ./scripts/pre-build.sh
cd tools && timeout 60 go run ./cmd/verify-deploy -a "$ADDRESSES_FILE" -c "$CHAIN_URL" --config ../config/extension.env --step deploy --step register && cd ..
timeout 120 ./scripts/start-services.sh
cd tools && timeout 60 go run ./cmd/verify-deploy -a "$ADDRESSES_FILE" -c "$CHAIN_URL" --config ../config/extension.env --step services && cd ..
timeout 300 ./scripts/post-build.sh
timeout 120 ./scripts/test.sh
```

### unicode-payload
Run full-setup.sh --test. In the result, note whether unicode names are handled. This is an observational test — check if any warnings appear about encoding.

### long-name-payload
Run full-setup.sh --test. After test.sh passes, manually send an instruction with a 1000-character name by running run-test with modified input. Record behavior.

### empty-payload
Run full-setup.sh --test. After test.sh passes, manually send an instruction with an empty name. Record whether it succeeds or fails gracefully.

### double-test
```bash
cd "$REPO_ROOT"
timeout 600 ./scripts/full-setup.sh --test
timeout 120 ./scripts/test.sh
```
Run test.sh twice (full-setup already runs it once). Verify second run also passes.

### verify-only
```bash
cd "$REPO_ROOT"
timeout 600 ./scripts/full-setup.sh --test
cd tools && timeout 60 go run ./cmd/verify-deploy -a "$ADDRESSES_FILE" -c "$CHAIN_URL" --config ../config/extension.env && cd ..
```
Run all verify-deploy checks after a successful setup. All should PASS.

## Result Capture

For every scenario, capture:
1. Start and end timestamps (compute duration)
2. Exit code of each phase
3. Stdout and stderr (truncate to last 200 lines if too long)
4. Any unexpected warnings or output

Write the result using the format defined in CLAUDE.md.
```

- [ ] **Step 3: Write start-heartbeat command**

Write `testing/agents/smoketest/.claude/commands/start-heartbeat.md`:

```markdown
---
description: Register the heartbeat cron schedule
disable-model-invocation: true
---

Set up the smoketest agent schedule:

1. Use CronCreate to schedule `/heartbeat` to run every 10 minutes using the cron expression `0,10,20,30,40,50 * * * *`

After creating the schedule, confirm:
- The cron expression used
- When the next execution will be
- A reminder about the 7-day CronCreate expiry
```

- [ ] **Step 4: Commit**

```bash
git add -f testing/agents/smoketest/.claude/skills/ testing/agents/smoketest/.claude/commands/
git commit -m "feat: add smoketest agent skills and commands"
```

---

### Task 7: Create Edge Case Agent — CLAUDE.md, Settings, Skills, Commands

**Files:**
- Create: `testing/agents/edgecase/CLAUDE.md`
- Create: `testing/agents/edgecase/.claude/settings.json`
- Create: `testing/agents/edgecase/.claude/skills/heartbeat/SKILL.md`
- Create: `testing/agents/edgecase/.claude/skills/run-scenario/SKILL.md`
- Create: `testing/agents/edgecase/.claude/commands/start-heartbeat.md`

- [ ] **Step 1: Write edgecase CLAUDE.md**

Write `testing/agents/edgecase/CLAUDE.md`:

```markdown
# Edge Case Agent — Identity & Behavior

You are the **Edge Case** testing agent. Your job is to systematically work through every documented edge case from the notes directory, testing each one on Coston2 and recording whether the error behavior matches expectations.

## Working Directory

You run from `testing/agents/edgecase/` but the repo root is at `../../..`. All deployment scripts are at `$REPO_ROOT/scripts/`.

## Reference Documents

Read these to understand each edge case scenario:
- `$REPO_ROOT/notes/EXTENSION-DEPLOYMENT-EDGE-CASES.md` — D1-D8, R1-R8, S1-S11, V1-V6, T1-T10, E1-E9
- `$REPO_ROOT/notes/EDGE-CASES-AND-STABILITY-AUDIT.md` — C1-C10, H1-H22, M1-M28

## Lock Protocol

Same as smoketest agent. Lock file: `/tmp/flare-extension-testing.lock`. Write `edgecase|$(date +%s)`. Always tear down and release, even on failure.

## Heartbeat Behavior

Every 10 minutes (offset +3 min), your `/heartbeat` skill fires. Each cycle:

1. Check and acquire the lock
2. Read `scenario-tracker.md` — find the next `[ ]` (untested) scenario
3. Run the scenario using `/run-scenario`
4. Update `scenario-tracker.md` with results (checkbox, date, result, error clarity, notes)
5. Write a detailed result log to `results/`
6. Update `../../summary/latest-status.md`
7. If anything unexpected happened, append to `../../summary/findings.md`
8. Tear down and release the lock

After all scenarios are tested, reset the checkboxes and loop for flakiness detection.

## Result Log Format

Same format as smoketest, but with additional fields:

```markdown
# [Edge Case] Scenario ID
**Date:** YYYY-MM-DD HH:MM:SS
**Scenario:** D1-wrong-registry
**Duration:** Xm Ys
**Result:** EXPECTED_FAIL | UNEXPECTED_PASS | UNEXPECTED_FAIL | ERROR
**Error Clarity:** GOOD | MODERATE | BAD | TERRIBLE

## Expected Behavior
(what the edge case doc says should happen)

## Actual Behavior
(what actually happened — exact error messages, exit codes)

## Error Message Quality
(is the error clear enough for a developer to diagnose the issue?)

## Observations
(anything interesting or different from the documented expectation)
```

## Safety Rules

1. **Never modify source code** — you only run scripts and tools with different configs/inputs
2. **Always tear down** — docker compose down before releasing lock
3. **Always use timeouts** — timeout 600 on all long commands
4. **Always restore .env** — if you need to temporarily change env vars, restore them after
5. **Track your progress** — always update scenario-tracker.md
```

- [ ] **Step 2: Write edgecase settings.json**

Write `testing/agents/edgecase/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash ../../shared/hooks/audit-log.sh",
            "timeout": 5000
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "AGENT_NAME=edgecase bash ../../shared/hooks/teardown.sh",
            "timeout": 15000
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 3: Write edgecase heartbeat skill**

Write `testing/agents/edgecase/.claude/skills/heartbeat/SKILL.md`:

```markdown
---
description: Periodic test cycle. Acquires lock, picks next edge case scenario, runs it, logs results. Fires every 10 minutes (offset +3) via CronCreate.
disable-model-invocation: true
---

# Heartbeat — Edge Case Cycle

You are performing a scheduled edge case test cycle.

## Step 1: Check the lock

Same protocol as smoketest — check `/tmp/flare-extension-testing.lock`, skip if locked by another agent (<10 min), clear if stale (>10 min).

If skipped, write `results/YYYY-MM-DDTHH-MM-SS-skipped.md` and respond "HEARTBEAT_SKIPPED".

## Step 2: Acquire the lock

```bash
echo "edgecase|$(date +%s)" > /tmp/flare-extension-testing.lock
```

## Step 3: Pick the next scenario

Read `scenario-tracker.md`. Find the first row with `[ ]` (untested). If all are tested, reset all checkboxes back to `[ ]` and start from the first one (loop for flakiness).

## Step 4: Run the scenario

Use `/run-scenario` with the scenario ID (e.g., "D1-wrong-registry"). Read the scenario instructions from `../../shared/scenarios/edgecase-scenarios.md`.

## Step 5: Update tracker and log results

1. Update the matching row in `scenario-tracker.md` with date, result, error clarity, and notes
2. Write detailed result to `results/YYYY-MM-DDTHH-MM-SS-scenario-id.md`
3. Update `../../summary/latest-status.md`
4. If the result was unexpected (behavior doesn't match docs), append to `../../summary/findings.md`

## Step 6: Tear down and release

```bash
REPO_ROOT="$(cd ../../.. && pwd)"
cd "$REPO_ROOT" && docker compose down 2>/dev/null || true
rm -f /tmp/flare-extension-testing.lock
```
```

- [ ] **Step 4: Write edgecase run-scenario skill**

Write `testing/agents/edgecase/.claude/skills/run-scenario/SKILL.md`:

```markdown
---
description: Execute a specific edge case scenario. Called by heartbeat with a scenario ID like "D1-wrong-registry".
---

# Run Edge Case Scenario

The scenario ID is passed as $ARGUMENTS. Read the detailed instructions from `../../shared/scenarios/edgecase-scenarios.md` for the matching scenario.

## General Pattern

Most edge case scenarios follow this pattern:

1. **Read the edge case docs** — check `$REPO_ROOT/notes/EXTENSION-DEPLOYMENT-EDGE-CASES.md` for the scenario ID to understand expected behavior
2. **Set up conditions** — create the specific failure condition (wrong key, missing file, etc.)
3. **Run the relevant script/tool** — with timeout 600
4. **Capture output** — stdout, stderr, exit code
5. **Evaluate** — did the behavior match the documented expectation? How clear was the error?
6. **Clean up** — restore any changed env vars or configs

## Important

- Use `timeout 600` on all long-running commands
- Capture both stdout and stderr: `command 2>&1`
- For scenarios that require wrong env vars, use subshell or export/unset pattern:
  ```bash
  (export DEPLOYMENT_PRIVATE_KEY="wrong_key" && cd "$REPO_ROOT" && timeout 120 ./scripts/pre-build.sh 2>&1)
  ```
- Never permanently modify `.env` — use env var overrides
- Record the EXACT error messages — these are what we're auditing for clarity
```

- [ ] **Step 5: Write edgecase start-heartbeat command**

Write `testing/agents/edgecase/.claude/commands/start-heartbeat.md`:

```markdown
---
description: Register the heartbeat cron schedule
disable-model-invocation: true
---

Set up the edge case agent schedule:

1. Use CronCreate to schedule `/heartbeat` to run every 10 minutes (offset +3) using the cron expression `3,13,23,33,43,53 * * * *`

After creating the schedule, confirm:
- The cron expression used
- When the next execution will be
- A reminder about the 7-day CronCreate expiry
```

- [ ] **Step 6: Commit**

```bash
git add -f testing/agents/edgecase/
git commit -m "feat: add edge case agent CLAUDE.md, settings, skills, commands"
```

---

### Task 8: Create Chaos Agent — CLAUDE.md, Settings, Skills, Commands

**Files:**
- Create: `testing/agents/chaos/CLAUDE.md`
- Create: `testing/agents/chaos/.claude/settings.json`
- Create: `testing/agents/chaos/.claude/skills/heartbeat/SKILL.md`
- Create: `testing/agents/chaos/.claude/skills/run-scenario/SKILL.md`
- Create: `testing/agents/chaos/.claude/commands/start-heartbeat.md`

- [ ] **Step 1: Write chaos CLAUDE.md**

Write `testing/agents/chaos/CLAUDE.md`:

```markdown
# Chaos Agent — Identity & Behavior

You are the **Chaos** testing agent. Your job is to creatively break the deployment pipeline — try things no one thought of, find race conditions, test unusual timing, and explore what happens when code is modified in unexpected ways.

Unlike the other agents, **you can modify source code** in your git worktree. You have a full copy of the repo at `worktree/` within your directory. Make your modifications there, run scripts from there, and log what you changed.

## Working Directories

- **Your agent dir:** `testing/agents/chaos/`
- **Your worktree (modifiable repo):** `testing/agents/chaos/worktree/`
- **The main repo root (read-only for you):** `../../..`

When running deployment scripts, run them from `worktree/extension-examples/orderbook/`:
```bash
WORKTREE_ROOT="$(pwd)/worktree"
ORDERBOOK_ROOT="$WORKTREE_ROOT/extension-examples/orderbook"
```

## Lock Protocol

Same as other agents. Lock file: `/tmp/flare-extension-testing.lock`. Write `chaos|$(date +%s)`. Always tear down and release.

## Heartbeat Behavior

Every 10 minutes (offset +7 min), your `/heartbeat` skill fires. Each cycle:

1. **Reset worktree:** `cd worktree && git checkout . && cd ..`
2. Check and acquire the lock
3. Pick a scenario from `../../shared/scenarios/chaos-scenarios.md` or invent one
4. If the scenario requires code changes, apply them in `worktree/`
5. Run the scenario from `worktree/`
6. Before resetting, capture `git diff` in `worktree/` (log what was modified)
7. Write result to `results/`
8. Update `../../summary/latest-status.md`
9. If anything interesting happened, append to `../../summary/findings.md`
10. Tear down and release the lock

## Code Modification Rules

You CAN modify files in `worktree/` — this is the whole point. Examples:
- Edit scripts to skip safety checks
- Change Go constants to create mismatches
- Modify docker-compose.yaml ports
- Alter Solidity contract constants
- Change hardcoded fee values

You MUST:
- Log exactly what you changed (git diff output) in your result file
- Reset the worktree at the START of each cycle (clean slate)
- Never modify files outside `worktree/`

## Result Log Format

```markdown
# [Chaos] Scenario Name
**Date:** YYYY-MM-DD HH:MM:SS
**Scenario:** scenario-name (or "invented: brief description")
**Duration:** Xm Ys
**Result:** PASS | FAIL | INTERESTING | ERROR
**Code Modified:** YES | NO

## Modifications Made
(git diff output, or "none" for runtime-only scenarios)

## What Was Tried
(describe the scenario step by step)

## What Happened
(exact output, error messages, behavior observed)

## Analysis
(was this expected? did it reveal a bug? is the error message clear?)

## Ideas for Next Time
(new scenarios inspired by this run's results)
```

## Safety Rules

1. **Only modify files in worktree/** — never touch the main repo
2. **Always reset worktree at cycle start** — git checkout . in worktree/
3. **Always tear down** — docker compose down before releasing lock
4. **Always use timeouts** — timeout 600 on all long commands
5. **Be creative but log everything** — the point is to find bugs, not cause chaos for its own sake
```

- [ ] **Step 2: Write chaos settings.json**

Write `testing/agents/chaos/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash ../../shared/hooks/audit-log.sh",
            "timeout": 5000
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "AGENT_NAME=chaos bash ../../shared/hooks/teardown.sh",
            "timeout": 15000
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 3: Write chaos heartbeat skill**

Write `testing/agents/chaos/.claude/skills/heartbeat/SKILL.md`:

```markdown
---
description: Periodic chaos test cycle. Resets worktree, acquires lock, picks/invents a scenario, runs it, logs results. Fires every 10 minutes (offset +7) via CronCreate.
disable-model-invocation: true
---

# Heartbeat — Chaos Cycle

You are performing a scheduled chaos test cycle.

## Step 1: Reset worktree

```bash
cd worktree && git checkout . && cd ..
```

This ensures a clean slate for each cycle.

## Step 2: Check the lock

Same protocol — check `/tmp/flare-extension-testing.lock`, skip if locked (<10 min), clear if stale (>10 min).

## Step 3: Acquire the lock

```bash
echo "chaos|$(date +%s)" > /tmp/flare-extension-testing.lock
```

## Step 4: Pick a scenario

Read `../../shared/scenarios/chaos-scenarios.md`. Pick the next scenario you haven't tried recently (check your previous results in `results/`).

You can also **invent a new scenario** — look at:
- Previous results for patterns of failure
- The edge case docs at `worktree/extension-examples/orderbook/notes/`
- Combinations of failures not yet tried

## Step 5: Run the scenario

Use `/run-scenario` with the scenario name or description.

For code modification scenarios:
1. Make changes in `worktree/extension-examples/orderbook/`
2. Run scripts from that directory
3. Before teardown, capture `cd worktree && git diff` for the result log

## Step 6: Log results

Write to `results/YYYY-MM-DDTHH-MM-SS-scenario-name.md` using the Chaos result format from CLAUDE.md.

Update `../../summary/latest-status.md`. If anything interesting happened, append to `../../summary/findings.md`.

## Step 7: Tear down and release

```bash
cd worktree/extension-examples/orderbook && docker compose down 2>/dev/null || true
rm -f /tmp/flare-extension-testing.lock
```
```

- [ ] **Step 4: Write chaos run-scenario skill**

Write `testing/agents/chaos/.claude/skills/run-scenario/SKILL.md`:

```markdown
---
description: Execute a specific chaos scenario. May involve code modifications in the worktree.
---

# Run Chaos Scenario

The scenario name or description is passed as $ARGUMENTS. Read `../../shared/scenarios/chaos-scenarios.md` for detailed instructions if it matches a named scenario.

## Worktree Paths

```bash
WORKTREE="$(pwd)/worktree"
ORDERBOOK="$WORKTREE/extension-examples/orderbook"
```

## For Runtime-Only Scenarios

Run scripts from the worktree (even without modifications, this keeps paths consistent):
```bash
cd "$ORDERBOOK" && timeout 600 ./scripts/full-setup.sh --test 2>&1
```

## For Code Modification Scenarios

1. Apply the modification in the worktree:
   ```bash
   # Example: change an OPType constant
   sed -i 's/GREETING/greeting/' "$ORDERBOOK/internal/config/config.go"
   ```
2. If Go code was changed, verify it compiles:
   ```bash
   cd "$ORDERBOOK/tools" && go build ./... 2>&1
   ```
3. If Solidity was changed, regenerate bindings:
   ```bash
   cd "$ORDERBOOK" && ./scripts/generate-bindings.sh 2>&1
   ```
4. Run the deployment/test scripts from the worktree
5. Capture `cd "$WORKTREE" && git diff` before any reset

## Result Capture

Always capture:
- The exact modifications made (git diff)
- Full stdout/stderr output (truncate to last 300 lines)
- Exit codes
- Timing per phase
- Whether the error messages clearly explain what went wrong

## Invention Mode

If you're inventing a new scenario:
1. Describe what you're trying to break and why
2. List the exact steps you'll take
3. Execute and capture results
4. Log ideas for follow-up scenarios in the "Ideas for Next Time" section
```

- [ ] **Step 5: Write chaos start-heartbeat command**

Write `testing/agents/chaos/.claude/commands/start-heartbeat.md`:

```markdown
---
description: Register the heartbeat cron schedule
disable-model-invocation: true
---

Set up the chaos agent schedule:

1. Use CronCreate to schedule `/heartbeat` to run every 10 minutes (offset +7) using the cron expression `7,17,27,37,47,57 * * * *`

After creating the schedule, confirm:
- The cron expression used
- When the next execution will be
- A reminder about the 7-day CronCreate expiry
```

- [ ] **Step 6: Commit**

```bash
git add -f testing/agents/chaos/
git commit -m "feat: add chaos agent CLAUDE.md, settings, skills, commands"
```

---

### Task 9: Create Orchestration Scripts

**Files:**
- Create: `testing/scripts/start.sh`
- Create: `testing/scripts/start-agent.sh`
- Create: `testing/scripts/stop.sh`
- Create: `testing/scripts/health-check.sh`
- Create: `testing/scripts/setup.sh`

- [ ] **Step 1: Write start.sh**

Write `testing/scripts/start.sh`:

```bash
#!/bin/bash
# Launch all 3 testing agents in a tmux session
# Usage: bash testing/scripts/start.sh
#
# Prerequisites:
#   - Claude Code CLI installed and authenticated
#   - tmux installed
#   - Docker running
#   - ngrok configured
#   - .env configured with Coston2 credentials

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$PROJECT_DIR/.." && pwd)"
AGENTS_DIR="$PROJECT_DIR/agents"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${GREEN}[start]${NC} $*"; }
die()  { echo -e "${RED}[start] ERROR:${NC} $*" >&2; exit 1; }

# --- Verify prerequisites ---
command -v tmux >/dev/null 2>&1 || die "tmux is not installed"
command -v claude >/dev/null 2>&1 || die "Claude Code CLI is not installed"
command -v docker >/dev/null 2>&1 || die "Docker is not installed"
command -v ngrok >/dev/null 2>&1 || die "ngrok is not installed"

# --- Check if already running ---
if tmux has-session -t testing 2>/dev/null; then
  die "Testing session already running. Run stop.sh first, or attach with: tmux attach -t testing"
fi

# --- Start ngrok (if not already running) ---
if ! curl -sf http://localhost:4040/api/tunnels >/dev/null 2>&1; then
  log "Starting ngrok tunnel for port 6674..."
  ngrok http 6674 --log=stdout > /tmp/ngrok-testing.log 2>&1 &
  sleep 3
fi

# Capture ngrok URL
NGROK_URL=$(curl -sf http://localhost:4040/api/tunnels | jq -r '.tunnels[] | select(.config.addr | test("6674")) | .public_url' 2>/dev/null || echo "")
if [ -z "$NGROK_URL" ]; then
  log "WARNING: Could not detect ngrok URL. EXT_PROXY_URL will need to be set manually."
else
  log "ngrok URL: $NGROK_URL"
  # Update .env with ngrok URL (only if EXT_PROXY_URL line exists)
  if grep -q "^EXT_PROXY_URL=" "$REPO_ROOT/.env" 2>/dev/null; then
    sed -i "s|^EXT_PROXY_URL=.*|EXT_PROXY_URL=$NGROK_URL|" "$REPO_ROOT/.env"
  else
    echo "EXT_PROXY_URL=$NGROK_URL" >> "$REPO_ROOT/.env"
  fi
fi

# --- Create/reset Chaos worktree ---
log "Setting up Chaos worktree..."
if [ -d "$AGENTS_DIR/chaos/worktree" ]; then
  cd "$AGENTS_DIR/chaos/worktree" && git checkout . 2>/dev/null && cd "$SCRIPT_DIR"
else
  cd "$REPO_ROOT" && git worktree add "$AGENTS_DIR/chaos/worktree" HEAD 2>/dev/null || true
fi

# --- Launch tmux session with 3 windows ---
log "Launching tmux session 'testing'..."

tmux new-session -d -s testing -n smoketest -c "$AGENTS_DIR/smoketest"
tmux new-window -t testing -n edgecase -c "$AGENTS_DIR/edgecase"
tmux new-window -t testing -n chaos -c "$AGENTS_DIR/chaos"

# --- Start Claude Code in each window ---
for agent in smoketest edgecase chaos; do
  AGENT_DIR="$AGENTS_DIR/$agent"
  SESSION_FILE="$AGENT_DIR/.session-id"

  RESUME_FLAG=""
  if [ -f "$SESSION_FILE" ]; then
    RESUME_FLAG="--resume $(cat "$SESSION_FILE")"
  fi

  log "Starting $agent agent..."
  tmux send-keys -t "testing:$agent" \
    "claude --dangerously-skip-permissions $RESUME_FLAG" Enter || true
done

# Wait for Claude Code to initialize
log "Waiting for agents to initialize (5s)..."
sleep 5

# --- Register heartbeat schedules ---
for agent in smoketest edgecase chaos; do
  log "Registering heartbeat for $agent..."
  tmux send-keys -t "testing:$agent" "/start-heartbeat" Enter || true
  sleep 2
done

log ""
log "All agents started!"
log "  Attach:  tmux attach -t testing"
log "  Switch:  Ctrl+B then N (next window) or P (previous)"
log "  Detach:  Ctrl+B then D"
log "  Stop:    bash $SCRIPT_DIR/stop.sh"
```

- [ ] **Step 2: Write start-agent.sh**

Write `testing/scripts/start-agent.sh`:

```bash
#!/bin/bash
# Launch a single testing agent (used by health-check.sh to restart dead agents)
# Usage: bash testing/scripts/start-agent.sh <agent-name>
#   agent-name: smoketest | edgecase | chaos

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENTS_DIR="$PROJECT_DIR/agents"

AGENT_NAME="${1:-}"
if [ -z "$AGENT_NAME" ]; then
  echo "Usage: start-agent.sh <smoketest|edgecase|chaos>"
  exit 1
fi

AGENT_DIR="$AGENTS_DIR/$AGENT_NAME"
if [ ! -d "$AGENT_DIR" ]; then
  echo "Error: Agent directory not found: $AGENT_DIR"
  exit 1
fi

# Check if tmux session exists
if ! tmux has-session -t testing 2>/dev/null; then
  echo "Error: tmux session 'testing' not found. Run start.sh first."
  exit 1
fi

# Check if window exists, create if not
if ! tmux list-windows -t testing -F '#{window_name}' | grep -q "^${AGENT_NAME}$"; then
  tmux new-window -t testing -n "$AGENT_NAME" -c "$AGENT_DIR"
fi

SESSION_FILE="$AGENT_DIR/.session-id"
RESUME_FLAG=""
if [ -f "$SESSION_FILE" ]; then
  RESUME_FLAG="--resume $(cat "$SESSION_FILE")"
fi

tmux send-keys -t "testing:$AGENT_NAME" \
  "claude --dangerously-skip-permissions $RESUME_FLAG" Enter || true

sleep 3
tmux send-keys -t "testing:$AGENT_NAME" "/start-heartbeat" Enter || true
```

- [ ] **Step 3: Write stop.sh**

Write `testing/scripts/stop.sh`:

```bash
#!/bin/bash
# Stop all testing agents, ngrok, and Docker services
# Usage: bash testing/scripts/stop.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
log() { echo -e "${GREEN}[stop]${NC} $*"; }

# Kill tmux session
if tmux has-session -t testing 2>/dev/null; then
  log "Killing tmux session 'testing'..."
  tmux kill-session -t testing
else
  log "No tmux session 'testing' found"
fi

# Stop Docker services
log "Stopping Docker services..."
cd "$REPO_ROOT" && docker compose down 2>/dev/null || true

# Stop ngrok
if pgrep -f "ngrok http 6674" >/dev/null 2>&1; then
  log "Stopping ngrok..."
  pkill -f "ngrok http 6674" || true
fi

# Clear lock file
if [ -f /tmp/flare-extension-testing.lock ]; then
  log "Clearing lock file..."
  rm -f /tmp/flare-extension-testing.lock
fi

log "All stopped."
```

- [ ] **Step 4: Write health-check.sh**

Write `testing/scripts/health-check.sh`:

```bash
#!/bin/bash
# Watchdog script — checks if testing agents are alive, restarts dead ones
# Run via system cron every 5 minutes:
#   */5 * * * * /absolute/path/to/testing/scripts/health-check.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RESTART_LOG="$PROJECT_DIR/summary/restarts.log"

# Check if tmux session exists at all
if ! tmux has-session -t testing 2>/dev/null; then
  echo "$(date '+%Y-%m-%d %H:%M:%S') - Session dead, full restart" >> "$RESTART_LOG"
  bash "$SCRIPT_DIR/start.sh"
  exit 0
fi

# Check each agent window
for agent in smoketest edgecase chaos; do
  # Check if the window exists and has a running process
  if ! tmux list-windows -t testing -F '#{window_name}' | grep -q "^${agent}$"; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $agent window missing, restarting" >> "$RESTART_LOG"
    bash "$SCRIPT_DIR/start-agent.sh" "$agent"
    continue
  fi

  # Check if the pane has a running claude process
  PANE_PID=$(tmux list-panes -t "testing:$agent" -F '#{pane_pid}' 2>/dev/null | head -1)
  if [ -n "$PANE_PID" ]; then
    # Check if any claude process is a child of the pane
    if ! pgrep -P "$PANE_PID" -f "claude" >/dev/null 2>&1; then
      echo "$(date '+%Y-%m-%d %H:%M:%S') - $agent process dead, restarting" >> "$RESTART_LOG"
      bash "$SCRIPT_DIR/start-agent.sh" "$agent"
    fi
  fi
done
```

- [ ] **Step 5: Write setup.sh**

Write `testing/scripts/setup.sh`:

```bash
#!/bin/bash
# First-time setup for the testing VM
# Usage: bash testing/scripts/setup.sh
#
# This script is interactive — it prompts for configuration values.
# Run it once on a fresh GCP VM.

set -euo pipefail

GREEN='\033[0;32m'; CYAN='\033[0;36m'; RED='\033[0;31m'; NC='\033[0m'
log()  { echo -e "${GREEN}[setup]${NC} $*"; }
step() { echo -e "\n${CYAN}=== $1 ===${NC}"; }
die()  { echo -e "${RED}[setup] ERROR:${NC} $*" >&2; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$PROJECT_DIR/.." && pwd)"

# --- Step 1: Check prerequisites ---
step "Checking prerequisites"

MISSING=()
command -v docker >/dev/null 2>&1 || MISSING+=("docker")
command -v docker compose version >/dev/null 2>&1 || MISSING+=("docker-compose")
command -v go >/dev/null 2>&1 || MISSING+=("go")
command -v node >/dev/null 2>&1 || MISSING+=("node")
command -v tmux >/dev/null 2>&1 || MISSING+=("tmux")
command -v ngrok >/dev/null 2>&1 || MISSING+=("ngrok")
command -v forge >/dev/null 2>&1 || MISSING+=("foundry (forge)")
command -v claude >/dev/null 2>&1 || MISSING+=("claude-code (npm install -g @anthropic-ai/claude-code)")
command -v jq >/dev/null 2>&1 || MISSING+=("jq")

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "Missing prerequisites:"
  for dep in "${MISSING[@]}"; do
    echo "  - $dep"
  done
  die "Install the above dependencies and re-run setup.sh"
fi

log "All prerequisites found"

# --- Step 2: Check .env ---
step "Checking .env configuration"

ENV_FILE="$REPO_ROOT/.env"
if [ ! -f "$ENV_FILE" ]; then
  if [ -f "$REPO_ROOT/.env.example" ]; then
    log "Copying .env.example to .env"
    cp "$REPO_ROOT/.env.example" "$ENV_FILE"
  else
    die ".env.example not found at $REPO_ROOT"
  fi
fi

# Check for required values
log "Please verify these values in $ENV_FILE:"
echo ""
echo "  CHAIN_URL=https://coston2-api.flare.network/ext/C/rpc"
echo "  LOCAL_MODE=false"
echo "  ADDRESSES_FILE=./config/coston2/deployed-addresses.json"
echo "  DEPLOYMENT_PRIVATE_KEY=<your funded Coston2 key>"
echo "  SIMULATED_TEE=true"
echo ""

read -p "Have you configured .env with a funded Coston2 key? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  die "Configure .env first, then re-run setup.sh"
fi

# --- Step 3: Check Claude Code auth ---
step "Checking Claude Code authentication"

log "Claude Code requires authentication. If not yet authenticated, run:"
echo "  claude /login"
echo ""
read -p "Is Claude Code authenticated? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  log "Run 'claude /login' and then re-run setup.sh"
  exit 0
fi

# --- Step 4: Create directory structure ---
step "Creating directory structure"

mkdir -p "$PROJECT_DIR/agents/smoketest/results"
mkdir -p "$PROJECT_DIR/agents/edgecase/results"
mkdir -p "$PROJECT_DIR/agents/chaos/results"
mkdir -p "$PROJECT_DIR/summary"

log "Directory structure ready"

# --- Step 5: Make scripts executable ---
step "Making scripts executable"

chmod +x "$SCRIPT_DIR/start.sh"
chmod +x "$SCRIPT_DIR/start-agent.sh"
chmod +x "$SCRIPT_DIR/stop.sh"
chmod +x "$SCRIPT_DIR/health-check.sh"
chmod +x "$PROJECT_DIR/shared/hooks/audit-log.sh"
chmod +x "$PROJECT_DIR/shared/hooks/teardown.sh"

log "Scripts are executable"

# --- Step 6: Set up cron watchdog ---
step "Setting up cron watchdog"

CRON_LINE="*/5 * * * * $SCRIPT_DIR/health-check.sh >> $PROJECT_DIR/summary/health-check.log 2>&1"
if crontab -l 2>/dev/null | grep -q "health-check.sh"; then
  log "Cron watchdog already installed"
else
  (crontab -l 2>/dev/null; echo "$CRON_LINE") | crontab -
  log "Cron watchdog installed: every 5 minutes"
fi

# --- Done ---
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} Setup complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "  1. Start the agents:  bash $SCRIPT_DIR/start.sh"
echo "  2. Attach to session: tmux attach -t testing"
echo "  3. Monitor results:   ls $PROJECT_DIR/agents/*/results/"
echo "  4. Check findings:    cat $PROJECT_DIR/summary/findings.md"
echo ""
```

- [ ] **Step 6: Make all scripts executable**

```bash
chmod +x testing/scripts/start.sh
chmod +x testing/scripts/start-agent.sh
chmod +x testing/scripts/stop.sh
chmod +x testing/scripts/health-check.sh
chmod +x testing/scripts/setup.sh
```

- [ ] **Step 7: Commit**

```bash
git add -f testing/scripts/
git commit -m "feat: add orchestration scripts (setup, start, stop, health-check)"
```

---

### Task 10: Final Integration — Verify and Test Locally

**Files:**
- Modify: `testing/.gitignore` (if needed)

- [ ] **Step 1: Verify all files exist**

```bash
find testing/ -type f | sort
```

Expected output should include all files from Tasks 1-9:
- `testing/scripts/{setup,start,start-agent,stop,health-check}.sh`
- `testing/agents/{smoketest,edgecase,chaos}/CLAUDE.md`
- `testing/agents/{smoketest,edgecase,chaos}/.claude/settings.json`
- `testing/agents/{smoketest,edgecase,chaos}/.claude/skills/heartbeat/SKILL.md`
- `testing/agents/{smoketest,edgecase,chaos}/.claude/skills/run-scenario/SKILL.md`
- `testing/agents/{smoketest,edgecase,chaos}/.claude/commands/start-heartbeat.md`
- `testing/shared/hooks/{audit-log,teardown}.sh`
- `testing/shared/scenarios/{smoketest,edgecase,chaos}-scenarios.md`
- `testing/agents/edgecase/scenario-tracker.md`
- `testing/summary/{findings,latest-status}.md`

- [ ] **Step 2: Verify hooks are executable**

```bash
ls -la testing/shared/hooks/
```

Expected: `-rwxr-xr-x` permissions on both `.sh` files.

- [ ] **Step 3: Verify scripts are executable**

```bash
ls -la testing/scripts/
```

Expected: `-rwxr-xr-x` permissions on all `.sh` files.

- [ ] **Step 4: Verify settings.json is valid JSON for each agent**

```bash
for agent in smoketest edgecase chaos; do
  echo -n "$agent: "
  jq empty "testing/agents/$agent/.claude/settings.json" && echo "OK" || echo "INVALID"
done
```

Expected: all three print "OK".

- [ ] **Step 5: Dry-run the lock mechanism**

```bash
# Simulate lock acquire
echo "smoketest|$(date +%s)" > /tmp/flare-extension-testing.lock
cat /tmp/flare-extension-testing.lock

# Simulate lock check
LOCK_CONTENT=$(cat /tmp/flare-extension-testing.lock)
LOCK_OWNER=$(echo "$LOCK_CONTENT" | cut -d'|' -f1)
LOCK_TIME=$(echo "$LOCK_CONTENT" | cut -d'|' -f2)
NOW=$(date +%s)
AGE=$(( NOW - LOCK_TIME ))
echo "Owner: $LOCK_OWNER, Age: ${AGE}s"

# Cleanup
rm -f /tmp/flare-extension-testing.lock
```

Expected: lock file created, parsed correctly, cleaned up.

- [ ] **Step 6: Final commit if any changes needed**

```bash
git status
# If any changes:
git add -f testing/
git commit -m "chore: final testing agent integration cleanup"
```
