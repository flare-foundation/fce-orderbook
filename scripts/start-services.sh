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
#   - Redis will be started on :6382 automatically (separate from infrastructure Redis)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
E2E="$SCRIPT_DIR/e2e.sh"
PID_DIR="$PROJECT_DIR/out/pids"
LOG_DIR="$PROJECT_DIR/out/logs"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${GREEN}[start-services]${NC} $*"; }
die()  { echo -e "${RED}[start-services] ERROR:${NC} $*" >&2; exit 1; }

# --- Load .env from project root (if present) ---
if [[ -f "$PROJECT_DIR/.env" ]]; then
    set -a
    source "$PROJECT_DIR/.env"
    set +a
fi

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

# --- Build Go binaries (once) so we run the actual binary, not `go run` ---
BIN_DIR="$PROJECT_DIR/out/bin"
mkdir -p "$BIN_DIR"
log "Building Go binaries..."
cd "$PROJECT_DIR/tools"
go build -o "$BIN_DIR/start-tee" ./cmd/start-tee
go build -o "$BIN_DIR/start-proxy" ./cmd/start-proxy

# --- Start extension TEE ---
log "Starting extension TEE node..."
EXTENSION_ID="$EXTENSION_ID" "$E2E" start ext-tee "$PID_DIR/ext-tee.pid" "$LOG_DIR/ext-tee.log" \
    "$BIN_DIR/start-tee" -extensionID "$EXTENSION_ID"

log "Waiting for extension TEE to initialize..."
sleep 5

# --- Start extension Redis on port 6382 via Docker Compose ---
log "Starting Redis via Docker Compose..."
docker compose -f "$PROJECT_DIR/docker-compose.yaml" up -d redis
log "Waiting for Redis on :6382..."
retries=0
while ! docker compose -f "$PROJECT_DIR/docker-compose.yaml" exec -T redis redis-cli ping > /dev/null 2>&1; do
    retries=$((retries + 1))
    if [ $retries -ge 15 ]; then
        die "Redis container failed to become healthy"
    fi
    sleep 1
done
log "Redis on :6382 ready"

# --- Start extension proxy ---
log "Starting extension proxy..."
PRIVATE_KEY="$PRIVATE_KEY" "$E2E" start ext-proxy "$PID_DIR/ext-proxy.pid" "$LOG_DIR/ext-proxy.log" \
    "$BIN_DIR/start-proxy"

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
echo "  Extension Redis  Docker container (port 6382)"
echo "  Extension TEE    PID $EXT_TEE_PID"
echo "  Extension Proxy  PID $EXT_PROXY_PID"
echo "  Proxy URL        http://localhost:6664"
echo ""
echo -e "${CYAN}Logs${NC}"
echo "  Redis log        docker compose logs redis"
echo "  TEE log          $LOG_DIR/ext-tee.log"
echo "  Proxy log        $LOG_DIR/ext-proxy.log"
echo ""
echo -e "${CYAN}Commands${NC}"
echo "  Status:  $SCRIPT_DIR/e2e.sh status $PID_DIR"
echo "  Stop:    $SCRIPT_DIR/stop-services.sh"
