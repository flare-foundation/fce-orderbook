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
