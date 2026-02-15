#!/bin/bash
# scripts/security/kali/utils/auth-setup.sh

# Function to get authentication token
get_auth_token() {
    local target_url="${1:-http://api-gateway:3000}"
    local email="${2:-admin@test.com}"
    local password="${3:-AdminPass123!}"
    local tenant_id="550e8400-e29b-41d4-a716-446655440000"
    
    log "Attempting to issue token for user: $email"

    # Try to login first
    local login_response=$(curl -s --max-time 10 --connect-timeout 5 -X POST "$target_url/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: $tenant_id" \
        -d "{\"email\":\"$email\",\"password\":\"$password\"}")
    
    log "Login response code: $?"
    # log "Login response body: $login_response" # Uncomment for debugging if needed

    local token=$(echo "$login_response" | jq -r '.data.access_token // empty')

    # If login fails (token is null or empty), try to register
    if [ -z "$token" ] || [ "$token" == "null" ]; then
        log "Login failed or user does not exist. Attempting to register new test user..."
        
        # Create a unique email to avoid conflicts if previous cleanup failed
        local timestamp=$(date +%s)
        local new_email="kali-test-${timestamp}@test.com"
        local new_password="SecureP@ss${timestamp}!"
        
        log "Registering user: $new_email"
        
        local register_response=$(curl -s --max-time 10 --connect-timeout 5 -X POST "$target_url/api/v1/auth/register" \
            -H "Content-Type: application/json" \
            -H "X-Tenant-ID: $tenant_id" \
            -d "{\"email\":\"$new_email\",\"password\":\"$new_password\",\"first_name\":\"Kali\",\"last_name\":\"Tester\"}")
            
        # Check if registration was successful
        local success=$(echo "$register_response" | jq -r '.success // false')
        
        if [ "$success" == "true" ]; then
            log "Registration successful. Logging in..."
            
            # Login with new credentials
            local new_login_response=$(curl -s --max-time 10 --connect-timeout 5 -X POST "$target_url/api/v1/auth/login" \
                -H "Content-Type: application/json" \
                -H "X-Tenant-ID: $tenant_id" \
                -d "{\"email\":\"$new_email\",\"password\":\"$new_password\"}")
            
            token=$(echo "$new_login_response" | jq -r '.data.access_token // empty')
            local user_id=$(echo "$new_login_response" | jq -r '.data.user.id // empty')
            
            log "Authenticated as new user. User ID: $user_id"
        else
            log "Registration failed. Response: $register_response"
            return 1
        fi
    else
        log "Login successful."
    fi

    if [ -n "$token" ] && [ "$token" != "null" ]; then
        echo "$token"
        return 0
    else
        log "Failed to obtain authentication token."
        return 1
    fi
}
