#!/usr/bin/env bash
# full-setup.sh — Run the complete extension lifecycle: pre-build → post-build → test.
#
# Usage:
#   ./scripts/full-setup.sh          # pre-build + post-build only
#   ./scripts/full-setup.sh --test   # pre-build + post-build + test
#
# This does NOT start Docker services. The extension TEE + proxy must be running
# before post-build.sh can succeed. Typical workflow:
#
#   1. Start infrastructure (e.g., from e2e/extension: make docker-up && make docker-wait)
#   2. Run: ./scripts/full-setup.sh --test
#      - pre-build.sh deploys contract + registers extension
#      - You start the extension TEE + proxy (pointing at your EXTENSION_ID)
#      - post-build.sh waits for services, then registers TEE version + machine
#      - test.sh sends instructions and verifies results
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'; NC='\033[0m'
log()  { echo -e "${GREEN}[full-setup]${NC} $*"; }
die()  { echo -e "${RED}[full-setup] ERROR:${NC} $*" >&2; exit 1; }

RUN_TESTS=false
for arg in "$@"; do
    case "$arg" in
        --test) RUN_TESTS=true ;;
        *) die "Unknown argument: $arg" ;;
    esac
done

# --- Phase 1: Pre-build ---
echo -e "\n${CYAN}╔══════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  Phase 1: Pre-build                  ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"
"$SCRIPT_DIR/pre-build.sh" || die "Pre-build failed"

# --- Pause: ensure services are running ---
echo ""
echo -e "${YELLOW}────────────────────────────────────────${NC}"
echo -e "${YELLOW} Ensure the extension TEE + proxy are running before continuing.${NC}"
echo -e "${YELLOW} The EXTENSION_ID from pre-build must be set on the TEE.${NC}"
echo -e "${YELLOW}────────────────────────────────────────${NC}"
echo ""
echo -e "Press ${CYAN}Enter${NC} to continue once services are ready..."
read -r

# --- Phase 2: Post-build ---
echo -e "\n${CYAN}╔══════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  Phase 2: Post-build                 ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"
"$SCRIPT_DIR/post-build.sh" || die "Post-build failed"

# --- Phase 3: Test (optional) ---
if [[ "$RUN_TESTS" == "true" ]]; then
    echo -e "\n${CYAN}╔══════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║  Phase 3: Test                       ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"
    "$SCRIPT_DIR/test.sh" || die "Tests failed"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} Full setup complete${NC}"
if [[ "$RUN_TESTS" == "true" ]]; then
    echo -e "${GREEN} (including tests)${NC}"
fi
echo -e "${GREEN}========================================${NC}"
