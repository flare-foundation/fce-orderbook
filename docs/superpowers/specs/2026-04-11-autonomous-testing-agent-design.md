# Autonomous Testing Agent — Design Spec

**Date:** 2026-04-11
**Status:** Approved
**Scope:** Continuous, autonomous testing of the Flare TEE extension deployment pipeline against Coston2 testnet

---

## Goal

Build an autonomous testing system that continuously deploys, tests, and tears down extensions on Coston2 testnet to find bugs, edge cases, flaky behavior, and unclear error messages. The system runs indefinitely on a GCP VM with no human intervention required.

## Architecture Overview

Three specialized Claude Code CLI agents run in a shared tmux session on a GCP VM. Each agent has a different testing focus. They coordinate via a lock file to avoid port conflicts, taking turns using the shared Docker/ngrok infrastructure.

```
GCP VM (e2-medium, Ubuntu 22.04)
├── tmux session "testing"
│   ├── Window 0: Smoketest agent (Claude Code CLI)
│   ├── Window 1: Edge Case agent (Claude Code CLI)
│   └── Window 2: Chaos agent (Claude Code CLI)
├── ngrok (shared tunnel, always running, exposes port 6674)
├── Docker (shared, one agent uses at a time)
└── System cron (health-check.sh every 5 min)
```

### Sequential Execution

The deployment stack binds to fixed ports (Redis 6382, ext-proxy 6673/6674, extension-tee 7701/7702, types-server 8100) and uses a single ngrok tunnel. Two agents cannot have services running simultaneously.

**Coordination mechanism:** File-based lock at `/tmp/flare-extension-testing.lock` (absolute path, accessible from any worktree).

- Before starting a test cycle, the agent checks for the lock
- If locked, the agent logs "skipped, locked by [other agent]" and exits the cycle
- If unlocked, the agent creates the lock (format: `agent-name|unix-timestamp`), runs its scenario, tears down services, removes the lock
- A stale lock (>10 min old) gets force-cleared with a warning logged to `summary/findings.md`
- The Stop hook always clears the lock on agent crash/exit

**Throughput:** ~4-5 full test cycles per hour across all three agents.

---

## Agent Definitions

### Agent 1: "Smoketest" — Happy Path Validator

**Purpose:** Continuously verify that the standard deployment flow works end-to-end on Coston2.

**Cycle:** Every 10 minutes via CronCreate.

**What it does each cycle:**
1. Acquire lock
2. Run `full-setup.sh --test` with standard Coston2 config
3. Verify all 4 phases complete successfully
4. Tear down services (`docker compose down`)
5. Log results (timing per phase, pass/fail, warnings/errors)
6. Release lock

**Scenario rotation:**
- Standard `full-setup.sh --test` (most common)
- Step-by-step manual execution (each phase run separately with explicit flags)
- Deploy + test + teardown + immediately deploy again (rapid cycle)
- Run `verify-deploy` before and after each phase
- Vary test payloads (different names, long strings, unicode, empty strings)

**What it catches:** Regressions, flaky infrastructure, Coston2 network issues, timing-dependent failures that only appear with repetition.

### Agent 2: "Edge Case" — Systematic Scenario Explorer

**Purpose:** Methodically work through every documented edge case from `notes/EXTENSION-DEPLOYMENT-EDGE-CASES.md` and `notes/EDGE-CASES-AND-STABILITY-AUDIT.md`.

**Cycle:** Every 15 minutes via CronCreate (offset by 5 min from Smoketest).

**What it does each cycle:**
1. Acquire lock
2. Read `scenario-tracker.md` to find the next untested scenario
3. Set up the specific conditions for that scenario
4. Run the relevant deployment phase(s)
5. Record: did the error match expectations? Was the error message clear? Did recovery work?
6. Update `scenario-tracker.md` with results
7. Tear down and release lock

**Scenario catalog (from edge case docs):**
- D1-D8: Contract deployment edge cases (wrong addresses, zero addresses, duplicate deploy, wrong key, wrong chain URL)
- R1-R8: Registration edge cases (partial failure, re-run after failure, duplicate instruction sender)
- S1-S11: Service startup edge cases (missing config, stale extension ID, port conflicts, wrong proxy URL)
- V1-V6: TEE version registration edge cases (wrong key, already registered, proxy not running)
- T1-T10: TEE machine registration edge cases (partial failure, already registered, unreachable host)
- E1-E9: Test/instruction edge cases (hash mismatches, encoding issues, timeout, fee mismatch)
- Verify-deploy validation: run verify-deploy at different stages, confirm it detects issues correctly

**After completing the full catalog:** Loops back and re-runs to detect flaky behavior.

### Agent 3: "Chaos" — Creative Adversarial Tester

**Purpose:** Try things no one thought of. Concurrent operations, race conditions, unusual timing, creative misconfigurations. Unlike the other agents, Chaos can modify source code, scripts, and configs to explore non-standard paths.

**Cycle:** Every 10 minutes via CronCreate (offset by 7 min from Smoketest).

