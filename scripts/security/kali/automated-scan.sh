#!/bin/bash
# scripts/security/kali/automated-scan.sh
#
# Runs inside the Kali container. Loads config, obtains auth tokens, then
# executes every scan module in sequence. Individual module failures are
# logged but do NOT abort the entire run (we want maximum coverage).

set -uo pipefail
# NOTE: -e is intentionally omitted at the top level so that a single
# failing module does not prevent subsequent modules from running.

CONFIG_DIR="/pentest/config"
MODULES_DIR="/pentest/modules"
RESULTS_DIR="/pentest/results"
SCAN_MODE="${SCAN_MODE:-normal}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log()     { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*" >&2; }
section() { echo -e "\n${BLUE}========================================${NC}" >&2; echo -e "${BLUE}$*${NC}" >&2; echo -e "${BLUE}========================================${NC}\n" >&2; }

# -------------------------------------------------------------------------
# Load configuration
# -------------------------------------------------------------------------
if [ -f "/pentest/utils/config-loader.sh" ]; then
    source /pentest/utils/config-loader.sh
else
    log "Warning: config-loader.sh not found, using defaults"
fi

section "Starting Automated Penetration Test"
log "Scan mode: $SCAN_MODE"
log "Results:   $RESULTS_DIR"
log "Target:    ${API_GATEWAY_URL:-http://api-gateway:3000}"

# -------------------------------------------------------------------------
# Obtain authentication token
# -------------------------------------------------------------------------
log "Authenticating..."

TOKEN=""
if [ -f "/pentest/utils/auth-setup.sh" ]; then
    source /pentest/utils/auth-setup.sh

    # Do NOT use 2>&1 here — get_auth_token writes log messages to stderr
    # and the token to stdout. Merging them contaminates TOKEN with log lines.
    TOKEN=$(get_auth_token "${API_GATEWAY_URL:-http://api-gateway:3000}") || EXIT_CODE=$?
    EXIT_CODE="${EXIT_CODE:-0}"

    if [ "$EXIT_CODE" -ne 0 ]; then
        log "Error getting auth token (exit code: $EXIT_CODE)"
        TOKEN=""
    fi
else
    log "Warning: auth-setup.sh not found, attempting legacy login..."
    TOKEN=$(curl -sf --max-time 10 --connect-timeout 5 -X POST \
        "${API_GATEWAY_URL:-http://api-gateway:3000}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"admin@test.com","password":"AdminPass123!"}' \
        | jq -r '.data.access_token // empty') || true
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    log "Warning: Authentication failed or timed out. Proceeding with unauthenticated scans only."
    TOKEN=""
else
    log "Authentication successful"
fi

export AUTH_TOKEN="$TOKEN"
export API_GATEWAY_URL="${API_GATEWAY_URL:-http://api-gateway:3000}"

# -------------------------------------------------------------------------
# Run scan modules sequentially
# -------------------------------------------------------------------------
MODULES=(
    "01-reconnaissance.sh"
    "02-web-scanning.sh"
    "03-vulnerability-scan.sh"
    "04-injection-tests.sh"
    "05-auth-tests.sh"
    "06-api-tests.sh"
    "07-ssrf-tests.sh"
    "08-file-upload-tests.sh"
)

PASS=0
FAIL=0

for module in "${MODULES[@]}"; do
    section "Running: $module"
    if [ -f "$MODULES_DIR/$module" ]; then
        if bash "$MODULES_DIR/$module"; then
            log "Module $module completed successfully"
            PASS=$((PASS + 1))
        else
            log "Module $module completed with warnings (exit $?)"
            FAIL=$((FAIL + 1))
        fi
    else
        log "Warning: Module $module not found, skipping"
    fi
done

section "Scan Complete"
log "Modules passed: $PASS  |  Modules with warnings: $FAIL"
log "Results saved to: $RESULTS_DIR"
