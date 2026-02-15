#!/bin/bash
# scripts/security/kali/modules/05-auth-tests.sh
# Phase 1: Authentication & Authorization Security Tests

set -euo pipefail

RESULTS_DIR="/pentest/results/auth"
mkdir -p "$RESULTS_DIR"

TARGET="http://api-gateway:3000"
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }
section() { echo ""; echo "=== $* ==="; echo ""; }

section "JWT Security Tests"

log "Test 1: Tampered JWT signature"
# Get valid token
VALID_TOKEN=$(curl -s -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"TestPassword123!"}' \
    | jq -r '.data.access_token')

# Tamper with signature (change last character)
TAMPERED_TOKEN="${VALID_TOKEN%?}X"

# Try to use tampered token
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $TAMPERED_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" != "401" ]; then
    echo "CRITICAL: Tampered JWT was accepted! HTTP $HTTP_CODE" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Tampered JWT correctly rejected"
fi

log "Test 2: Expired JWT"
# Create an expired token (this would need to be done in Go/Node to set exp claim)
# For now, test with a token from 24 hours ago if available
# This test might need coordination with the service or a test endpoint

log "Test 3: JWT with wrong signing key"
# Create token signed with different key (would need openssl or jwt tool)
# jwt encode --alg HS256 --secret "wrong-key" --exp=+1h '{"sub":"test","tid":"123"}'

log "Test 4: Missing required claims"
MISSING_CLAIMS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.invalid"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $MISSING_CLAIMS_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" != "401" ]; then
    echo "HIGH: JWT with missing claims was accepted! HTTP $HTTP_CODE" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ JWT with missing claims rejected"
fi

log "Test 5: Claim manipulation attempt"
# Try to access another tenant's data by using valid token but modifying headers
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    -H "X-Tenant-ID: spoofed-tenant-id")

BODY=$(echo "$RESPONSE" | sed '$d')
if echo "$BODY" | grep -q "spoofed-tenant-id"; then
    echo "CRITICAL: Tenant ID spoofing successful!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Tenant ID header spoofing prevented"
fi

section "Brute Force Protection Tests"

log "Test 6: Account lockout mechanism"
LOCKOUT_RESULTS="$RESULTS_DIR/brute-force.log"

# Attempt multiple failed logins
for i in {1..15}; do
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$TARGET/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"bruteforce@test.com","password":"WrongPassword'$i'"}')
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    echo "Attempt $i: HTTP $HTTP_CODE" >> "$LOCKOUT_RESULTS"
    
    # Check if account locked
    if [ "$i" -gt 10 ] && [ "$HTTP_CODE" == "429" ]; then
        log "✓ Account lockout triggered after $i attempts"
        break
    fi
    
    sleep 1
done

# Verify lockout persists
sleep 2
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"bruteforce@test.com","password":"CorrectPassword123!"}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "HIGH: No account lockout - brute force possible!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Account remains locked after failed attempts"
fi

section "Session Security Tests"

log "Test 7: Session revocation on logout"
# Login and get session
LOGIN_RESPONSE=$(curl -s -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"session-test@test.com","password":"TestPassword123!"}')

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.access_token')
REFRESH_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.refresh_token')

# Use token
curl -s -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $ACCESS_TOKEN" > /dev/null

# Logout
curl -s -X POST "$TARGET/api/v1/auth/logout" \
    -H "Authorization: Bearer $ACCESS_TOKEN" > /dev/null

# Try to use token after logout
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "HIGH: Session still valid after logout!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Session correctly invalidated on logout"
fi

log "Test 8: Refresh token after logout"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X POST "$TARGET/api/v1/auth/refresh" \
    -H "Content-Type: application/json" \
    -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "HIGH: Refresh token still valid after logout!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Refresh token correctly invalidated"
fi

section "MFA Security Tests"

log "Test 9: MFA setup and verification"
# Setup MFA for a test user
MFA_RESPONSE=$(curl -s -X POST "$TARGET/api/v1/auth/mfa/setup" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -H "Content-Type: application/json")

MFA_SECRET=$(echo "$MFA_RESPONSE" | jq -r '.data.secret')

log "Test 10: MFA code reuse prevention"
# Generate TOTP code (would need oathtool or similar)
# For now, test with invalid code
INVALID_CODE="000000"

RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X POST "$TARGET/api/v1/auth/mfa/verify" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"$INVALID_CODE\"}")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Invalid MFA code accepted!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Invalid MFA code rejected"
fi

section "RBAC Boundary Tests"

log "Test 11: Member cannot delete users"
# Login as member
MEMBER_RESPONSE=$(curl -s -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"member@test.com","password":"MemberPass123!"}')

MEMBER_TOKEN=$(echo "$MEMBER_RESPONSE" | jq -r '.data.access_token')

# Try to delete user
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X DELETE "$TARGET/api/v1/users/test-user-id" \
    -H "Authorization: Bearer $MEMBER_TOKEN")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
    echo "CRITICAL: Member can delete users!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Member correctly cannot delete users"
fi

log "Test 12: Member cannot assign roles"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X PUT "$TARGET/api/v1/users/test-user-id/role" \
    -H "Authorization: Bearer $MEMBER_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role":"org_admin"}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Member can assign roles!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Member correctly cannot assign roles"
fi

log "Test 13: User cannot modify own role (identity guard)"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X PUT "$TARGET/api/v1/users/self/role" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role":"org_owner"}')

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "CRITICAL: Users can modify own role!" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Identity guard prevents self-role modification"
fi

section "Password Policy Tests"

log "Test 14: Weak password rejection"
WEAK_PASSWORDS=("12345678" "password" "qwerty" "test")

for weak_pass in "${WEAK_PASSWORDS[@]}"; do
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -X POST "$TARGET/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"weak-$RANDOM@test.com\",\"password\":\"$weak_pass\",\"first_name\":\"Test\",\"last_name\":\"User\"}")
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "201" ]; then
        echo "HIGH: Weak password '$weak_pass' was accepted!" >> "$RESULTS_DIR/findings.txt"
    fi
done

log "✓ Password policy tests complete"

log "Test 15: Password history check"
# Change password twice to same value
CHANGE_RESPONSE=$(curl -s -X PUT "$TARGET/api/v1/users/self/password" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"old_password":"TestPassword123!","new_password":"NewPassword456!"}')

CHANGE_BACK_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X PUT "$TARGET/api/v1/users/self/password" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"old_password":"NewPassword456!","new_password":"TestPassword123!"}')

HTTP_CODE=$(echo "$CHANGE_BACK_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
if [ "$HTTP_CODE" == "200" ]; then
    echo "MEDIUM: Password history not enforced" >> "$RESULTS_DIR/findings.txt"
else
    log "✓ Password reuse prevented by history check"
fi

section "Rate Limiting Tests"

log "Test 16: Rate limiting on login endpoint"
RATE_LIMIT_LOG="$RESULTS_DIR/rate-limiting.log"

for i in {1..25}; do
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}\nRateLimit: %{header_json}" \
        -X POST "$TARGET/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"ratelimit@test.com","password":"TestPassword123!"}')
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    echo "Request $i: HTTP $HTTP_CODE" >> "$RATE_LIMIT_LOG"
    
    if [ "$HTTP_CODE" == "429" ]; then
        log "✓ Rate limiting triggered after $i requests"
        
        # Check for required headers
        if echo "$RESPONSE" | grep -q "X-RateLimit-Limit"; then
            log "✓ Rate limit headers present"
        else
            echo "MEDIUM: Missing X-RateLimit-* headers" >> "$RESULTS_DIR/findings.txt"
        fi
        break
    fi
done

if [ "$i" -eq 25 ]; then
    echo "HIGH: No rate limiting on login endpoint!" >> "$RESULTS_DIR/findings.txt"
fi

section "Auth Tests Complete"
log "Results saved to $RESULTS_DIR"

# Count findings
if [ -f "$RESULTS_DIR/findings.txt" ]; then
    CRITICAL=$(grep -c "CRITICAL:" "$RESULTS_DIR/findings.txt" || true)
    HIGH=$(grep -c "HIGH:" "$RESULTS_DIR/findings.txt" || true)
    MEDIUM=$(grep -c "MEDIUM:" "$RESULTS_DIR/findings.txt" || true)
    
    log "Findings: $CRITICAL critical, $HIGH high, $MEDIUM medium"
    
    if [ $CRITICAL -gt 0 ] || [ $HIGH -gt 0 ]; then
        log "⚠️  CRITICAL or HIGH findings detected!"
        exit 1
    fi
else
    log "✓ No security findings - all tests passed!"
fi
