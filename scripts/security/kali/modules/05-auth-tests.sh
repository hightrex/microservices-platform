#!/bin/bash
# scripts/security/kali/modules/05-auth-tests.sh

set -uo pipefail

RESULTS_DIR="/pentest/results/auth"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting Auth Security Tests..."

# ------------------------------------------------------------------
# JWT Manipulation Tests
# ------------------------------------------------------------------
log "Testing JWT Manipulation..."

if [ -n "$AUTH_TOKEN" ]; then
    # 1. Tampered Signature
    TAMPERED_TOKEN=$(echo "$AUTH_TOKEN" | awk -F. '{OFS="."; print $1, "eyJzdWIiOiJtam9sbmlyIn0", $3}')

    log "Testing Tampered Signature..."
    http_code=$(curl -s --max-time 10 -o /dev/null -w "%{http_code}" -X GET "$TARGET/api/v1/users" \
        -H "Authorization: Bearer $TAMPERED_TOKEN") || true

    # 400 = malformed token (rejected at parse), 401 = invalid signature (rejected at verify)
    # Both are valid rejections. Only 200/2xx would be a real vulnerability.
    if [[ "$http_code" =~ ^(400|401|403)$ ]]; then
        log "PASS: Tampered signature rejected ($http_code)"
    elif [[ "$http_code" =~ ^2 ]]; then
        log "FAIL: Tampered signature ACCEPTED ($http_code) - CRITICAL"
        echo "CRITICAL: Tampered JWT accepted with HTTP $http_code (should be 4xx)" >> "$RESULTS_DIR/findings.txt"
    else
        log "INFO: Tampered signature got unexpected code ($http_code)"
        echo "INFO: Tampered JWT got unexpected HTTP $http_code (expected 400/401)" >> "$RESULTS_DIR/findings.txt"
    fi

    # 2. None Algorithm (Header Manipulation)
    NONE_ALG_HEADER="eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
    PAYLOAD=$(echo "$AUTH_TOKEN" | cut -d. -f2)
    NONE_TOKEN="$NONE_ALG_HEADER.$PAYLOAD."

    log "Testing 'none' algorithm..."
    http_code=$(curl -s --max-time 10 -o /dev/null -w "%{http_code}" -X GET "$TARGET/api/v1/users" \
        -H "Authorization: Bearer $NONE_TOKEN") || true

    if [[ "$http_code" =~ ^(400|401|403)$ ]]; then
        log "PASS: None algorithm rejected ($http_code)"
    elif [[ "$http_code" =~ ^2 ]]; then
        log "FAIL: None algorithm ACCEPTED ($http_code) - CRITICAL"
        echo "CRITICAL: JWT 'none' algorithm accepted with HTTP $http_code (should be 4xx)" >> "$RESULTS_DIR/findings.txt"
    else
        log "INFO: None algorithm got unexpected code ($http_code)"
        echo "INFO: JWT 'none' algorithm got unexpected HTTP $http_code (expected 400/401)" >> "$RESULTS_DIR/findings.txt"
    fi
else
    log "Skipping JWT manipulation tests - no auth token available"
fi

# ------------------------------------------------------------------
# Brute Force Protection
# ------------------------------------------------------------------
log "Testing Brute Force Protection..."

EMAIL="bruteforce-test@test.com"
WRONG_PASS="WrongPass"
MAX_ATTEMPTS=15
LOCKOUT_TRIGGERED=false

for i in $(seq 1 $MAX_ATTEMPTS); do
    code=$(curl -sf --max-time 10 -o /dev/null -w "%{http_code}" -X POST "$TARGET/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$EMAIL\", \"password\":\"$WRONG_PASS$i\"}") || true

    if [ "$code" == "429" ]; then
        log "PASS: Rate limiting/lockout triggered at attempt $i"
        echo "INFO: Brute force protection verified (lockout at attempt $i)" >> "$RESULTS_DIR/findings.txt"
        LOCKOUT_TRIGGERED=true
        break
    fi
    sleep 0.2
done

if [ "$LOCKOUT_TRIGGERED" == "false" ]; then
    log "FAIL: No rate limiting after $MAX_ATTEMPTS attempts"
    echo "HIGH: No brute force protection after $MAX_ATTEMPTS login attempts" >> "$RESULTS_DIR/findings.txt"
fi

# ------------------------------------------------------------------
# Session Security
# ------------------------------------------------------------------
log "Testing Session Security..."

# Register a fresh user specifically for the session test
SESSION_EMAIL="session-test-$(date +%s)@test.com"
SESSION_PASS="SessionP@ss123!"
TENANT_ID="550e8400-e29b-41d4-a716-446655440000"

curl -sf --max-time 10 -X POST "$TARGET/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"email\":\"$SESSION_EMAIL\",\"password\":\"$SESSION_PASS\",\"first_name\":\"Session\",\"last_name\":\"Test\"}" \
    > /dev/null 2>&1 || true

TEMP_TOKEN=$(curl -s --max-time 10 -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"email\":\"$SESSION_EMAIL\",\"password\":\"$SESSION_PASS\"}" \
    | jq -r '.data.access_token // empty') || true

if [ -n "$TEMP_TOKEN" ] && [ "$TEMP_TOKEN" != "null" ]; then
    # Logout
    curl -sf --max-time 10 -X POST "$TARGET/api/v1/auth/logout" \
        -H "Authorization: Bearer $TEMP_TOKEN" > /dev/null 2>&1 || true

    # Try to use token after logout
    code=$(curl -sf --max-time 10 -o /dev/null -w "%{http_code}" -X GET "$TARGET/api/v1/users" \
        -H "Authorization: Bearer $TEMP_TOKEN") || true

    if [ "$code" == "401" ]; then
        log "PASS: Token rejected after logout"
    else
        log "FAIL: Token accepted after logout ($code)"
        echo "HIGH: Token still valid after logout (got $code, expected 401)" >> "$RESULTS_DIR/findings.txt"
    fi
else
    log "Skipping logout test - could not get temp token"
fi

# ------------------------------------------------------------------
# Password Policy
# ------------------------------------------------------------------
log "Testing Password Policy..."

code=$(curl -sf --max-time 10 -o /dev/null -w "%{http_code}" -X POST "$TARGET/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "weakpass@test.com",
        "password": "123",
        "first_name": "Weak",
        "last_name": "Pass"
    }') || true

if [ "$code" == "400" ]; then
    log "PASS: Weak password rejected"
else
    log "FAIL: Weak password accepted ($code)"
    echo "MEDIUM: Weak password policy not enforced (got $code, expected 400)" >> "$RESULTS_DIR/findings.txt"
fi

log "Auth tests complete"
