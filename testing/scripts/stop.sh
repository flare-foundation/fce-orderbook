#!/bin/bash
# Stop all testing agents, ngrok, and Docker services
# Usage: bash testing/scripts/stop.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
log() { echo -e "${GREEN}[stop]${NC} $*"; }

# Kill tmux sessions
for agent in smoketest edgecase chaos; do
  SESSION="testing-$agent"
  if tmux has-session -t "$SESSION" 2>/dev/null; then
    log "Killing tmux session '$SESSION'..."
    tmux kill-session -t "$SESSION"
  else
    log "No tmux session '$SESSION' found"
  fi
done

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