**Runs in its own git worktree** to avoid interfering with Smoketest and Edge Case, which rely on unmodified code. See "Chaos Worktree" section below.

**What it does each cycle:**
1. Reset worktree to clean state (`git checkout .` in the worktree)
2. Acquire lock
3. Pick or invent a chaos scenario — may involve modifying scripts, Go code, Solidity, or configs
4. Apply modifications in the worktree
5. Execute the scenario from the worktree
6. Log what was modified, what was run, and what happened
7. Tear down and release lock

**Seed scenarios (runtime-only, no code changes):**
- Deploy two extensions simultaneously (two `pre-build.sh` in parallel — on-chain only, no port conflict)
- Kill Docker containers mid-registration (`docker kill extension-tee` during post-build)
- Run phases out of order (test before post-build, post-build before start-services)
- Corrupt `config/extension.env` between phases (wrong EXTENSION_ID, garbage values)
- Send instructions with malformed payloads (empty JSON, oversized payload, wrong encoding)
- Start services with a stale/wrong EXTENSION_ID
- Run everything twice without teardown between runs
- Change `.env` values between phases (different PRIV_KEY for pre-build vs post-build)
- Send 50 instructions in rapid succession
- Register the same instruction sender contract for two different extensions
- Deploy right at a signing policy epoch boundary (if detectable)

**Seed scenarios (code/script modification):**
- Modify `pre-build.sh` to skip preflight checks, see what happens downstream
- Change OPType constants in Go to create hash mismatches, verify error clarity
- Alter Dockerfile to use a different base image, see if code hash change breaks registration
- Modify `docker-compose.yaml` to use wrong ports, verify error messages
- Change hardcoded instruction fee values in Go tools
- Swap JSON encoding for ABI encoding in a handler
- Remove the `timeout` from `WaitMined` calls, verify behavior
- Change `bytes32("GREETING")` case in Solidity, check if the mismatch is caught

**The agent is encouraged to invent new scenarios** based on patterns it observes in previous runs.

### Chaos Worktree

Chaos runs in a separate git worktree so it can freely modify code without affecting the other two agents.

**Setup (by `start.sh`):**
```bash
git worktree add testing/agents/chaos/worktree HEAD
```

**Each cycle:**
1. `cd testing/agents/chaos/worktree && git checkout .` — reset all modifications
2. Apply scenario-specific changes (edit scripts, Go files, Solidity, configs)
3. Run deployment scripts from the worktree
4. Log diffs of what was changed (`git diff` before reset)

**Lock file:** Uses the same absolute path `/tmp/flare-extension-testing.lock` as other agents. Docker ports are host-level regardless of worktree, so the lock still protects against port conflicts.

**Build context:** The worktree contains the full repo, so `docker compose build` and `go run` work normally from within it.

---

## Project Structure

```
testing/
├── scripts/
│   ├── setup.sh              # First-time GCP VM setup
│   ├── start.sh              # Launch ngrok + all 3 agents in tmux
│   ├── start-agent.sh        # Launch a single agent by name
│   ├── stop.sh               # Stop all 3 agents + ngrok
│   └── health-check.sh       # System cron watchdog (every 5 min)
│
├── agents/
│   ├── smoketest/
│   │   ├── CLAUDE.md          # Agent identity, behavior, memory rules
│   │   ├── .claude/
│   │   │   ├── settings.json  # Hooks config
│   │   │   └── skills/
│   │   │       ├── heartbeat/SKILL.md
│   │   │       └── run-scenario/SKILL.md
│   │   └── results/           # One .md file per test run
│   │
│   ├── edgecase/
│   │   ├── CLAUDE.md
│   │   ├── .claude/
│   │   │   ├── settings.json
│   │   │   └── skills/
│   │   │       ├── heartbeat/SKILL.md
│   │   │       └── run-scenario/SKILL.md
│   │   ├── results/
│   │   └── scenario-tracker.md  # Checklist: which scenarios tested, results
│   │
│   └── chaos/
│       ├── CLAUDE.md
│       ├── .claude/
│       │   ├── settings.json
│       │   └── skills/
│       │       ├── heartbeat/SKILL.md
│       │       └── run-scenario/SKILL.md
│       ├── results/
│       └── worktree/          # Git worktree (created by start.sh, gitignored)
│
├── shared/
│   ├── hooks/
│   │   ├── audit-log.sh       # PostToolUse: log tool calls
│   │   ├── timeout-guard.sh   # PreToolUse: warn on long commands without timeout
│   │   └── teardown.sh        # Stop: docker compose down + release lock
│   └── scenarios/
│       ├── smoketest-scenarios.md
│       ├── edgecase-scenarios.md
│       └── chaos-scenarios.md
│
└── summary/
    ├── latest-status.md       # Auto-updated: last run result per agent
    └── findings.md            # Notable bugs, unclear errors, unexpected behavior
```

---

## Result Log Format

Each test run produces a file in the agent's `results/` directory:

**Filename:** `YYYY-MM-DDTHH-MM-SS-scenario-name.md`

