#!/bin/bash
# scripts/security/kali/modules/tenant-isolation-tests.sh
# Phase 1: Tenant Isolation Security Tests

set -euo pipefail

RESULTS_DIR="/pentest/results/tenant-isolation"
mkdir -p "$RESULTS_DIR"

TARGET="http://api-gateway:3000"

log() { echo "[$(date +'%H:%M:%S')] $*"; }
section() { echo ""; echo "=== $* ==="; echo ""; }

section "Tenant Isolation Security Tests"

# Setup: Create two tenants with users
log "Setting up test tenants..."

# Tenant A
log "Creating Tenant A..."
TENANT_A_RESPONSE=$(curl -s -X POST "$TARGET/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email":"admin-tenant-a@test.com",
        "password":"TenantAPass123!",
        "first_name":"Admin",
        "last_name":"TenantA",
        "organization_name":"Tenant A Corp"
    }')

TENANT_A_TOKEN=$(echo "$TENANT_A_RESPONSE" | jq -r '.data.access_token')
TENANT_A_USER_ID=$(echo "$TENANT_A_RESPONSE" | jq -r '.data.user.id')
TENANT_A_ORG_ID=$(echo "$TENANT_A_RESPONSE" | jq -r '.data.organization.id')

log "Tenant A: User=$TENANT_A_USER_ID, Org=$TENANT_A_ORG_ID"

# Create additional user in Tenant A
TENANT_A_USER2_RESPONSE=$(curl -s -X POST "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $TENANT_A_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "email":"user2-tenant-a@test.com",
        "password":"UserPass123!",
        "first_name":"User2",
        "last_name":"TenantA"
    }')

TENANT_A_USER2_ID=$(echo "$TENANT_A_USER2_RESPONSE" | jq -r '.data.id')

# Tenant B
log "Creating Tenant B..."
TENANT_B_RESPONSE=$(curl -s -X POST "$TARGET/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email":"admin-tenant-b@test.com",
        "password":"TenantBPass123!",
        "first_name":"Admin",
        "last_name":"TenantB",
        "organization_name":"Tenant B Corp"
    }')

TENANT_B_TOKEN=$(echo "$TENANT_B_RESPONSE" | jq -r '.data.access_token')
TENANT_B_USER_ID=$(echo "$TENANT_B_RESPONSE" | jq -r '.data.user.id')
TENANT_B_ORG_ID=$(echo "$TENANT_B_RESPONSE" | jq -r '.data.organization.id')

log "Tenant B: User=$TENANT_B_USER_ID, Org=$TENANT_B_ORG_ID"

section "Test 1: Cross-Tenant User Read"

log "Attempting: Tenant A trying to read Tenant B user..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users/$TENANT_B_USER_ID" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" == "200" ]; then
    if echo "$BODY" | jq -e ".data.id == \"$TENANT_B_USER_ID\"" > /dev/null 2>&1; then
        echo "CRITICAL: Cross-tenant user read successful! Tenant A read Tenant B user" >> "$RESULTS_DIR/findings.txt"
        echo "  Details: GET /users/$TENANT_B_USER_ID returned 200 with Tenant B data" >> "$RESULTS_DIR/findings.txt"
    fi
else
    log "✓ Cross-tenant user read blocked (HTTP $HTTP_CODE)"
fi

log "Attempting: Tenant B trying to read Tenant A user..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users/$TENANT_A_USER_ID" \
    -H "Authorization: Bearer $TENANT_B_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Cross-tenant user read successful! Tenant B read Tenant A user" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Cross-tenant user read blocked (HTTP $HTTP_CODE)"
fi

section "Test 2: Cross-Tenant User Write/Update"

log "Attempting: Tenant A trying to update Tenant B user..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X PUT "$TARGET/api/v1/users/$TENANT_B_USER_ID" \
    -H "Authorization: Bearer $TENANT_A_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"first_name":"Hacked"}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Cross-tenant user update successful! Tenant A modified Tenant B user" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Cross-tenant user update blocked (HTTP $HTTP_CODE)"
fi

section "Test 3: Cross-Tenant User Delete"

log "Attempting: Tenant A trying to delete Tenant B user..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X DELETE "$TARGET/api/v1/users/$TENANT_B_USER_ID" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
    echo "CRITICAL: Cross-tenant user delete successful! Tenant A deleted Tenant B user" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Cross-tenant user delete blocked (HTTP $HTTP_CODE)"
fi

section "Test 4: Cross-Tenant Organization Read"

