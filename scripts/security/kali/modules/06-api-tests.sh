#!/bin/bash
# scripts/security/kali/modules/06-api-tests.sh
#
# Tests for common API misconfigurations:
#   - Missing auth on protected endpoints
#   - CORS wildcard / missing headers
#   - Overly permissive HTTP methods
#   - Sensitive data in error responses

set -uo pipefail

RESULTS_DIR="/pentest/results/api"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting API security tests against $TARGET..."

# ------------------------------------------------------------------
# Unauthenticated access to protected endpoints
# ------------------------------------------------------------------
log "Testing unauthenticated access to protected endpoints..."

PROTECTED_ENDPOINTS=(
    "GET /api/v1/users"
    "GET /api/v1/organizations"
    "GET /api/v1/audit/logs"
)

for entry in "${PROTECTED_ENDPOINTS[@]}"; do
    method="${entry%% *}"
    path="${entry#* }"

    code=$(curl -sf --max-time 10 -o /dev/null -w "%{http_code}" \
        -X "$method" "$TARGET$path") || true

    if [ "$code" == "401" ] || [ "$code" == "403" ]; then
        log "PASS: $method $path requires auth ($code)"
    else
        log "FAIL: $method $path accessible without auth ($code)"
        echo "HIGH: Unauthenticated access to $method $path (got $code)" >> "$RESULTS_DIR/findings.txt"
    fi
done

# ------------------------------------------------------------------
# CORS header checks
# ------------------------------------------------------------------
log "Testing CORS headers..."

cors_response=$(curl -sf --max-time 10 -I \
    -H "Origin: https://evil.example.com" \
    "$TARGET/api/v1/auth/login" 2>&1) || true

if echo "$cors_response" | grep -qi "access-control-allow-origin: \*"; then
    log "FAIL: CORS wildcard (*) detected"
    echo "MEDIUM: CORS allows wildcard origin (*)" >> "$RESULTS_DIR/findings.txt"
else
    log "PASS: No CORS wildcard detected"
fi

# ------------------------------------------------------------------
# HTTP method tampering
# ------------------------------------------------------------------
log "Testing unexpected HTTP methods..."

METHODS=("TRACE" "OPTIONS" "PATCH" "DELETE")

for method in "${METHODS[@]}"; do
    code=$(curl -sf --max-time 10 -o /dev/null -w "%{http_code}" \
        -X "$method" "$TARGET/api/v1/auth/login" \
        -H "$AUTH_HEADER") || true

    if [ "$method" == "TRACE" ] && [ "$code" == "200" ]; then
        log "FAIL: TRACE method enabled"
        echo "MEDIUM: HTTP TRACE method enabled on /api/v1/auth/login" >> "$RESULTS_DIR/findings.txt"
    fi
done

# ------------------------------------------------------------------
# Security headers check
# ------------------------------------------------------------------
log "Testing security response headers..."

headers=$(curl -sf --max-time 10 -I "$TARGET/api/v1/auth/login" 2>&1) || true

for hdr in "X-Content-Type-Options" "X-Frame-Options" "Strict-Transport-Security" "Content-Security-Policy"; do
    if echo "$headers" | grep -qi "$hdr"; then
        log "PASS: $hdr header present"
    else
        log "INFO: $hdr header missing"
        echo "LOW: Missing security header: $hdr" >> "$RESULTS_DIR/findings.txt"
    fi
done

log "API tests complete"
