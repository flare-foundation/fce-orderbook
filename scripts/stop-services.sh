#!/usr/bin/env bash
#
# Stop extension services.
#
# By default, stops Docker Compose services (matching the compose files based on LOCAL_MODE).
# Pass --local to stop background Go processes instead.
#
# Usage:
#   ./scripts/stop-services.sh              # docker compose down (default)
#   ./scripts/stop-services.sh --local      # stop background Go processes
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
log()  { echo -e "${GREEN}[stop-services]${NC} $*"; }

# --- Parse flags ---
USE_LOCAL=false
for arg in "$@"; do
    case "$arg" in
        --local) USE_LOCAL=true ;;
        *) echo -e "${RED}[stop-services] ERROR:${NC} Unknown argument: $arg" >&2; exit 1 ;;
    esac
done

# --- Load .env from project root (if present) ---
if [[ -f "$PROJECT_DIR/.env" ]]; then
    set -a
    source "$PROJECT_DIR/.env"
    set +a
fi

LOCAL_MODE="${LOCAL_MODE:-true}"

if [[ "$USE_LOCAL" == "true" ]]; then
    # --- Stop background Go processes ---
    E2E="$SCRIPT_DIR/e2e.sh"
    PID_DIR="$PROJECT_DIR/out/pids"

    log "Stopping background Go processes..."
    "$E2E" stop-all "$PID_DIR"
    log "Done."
else
    # --- Stop Docker Compose services ---
    COMPOSE_FILES=("-f" "$PROJECT_DIR/docker-compose.yaml")

    if [[ "$LOCAL_MODE" != "true" ]]; then
        COMPOSE_FILES+=("-f" "$PROJECT_DIR/docker-compose.coston2.yaml")
    fi

    log "Stopping Docker Compose services..."
    docker compose "${COMPOSE_FILES[@]}" down
    log "Done."
fi