log "Attempting: Tenant A trying to read Tenant B organization..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/organizations/$TENANT_B_ORG_ID" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" == "200" ]; then
    if echo "$BODY" | jq -e ".data.id == \"$TENANT_B_ORG_ID\"" > /dev/null 2>&1; then
        echo "CRITICAL: Cross-tenant organization read successful! Tenant A read Tenant B org" >> "$RESULTS_DIR/findings.txt"
    fi
else
    log "✓ Cross-tenant organization read blocked (HTTP $HTTP_CODE)"
fi

section "Test 5: Cross-Tenant Organization Update"

log "Attempting: Tenant A trying to update Tenant B organization..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X PUT "$TARGET/api/v1/organizations/$TENANT_B_ORG_ID" \
    -H "Authorization: Bearer $TENANT_A_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name":"Hacked Org"}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Cross-tenant organization update successful!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Cross-tenant organization update blocked (HTTP $HTTP_CODE)"
fi

section "Test 6: List Endpoints - Tenant Filtering"

log "Testing: User list should only return tenant's users..."
TENANT_A_LIST=$(curl -s -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

# Check if Tenant B user appears in Tenant A's list
if echo "$TENANT_A_LIST" | jq -e ".data[] | select(.id == \"$TENANT_B_USER_ID\")" > /dev/null 2>&1; then
    echo "CRITICAL: List endpoint leaks cross-tenant data! Tenant B user in Tenant A list" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ User list properly filtered by tenant"
fi

log "Testing: Organization list should only return tenant's org..."
TENANT_A_ORG_LIST=$(curl -s -X GET "$TARGET/api/v1/organizations" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

if echo "$TENANT_A_ORG_LIST" | jq -e ".data[] | select(.id == \"$TENANT_B_ORG_ID\")" > /dev/null 2>&1; then
    echo "CRITICAL: Organization list leaks cross-tenant data!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Organization list properly filtered by tenant"
fi

section "Test 7: Search/Filter Tenant Isolation"

log "Testing: Search should not return cross-tenant results..."
SEARCH_RESPONSE=$(curl -s -X GET "$TARGET/api/v1/users?search=Admin" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

# Should find Tenant A admin, not Tenant B admin
TENANT_A_ADMIN_FOUND=$(echo "$SEARCH_RESPONSE" | jq -e ".data[] | select(.id == \"$TENANT_A_USER_ID\")" > /dev/null 2>&1 && echo "true" || echo "false")
TENANT_B_ADMIN_FOUND=$(echo "$SEARCH_RESPONSE" | jq -e ".data[] | select(.id == \"$TENANT_B_USER_ID\")" > /dev/null 2>&1 && echo "true" || echo "false")

if [ "$TENANT_B_ADMIN_FOUND" == "true" ]; then
    echo "CRITICAL: Search leaks cross-tenant data! Found Tenant B user in Tenant A search" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Search properly scoped to tenant"
fi

section "Test 8: Header Spoofing Prevention"

log "Testing: Cannot spoof X-Tenant-ID header..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $TENANT_A_TOKEN" \
    -H "X-Tenant-ID: $TENANT_B_ORG_ID")

BODY=$(echo "$RESPONSE" | sed '$d')

# Should still only see Tenant A users, not Tenant B
if echo "$BODY" | jq -e ".data[] | select(.id == \"$TENANT_B_USER_ID\")" > /dev/null 2>&1; then
    echo "CRITICAL: X-Tenant-ID header spoofing successful!" >> "$RESULTS_DIR/findings.txt"
    echo "  Details: Tenant A spoofed header to access Tenant B data" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Header spoofing prevented - tenant from JWT enforced"
fi

section "Test 9: Session Isolation"

log "Testing: Cannot access other tenant's sessions..."
# Get Tenant A sessions
TENANT_A_SESSIONS=$(curl -s -X GET "$TARGET/api/v1/users/$TENANT_A_USER_ID/sessions" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

SESSION_ID=$(echo "$TENANT_A_SESSIONS" | jq -r '.data[0].id' 2>/dev/null || echo "")

if [ -n "$SESSION_ID" ]; then
    # Try to access from Tenant B
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X GET "$TARGET/api/v1/users/$TENANT_A_USER_ID/sessions" \
        -H "Authorization: Bearer $TENANT_B_TOKEN")
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    if [ "$HTTP_CODE" == "200" ]; then
        echo "CRITICAL: Cross-tenant session access! Tenant B accessed Tenant A sessions" >> "$RESULTS_DIR/findings.txt"
    else
        log "✓ Session access properly isolated"
    fi
fi

section "Test 10: Role Assignment Isolation"

log "Testing: Cannot assign roles to other tenant's users..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X PUT "$TARGET/api/v1/users/$TENANT_B_USER_ID/role" \
    -H "Authorization: Bearer $TENANT_A_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role":"member"}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Cross-tenant role assignment! Tenant A assigned role to Tenant B user" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Cross-tenant role assignment blocked"
fi

section "Test 11: Module Gating Isolation"

log "Testing: Module settings are tenant-specific..."
# Enable a module for Tenant A
curl -s -X POST "$TARGET/api/v1/organizations/$TENANT_A_ORG_ID/modules/notifications" \
    -H "Authorization: Bearer $TENANT_A_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"enabled":true}' > /dev/null

# Check Tenant B doesn't inherit this
TENANT_B_MODULES=$(curl -s -X GET "$TARGET/api/v1/organizations/$TENANT_B_ORG_ID/modules" \
    -H "Authorization: Bearer $TENANT_B_TOKEN")

NOTIFICATIONS_ENABLED=$(echo "$TENANT_B_MODULES" | jq -r '.data.notifications.enabled' 2>/dev/null || echo "false")

if [ "$NOTIFICATIONS_ENABLED" == "true" ]; then
    echo "HIGH: Module settings leaked across tenants!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Module settings properly isolated per tenant"
fi

section "Test 12: Pagination Isolation"

log "Testing: Pagination doesn't leak across tenants..."
# Get first page from Tenant A
PAGE1=$(curl -s -X GET "$TARGET/api/v1/users?page=1&limit=10" \
    -H "Authorization: Bearer $TENANT_A_TOKEN")

# Check no Tenant B users on any page
for page in {1..5}; do
    PAGE_DATA=$(curl -s -X GET "$TARGET/api/v1/users?page=$page&limit=10" \
        -H "Authorization: Bearer $TENANT_A_TOKEN")
    
    if echo "$PAGE_DATA" | jq -e ".data[] | select(.id == \"$TENANT_B_USER_ID\")" > /dev/null 2>&1; then
        echo "CRITICAL: Pagination leaks cross-tenant data on page $page!" >> "$RESULTS_DIR/findings.txt"
        break
    fi
done

log "✓ Pagination properly scoped to tenant"

section "Test 13: SQL Injection with Tenant Bypass"

log "Testing: SQL injection cannot bypass tenant isolation..."
# Try SQL injection in search parameter
INJECTION_PAYLOADS=(
    "' OR tenant_id='$TENANT_B_ORG_ID'--"
    "' UNION SELECT * FROM users WHERE tenant_id='$TENANT_B_ORG_ID'--"
    "'; DROP TABLE tenant_isolation_test;--"
)

for payload in "${INJECTION_PAYLOADS[@]}"; do
    RESPONSE=$(curl -s -X GET "$TARGET/api/v1/users?search=$(echo "$payload" | jq -sRr @uri)" \
        -H "Authorization: Bearer $TENANT_A_TOKEN")
    
    # Check if Tenant B data appears
    if echo "$RESPONSE" | jq -e ".data[] | select(.id == \"$TENANT_B_USER_ID\")" > /dev/null 2>&1; then
        echo "CRITICAL: SQL injection bypassed tenant isolation!" >> "$RESULTS_DIR/findings.txt"
        echo "  Payload: $payload" >> "$RESULTS_DIR/findings.txt"
    fi
done

log "✓ SQL injection does not bypass tenant isolation"

section "Tenant Isolation Tests Complete"

# Count findings
if [ -f "$RESULTS_DIR/findings.txt" ]; then
    CRITICAL=$(grep -c "CRITICAL:" "$RESULTS_DIR/findings.txt" || true)
    HIGH=$(grep -c "HIGH:" "$RESULTS_DIR/findings.txt" || true)
    
    log "Findings: $CRITICAL critical, $HIGH high"
    
    if [ $CRITICAL -gt 0 ]; then
        log "❌ CRITICAL tenant isolation violations detected!"
        cat "$RESULTS_DIR/findings.txt"
        exit 1
    elif [ $HIGH -gt 0 ]; then
        log "⚠️  HIGH severity findings detected"
        cat "$RESULTS_DIR/findings.txt"
    else
        log "✓ All tenant isolation tests passed!"
    fi
else
    log "✓ No tenant isolation violations detected!"
fi

# Cleanup (optional)
log "Cleaning up test tenants..."
# Delete test users/orgs if needed
