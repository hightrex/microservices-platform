#!/bin/bash
# scripts/security/kali/modules/08-file-upload-tests.sh
#
# File upload attack tests.
# Skipped if the file upload endpoints are not yet implemented (Phase 2).

set -uo pipefail

RESULTS_DIR="/pentest/results/upload"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting File Upload tests..."

# Check if file upload endpoint exists
code=$(curl -sf --max-time 10 -o /dev/null -w "%{http_code}" \
    -X POST "$TARGET/api/v1/files/upload" \
    -H "$AUTH_HEADER" \
    -F "file=@/dev/null") || true

if [ "$code" == "404" ]; then
    log "File upload endpoint not implemented yet (404). Skipping tests."
    exit 0
fi

# If endpoint exists, run basic upload tests
log "File upload endpoint exists (HTTP $code). Running tests..."

# Test: upload a file with a dangerous extension
echo '<?php echo "pwned"; ?>' > /tmp/shell.php
curl -sf --max-time 10 -X POST "$TARGET/api/v1/files/upload" \
    -H "$AUTH_HEADER" \
    -F "file=@/tmp/shell.php" \
    -w "\nStatus: %{http_code}\n" \
    >> "$RESULTS_DIR/dangerous-ext.log" 2>&1 || true
rm -f /tmp/shell.php

# Test: upload oversized file (10MB of zeros)
dd if=/dev/zero of=/tmp/large.bin bs=1M count=10 2>/dev/null
curl -sf --max-time 30 -X POST "$TARGET/api/v1/files/upload" \
    -H "$AUTH_HEADER" \
    -F "file=@/tmp/large.bin" \
    -w "\nStatus: %{http_code}\n" \
    >> "$RESULTS_DIR/oversize.log" 2>&1 || true
rm -f /tmp/large.bin

log "File upload tests complete"
