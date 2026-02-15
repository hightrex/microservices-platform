#!/bin/bash
# scripts/security/kali/modules/02-web-scanning.sh
set -uo pipefail

RESULTS_DIR="/pentest/results/web"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting Nikto scan against $TARGET..."

# -Format json : JSON output
# -Tuning 6    : tuning category 6 = "auth bypass" tests
#   (previous syntax "-Tuning x 6" was invalid; space between flag and number
#    caused nikto to misparse the arguments)
# -maxtime 300 : cap at 5 minutes to avoid hangs
nikto -h "$TARGET" \
    -o "$RESULTS_DIR/nikto-report.json" \
    -Format json \
    -Tuning 6 \
    -maxtime 300s 2>&1 | tee "$RESULTS_DIR/nikto.log" || {
        log "Nikto exited with code $? (may still have partial results)"
    }

log "Nikto scan complete"