```markdown
# [Agent Name] Scenario Name
**Date:** 2026-04-11 14:30:00
**Scenario:** full-setup-standard
**Duration:** 4m 32s
**Result:** PASS | FAIL | PARTIAL | ERROR

## Phases
| Phase | Status | Duration | Notes |
|-------|--------|----------|-------|
| pre-build | PASS | 45s | |
| start-services | PASS | 30s | |
| post-build | PASS | 2m 10s | |
| test | PASS | 1m 07s | |

## Errors
(none, or detailed error output with stderr captured)

## Observations
(anything interesting, even on success — timing anomalies, warnings, unexpected output)
```

---

## Hooks

### audit-log.sh (PostToolUse, matches: `Bash|Edit|Write`)
Logs every tool invocation with timestamp to the agent's `results/audit.log`. Same pattern as claude-code-bot.

### timeout-guard.sh (PreToolUse, matches: `Bash`)
Warns (does not block) if a bash command matches known long-running patterns (`go run`, `docker compose`, `full-setup.sh`, `pre-build.sh`, `post-build.sh`, `test.sh`) without a `timeout` prefix. Real enforcement is in the skill instructions.

### teardown.sh (Stop hook)
On session end or crash:
1. Run `docker compose down` from the repo root
2. Remove `/tmp/flare-extension-testing.lock` if it belongs to this agent
3. Prevents orphaned containers and stale locks

---

## Server Setup

### GCP VM Spec
- **Machine type:** e2-standard-4 (4 vCPU, 16GB RAM)
- **OS:** Ubuntu 22.04 LTS
- **Disk:** 50GB standard persistent
- **Region:** Any (Coston2 RPC is public)

### Prerequisites
1. Docker + Docker Compose
2. Go (version matching repo's go.mod)
3. Node.js (required by Claude Code CLI)
4. tmux
5. ngrok (expose ext-proxy to data providers)
6. Foundry (forge, for Solidity compilation)
7. Claude Code CLI (`npm install -g @anthropic-ai/claude-code`)
8. jq

### setup.sh
1. Install all prerequisites
2. Clone the `tee` repo
3. Copy `.env.example` to `.env`, configure: funded Coston2 private key, `CHAIN_URL=https://coston2-api.flare.network/ext/C/rpc`, `LOCAL_MODE=false`, `ADDRESSES_FILE=./config/coston2/deployed-addresses.json`
4. Configure ngrok auth token
5. Run `claude /login` (interactive, done once manually)
6. Create the `testing/` directory structure
7. Install system cron: `*/5 * * * * /path/to/testing/scripts/health-check.sh`

### start.sh
1. Start ngrok in background (expose port 6674), capture URL to `.env` as `EXT_PROXY_URL`
2. Create/reset Chaos worktree:
   ```bash
   git worktree add testing/agents/chaos/worktree HEAD 2>/dev/null || \
     (cd testing/agents/chaos/worktree && git checkout .)
   ```
3. Launch tmux session `testing` with 3 windows:
   ```bash
   tmux new-session -d -s testing -n smoketest -c testing/agents/smoketest
   tmux new-window -t testing -n edgecase -c testing/agents/edgecase
   tmux new-window -t testing -n chaos -c testing/agents/chaos
   ```
4. In each window, launch Claude Code:
   ```bash
   claude --dangerously-skip-permissions \
     --resume "$(cat .session-id 2>/dev/null || true)" \
     /start-heartbeat
   ```
5. Each agent's `/start-heartbeat` registers its CronCreate schedule

### health-check.sh (system cron, every 5 min)
1. Check if tmux session `testing` exists
2. For each window, check if Claude Code process is alive
3. Restart dead agents via `start-agent.sh <name>`
4. Log restarts to `testing/summary/restarts.log`

### stop.sh
1. Kill all Claude Code processes in tmux
2. Kill tmux session
3. Stop ngrok
4. Run `docker compose down`
5. Clear lock file

---

## CronCreate Schedules

| Agent | Cron Expression | Triggers at |
|-------|----------------|-------------|
| Smoketest | `0,10,20,30,40,50 * * * *` | :00, :10, :20, :30, :40, :50 |
| Edge Case | `3,13,23,33,43,53 * * * *` | :03, :13, :23, :33, :43, :53 |
| Chaos | `7,17,27,37,47,57 * * * *` | :07, :17, :27, :37, :47, :57 |

Each agent triggers every 10 minutes, offset by 3-7 minutes to minimize simultaneous wake-ups. With tests taking ~5-10 min, lock contention will still occur occasionally — agents gracefully skip their cycle when locked.

CronCreate jobs expire after 7 days. `start.sh` re-registers them on every start, and `health-check.sh` restarts agents that die (which triggers re-registration).

---

## What Is NOT In Scope

- **Modifying extension source code** — agents run existing scripts, they don't change the Go/Solidity code
- **Fixing bugs found** — agents log findings, humans fix them
- **Telegram/notifications** — results are log files only, review via SSH
- **Multi-VM** — single VM is sufficient for sequential execution
- **Local devnet testing** — Coston2 only
