#!/bin/bash
# scripts/security/kali/modules/07-ssrf-tests.sh

set -uo pipefail

RESULTS_DIR="/pentest/results/ssrf"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting SSRF tests..."

# SSRF test targets
SSRF_PAYLOADS=(
    "http://127.0.0.1"
    "http://localhost"
    "http://169.254.169.254/latest/meta-data/"  # AWS metadata
    "http://metadata.google.internal"            # GCP metadata
    "http://10.0.0.1"                            # Private IP
    "http://172.16.0.1"                          # Private IP
    "http://192.168.1.1"                         # Private IP
    "file:///etc/passwd"                         # File URI
    "gopher://localhost:6379/_INFO"              # Redis via gopher
)

# ------------------------------------------------------------------
# Test notification webhook URLs for SSRF
# ------------------------------------------------------------------
log "Testing SSRF via notification webhooks..."
for payload in "${SSRF_PAYLOADS[@]}"; do
    log "Testing payload: $payload"

    response=$(curl -sf --max-time 10 -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$TARGET/api/v1/notifications/send" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "{
            \"channel\": \"webhook\",
            \"webhook_url\": \"$payload\",
            \"template_id\": \"test-template\",
            \"data\": {}
        }" 2>&1) || true

    echo "$response" >> "$RESULTS_DIR/webhook-ssrf.log"

    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2)
    if [ "$http_code" == "200" ] || [ "$http_code" == "201" ]; then
        log "FAIL: SSRF payload accepted ($payload -> $http_code)"
        echo "HIGH: POTENTIAL SSRF via webhook: $payload returned $http_code" >> "$RESULTS_DIR/findings.txt"
    else
        log "PASS: SSRF payload blocked ($payload -> ${http_code:-no_response})"
    fi
done

# ------------------------------------------------------------------
# Test SSRF via file upload URLs
# ------------------------------------------------------------------
log "Testing SSRF via file upload..."
for payload in "${SSRF_PAYLOADS[@]}"; do
    response=$(curl -sf --max-time 10 -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$TARGET/api/v1/files/upload-from-url" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "{\"url\": \"$payload\"}" 2>&1) || true

    echo "$response" >> "$RESULTS_DIR/file-upload-ssrf.log"

    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2)
    if [ "$http_code" == "200" ] || [ "$http_code" == "201" ]; then
        echo "HIGH: POTENTIAL SSRF via file upload: $payload returned $http_code" >> "$RESULTS_DIR/findings.txt"
    fi
done

# ------------------------------------------------------------------
# DNS rebinding test
# ------------------------------------------------------------------
log "Testing DNS rebinding..."
curl -sf --max-time 10 "$TARGET/api/v1/notifications/send" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d '{"channel": "webhook", "webhook_url": "http://evil.com"}' \
    >> "$RESULTS_DIR/dns-rebinding.log" 2>&1 || true

log "SSRF tests complete"
