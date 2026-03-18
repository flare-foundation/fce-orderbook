#!/usr/bin/env bash
#
# TEMPORARY: Stop extension TEE node and proxy background processes.
# This script will be replaced by `docker compose down` once the Dockerfile is added.
#
# Usage:
#   ./scripts/stop-services.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
E2E="$SCRIPT_DIR/e2e.sh"
PID_DIR="$PROJECT_DIR/out/pids"

echo "Stopping extension services..."
"$E2E" stop-all "$PID_DIR"

# Kill any orphaned processes still holding TEE/proxy ports.
for port in 5501 7701 7702 6663 6664; do
    pid=$(lsof -ti ":$port" 2>/dev/null) || true
    if [ -n "$pid" ]; then
        echo "Killing orphaned process on port $port (PID $pid)"
        kill $pid 2>/dev/null || true
    fi
done

echo "Stopping Redis container..."
docker compose -f "$PROJECT_DIR/docker-compose.yaml" down redis 2>/dev/null || true
echo "Done."
