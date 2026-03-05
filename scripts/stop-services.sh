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
echo "Done."
