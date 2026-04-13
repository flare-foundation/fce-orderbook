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
