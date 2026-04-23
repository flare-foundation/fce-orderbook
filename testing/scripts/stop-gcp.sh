#!/usr/bin/env bash
set -euo pipefail

# Stop the GCP VM so it stops incurring compute charges.
# The instance and disk are preserved — restart later with launch-gcp.sh.
# This script is gitignored — it contains project-specific config.
#
# Usage:
#   bash testing/scripts/stop-gcp.sh

PROJECT="flare-network-sandbox"
ZONE="us-central1-a"
INSTANCE="flare-tee-testing"

echo "=== Stopping Flare TEE Testing VM ==="
echo "Project:  $PROJECT"
echo "Instance: $INSTANCE"
echo "Zone:     $ZONE"
echo ""

# Check current status
STATUS=$(gcloud compute instances describe "$INSTANCE" \
    --project="$PROJECT" \
    --zone="$ZONE" \
    --format="value(status)" 2>/dev/null || echo "NOT_FOUND")

if [[ "$STATUS" == "NOT_FOUND" ]]; then
    echo "Instance '$INSTANCE' does not exist. Nothing to stop."
    exit 0
fi

echo "Current status: $STATUS"

if [[ "$STATUS" == "TERMINATED" ]]; then
    echo "Instance is already stopped. Nothing to do."
    exit 0
fi

gcloud compute instances stop "$INSTANCE" \
    --project="$PROJECT" \
    --zone="$ZONE"

echo ""
echo "=== VM stopped ==="
echo "Disk and instance config are preserved. No compute charges while TERMINATED."
echo "Restart with: bash testing/scripts/launch-gcp.sh"
