#!/usr/bin/env bash
#
# TEMPORARY: Start extension TEE node and proxy as background Go processes.
# This script will be replaced by `docker compose up` once the Dockerfile is added.
#
# Usage:
#   ./scripts/start-services.sh              # uses EXTENSION_ID from config/extension.env
#   EXTENSION_ID=0x... ./scripts/start-services.sh
#
# Prerequisites:
#   - Infrastructure running (Hardhat, indexer, Redis on :6380, normal TEE + proxy)
#   - config/extension.env exists (created by pre-build.sh), OR EXTENSION_ID is set
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
E2E="$SCRIPT_DIR/e2e.sh"
PID_DIR="$PROJECT_DIR/out/pids"
LOG_DIR="$PROJECT_DIR/out/logs"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${GREEN}[start-services]${NC} $*"; }
die()  { echo -e "${RED}[start-services] ERROR:${NC} $*" >&2; exit 1; }

# --- Load extension config ---
CONFIG_FILE="$PROJECT_DIR/config/extension.env"
if [[ -f "$CONFIG_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$CONFIG_FILE"
fi

EXTENSION_ID="${EXTENSION_ID:-}"
PRIVATE_KEY="${PRIVATE_KEY:-0x983760a4ebf75b2ac3a93531168a0f225d01e5dc6e3568adbd46233ba1fb4fa4}"

[[ -n "$EXTENSION_ID" ]] || die "EXTENSION_ID not set. Run pre-build.sh first or set it manually."

log "Extension ID: $EXTENSION_ID"

# --- Start extension TEE ---
log "Starting extension TEE node..."
cd "$PROJECT_DIR/tools"
EXTENSION_ID="$EXTENSION_ID" "$E2E" start ext-tee "$PID_DIR/ext-tee.pid" "$LOG_DIR/ext-tee.log" \
    go run ./cmd/start-tee -extensionID "$EXTENSION_ID"

log "Waiting for extension TEE to initialize..."
sleep 5

# --- Start extension proxy ---
log "Starting extension proxy..."
PRIVATE_KEY="$PRIVATE_KEY" "$E2E" start ext-proxy "$PID_DIR/ext-proxy.pid" "$LOG_DIR/ext-proxy.log" \
    go run ./cmd/start-proxy

cd "$PROJECT_DIR"

# --- Wait for proxy to be ready ---
log "Waiting for extension proxy..."
"$E2E" wait-for-url "http://localhost:6664/info" 60

# --- Summary ---
EXT_TEE_PID=$(cat "$PID_DIR/ext-tee.pid" 2>/dev/null || echo "?")
EXT_PROXY_PID=$(cat "$PID_DIR/ext-proxy.pid" 2>/dev/null || echo "?")

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} Services started${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${CYAN}Processes${NC}"
echo "  Extension TEE    PID $EXT_TEE_PID"
echo "  Extension Proxy  PID $EXT_PROXY_PID"
echo "  Proxy URL        http://localhost:6664"
echo ""
echo -e "${CYAN}Logs${NC}"
echo "  TEE log          $LOG_DIR/ext-tee.log"
echo "  Proxy log        $LOG_DIR/ext-proxy.log"
echo ""
echo -e "${CYAN}Commands${NC}"
echo "  Status:  $SCRIPT_DIR/e2e.sh status $PID_DIR"
echo "  Stop:    $SCRIPT_DIR/stop-services.sh"
