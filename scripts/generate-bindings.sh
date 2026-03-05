#!/usr/bin/env bash
# generate-bindings.sh — Compile Solidity contracts and generate Go bindings.
#
# Prerequisites: forge (Foundry), jq
#
# Usage: ./scripts/generate-bindings.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- CUSTOMIZE: Set your contract name and Go package ---
CONTRACT_NAME="MyExtensionInstructionSender"   # Must match Solidity contract name
GO_PKG="myextension"                            # Go package name for bindings
BINDINGS_DIR="$PROJECT_DIR/tools/pkg/contracts/$GO_PKG"

# --- Scaffold placeholder check ---
if [[ "$CONTRACT_NAME" == "MyExtensionInstructionSender" ]]; then
    echo ""
    echo "ERROR: CONTRACT_NAME is still set to the scaffold placeholder 'MyExtensionInstructionSender'."
    echo ""
    echo "Before running this script, you must:"
    echo "  1. Rename the contract in contracts/InstructionSender.sol"
    echo "  2. Update CONTRACT_NAME and GO_PKG in this script"
    echo "  3. Rename tools/pkg/contracts/myextension/ to match GO_PKG"
    echo "  4. Update the go:generate directive in the new directory"
    echo "  5. Update imports in tools/pkg/utils/instructions.go and tools/cmd/deploy-contract/main.go"
    echo ""
    echo "See README.md for full instructions."
    exit 1
fi

cd "$PROJECT_DIR"

echo "=== Step 1: Compile Solidity contracts ==="
forge build

# Verify the contract name in the source matches what we expect
if ! grep -q "contract ${CONTRACT_NAME}" "$PROJECT_DIR/contracts/InstructionSender.sol" 2>/dev/null; then
    echo ""
    echo "ERROR: Contract name '${CONTRACT_NAME}' not found in contracts/InstructionSender.sol."
    echo "Make sure the contract name in InstructionSender.sol matches CONTRACT_NAME in this script."
    exit 1
fi

echo "=== Step 2: Extract ABI and BIN ==="
FORGE_OUT="$PROJECT_DIR/out/InstructionSender.sol/${CONTRACT_NAME}.json"
if [[ ! -f "$FORGE_OUT" ]]; then
    echo "ERROR: forge output not found at $FORGE_OUT"
    echo "Check that CONTRACT_NAME matches your Solidity contract name."
    exit 1
fi

mkdir -p "$BINDINGS_DIR"

# Extract ABI (JSON array)
jq '.abi' "$FORGE_OUT" > "$BINDINGS_DIR/${CONTRACT_NAME}.abi"

# Extract bytecode (hex string, strip 0x prefix)
jq -r '.bytecode.object' "$FORGE_OUT" | sed 's/^0x//' > "$BINDINGS_DIR/${CONTRACT_NAME}.bin"

echo "  ABI → $BINDINGS_DIR/${CONTRACT_NAME}.abi"
echo "  BIN → $BINDINGS_DIR/${CONTRACT_NAME}.bin"

echo "=== Step 3: Generate Go bindings ==="
cd "$PROJECT_DIR/tools"
go generate ./pkg/contracts/$GO_PKG/

echo "=== Done ==="
echo "Generated: $BINDINGS_DIR/autogen.go"
