#!/bin/bash
# scripts/security/kali/automated-scan.sh

set -euo pipefail

CONFIG_DIR="/pentest/config"
MODULES_DIR="/pentest/modules"
RESULTS_DIR="/pentest/results"
SCAN_MODE="${SCAN_MODE:-normal}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*" >&2; }
section() { echo -e "\n${BLUE}========================================${NC}"; echo -e "${BLUE}$*${NC}"; echo -e "${BLUE}========================================${NC}\n"; }

# Load configuration
if [ -f "/pentest/utils/config-loader.sh" ]; then
    source /pentest/utils/config-loader.sh
else
    log "Warning: config-loader.sh not found, using defaults"
fi

section "Starting Automated Penetration Test"
log "Scan mode: $SCAN_MODE"
log "Target: API Gateway at http://api-gateway:3000"

# Obtain authentication token
log "Authenticating..."

if [ -f "/pentest/utils/auth-setup.sh" ]; then
    source /pentest/utils/auth-setup.sh
    # Try to get token, defaulting to admin user but falling back to registration
    set +e # Disable exit on error for this call
    TOKEN=$(get_auth_token "http://api-gateway:3000")
    EXIT_CODE=$?
    set -e # Re-enable exit on error
    
    if [ $EXIT_CODE -ne 0 ]; then
        log "Error getting auth token (Exit code: $EXIT_CODE)"
        TOKEN=""
    fi
else
    log "Error: auth-setup.sh not found, attempting legacy login..."
    TOKEN=$(curl -s --max-time 10 --connect-timeout 5 -X POST http://api-gateway:3000/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d '{"email":"admin@test.com","password":"AdminPass123!"}' \
        | jq -r '.data.access_token // empty')
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    log "Warning: Authentication failed or timed out. Proceeding with unauthenticated scans only."
    TOKEN=""
else
    log "Authentication successful"
fi

export AUTH_TOKEN="$TOKEN"

# Run scan modules sequentially
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

for module in "${MODULES[@]}"; do
    section "Running: $module"
    if [ -f "$MODULES_DIR/$module" ]; then
        bash "$MODULES_DIR/$module" || log "Module $module completed with warnings"
    else
        log "Warning: Module $module not found, skipping"
    fi
done

section "Scan Complete"
log "Results saved to: $RESULTS_DIR"
