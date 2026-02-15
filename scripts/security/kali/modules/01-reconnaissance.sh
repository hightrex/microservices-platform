#!/bin/bash
# scripts/security/kali/modules/01-reconnaissance.sh
set -uo pipefail

RESULTS_DIR="/pentest/results/recon"
mkdir -p "$RESULTS_DIR"

TARGET_HOST="${API_GATEWAY_URL:-http://api-gateway:3000}"
# Strip protocol/port to get just the hostname for nmap
TARGET_HOST_BARE=$(echo "$TARGET_HOST" | sed -E 's|https?://||; s|:[0-9]+$||')

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting Nmap scan on $TARGET_HOST_BARE..."

# Scan top 1000 ports (fast) instead of all 65535 (-p-) which takes hours.
# --unprivileged: safe inside unprivileged containers (no raw socket scans).
nmap -sV \
    --top-ports 1000 \
    --open \
    --unprivileged \
    --max-retries 2 \
    --host-timeout 300s \
    -oX "$RESULTS_DIR/nmap-scan.xml" \
    "$TARGET_HOST_BARE" \
    > "$RESULTS_DIR/nmap-scan.txt" 2>&1 || {
        log "Nmap exited with code $? (may still have partial results)"
    }

log "Recon complete"
