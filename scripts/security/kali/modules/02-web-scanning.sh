#!/bin/bash
# scripts/security/kali/modules/02-web-scanning.sh
set -euo pipefail
RESULTS_DIR="/pentest/results/web"
mkdir -p "$RESULTS_DIR"
TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting Nikto scan..."
nikto -h "$TARGET" -o "$RESULTS_DIR/nikto-report.json" -Format json -Tuning x 6
log "Nikto scan complete"
