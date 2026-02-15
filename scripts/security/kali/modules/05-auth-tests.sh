#!/bin/bash
# scripts/security/kali/modules/05-auth-tests.sh

set -euo pipefail

RESULTS_DIR="/pentest/results/auth"
mkdir -p "$RESULTS_DIR"

TARGET="${API_GATEWAY_URL:-http://api-gateway:3000}"
# We need a fresh token for some tests
AUTH_HEADER="Authorization: Bearer $AUTH_TOKEN"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

log "Starting Auth Security Tests..."

# ------------------------------------------------------------------
# JWT Manipulation Tests
# ------------------------------------------------------------------
log "Testing JWT Manipulation..."

# 1. Tampered Signature
# Create a token with modified payload but original signature
TAMPERED_TOKEN=$(echo "$AUTH_TOKEN" | awk -F. '{OFS="."; print $1, "eyJzdWIiOiJtam9sbmlyIn0", $3}')

log "Testing Tampered Signature..."
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $TAMPERED_TOKEN")

if [ "$http_code" == "401" ]; then
    log "✓ Tampered signature rejected (401)"
else
    log "❌ Tampered signature accepted ($http_code)"
    echo "CRITICAL: Tampered JWT accepted" >> "$RESULTS_DIR/findings.txt"
fi

# 2. None Algorithm (Header Manipulation)
# Create a token with "alg": "none"
# basic base64 encoding of {"alg":"none","typ":"JWT"} -> eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0
NONE_ALG_HEADER="eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
PAYLOAD=$(echo "$AUTH_TOKEN" | cut -d. -f2)
NONE_TOKEN="$NONE_ALG_HEADER.$PAYLOAD."

log "Testing 'none' algorithm..."
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$TARGET/api/v1/users" \
    -H "Authorization: Bearer $NONE_TOKEN")

if [ "$http_code" == "401" ]; then
    log "✓ None algorithm rejected (401)"
else
    log "❌ None algorithm accepted ($http_code)"
    echo "CRITICAL: JWT 'none' algorithm accepted" >> "$RESULTS_DIR/findings.txt"
fi

# ------------------------------------------------------------------
# Brute Force Protection
# ------------------------------------------------------------------
log "Testing Brute Force Protection..."

# Try to login 15 times with wrong password
EMAIL="admin@test.com"
WRONG_PASS="WrongPass"

count=0
max_attempts=15
failed_logins=0

for i in $(seq 1 $max_attempts); do
    code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$TARGET/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$EMAIL\", \"password\":\"$WRONG_PASS$i\"}")
    
    if [ "$code" == "429" ]; then
        log "✓ Rate limiting/Lockout triggered at attempt $i"
        echo "INFO: Brute force protection verified" >> "$RESULTS_DIR/findings.txt"
        break
    fi
    sleep 0.2
done

# ------------------------------------------------------------------
# Session Security
# ------------------------------------------------------------------
log "Testing Session Security..."

# Test session invalidation on logout
# Get a new token specifically for this test
TEMP_TOKEN=$(curl -s -X POST "$TARGET/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@test.com","password":"AdminPass123!"}' \
    | jq -r '.data.access_token')

if [ -n "$TEMP_TOKEN" ] && [ "$TEMP_TOKEN" != "null" ]; then
    # Logout
    curl -s -X POST "$TARGET/api/v1/auth/logout" -H "Authorization: Bearer $TEMP_TOKEN"
    
    # Try to use token
    code=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$TARGET/api/v1/users" \
        -H "Authorization: Bearer $TEMP_TOKEN")
        
    if [ "$code" == "401" ]; then
        log "✓ Token rejected after logout"
    else
        log "❌ Token accepted after logout ($code)"
        echo "HIGH: Token still valid after logout" >> "$RESULTS_DIR/findings.txt"
    fi
else
    log "Skipping logout test - could not get temp token"
fi

# ------------------------------------------------------------------
# Password Policy
# ------------------------------------------------------------------
log "Testing Password Policy..."

# Try to register with weak password
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$TARGET/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "weakpass@test.com",
        "password": "123",
        "first_name": "Weak",
        "last_name": "Pass"
    }')

if [ "$code" == "400" ]; then
    log "✓ Weak password rejected"
else
    log "❌ Weak password accepted ($code)"
    echo "MEDIUM: Weak password policy not enforced" >> "$RESULTS_DIR/findings.txt"
fi

log "Auth tests complete"
