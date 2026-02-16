#!/bin/bash
set -e

# Seed test data for Newman / integration test runs.
# Creates roles for the default test tenant and bootstraps an admin user.

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

TENANT_ID="550e8400-e29b-41d4-a716-446655440000"
ADMIN_EMAIL="admin-test@platform.local"
ADMIN_PASSWORD='AdminP@ss123!'
GATEWAY_URL="${GATEWAY_URL:-http://localhost:3000}"
DB_CONTAINER="platform-postgres"
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="auth_db"

psql_exec() {
    podman exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "$1"
}

echo "==> Waiting for Postgres..."
until podman exec "$DB_CONTAINER" pg_isready -U "$DB_USER" >/dev/null 2>&1; do
    sleep 1
done

echo "==> Waiting for API Gateway..."
until curl -sf "$GATEWAY_URL/health" >/dev/null 2>&1; do
    sleep 2
done

# --- 1. Seed system roles for the test tenant --------------------------------
echo "==> Seeding roles for tenant $TENANT_ID..."
ROLES=("org_owner" "org_admin" "manager" "member" "viewer")
for role in "${ROLES[@]}"; do
    EXISTS=$(psql_exec "SELECT count(*) FROM roles WHERE tenant_id='$TENANT_ID' AND name='$role';")
    if [ "$EXISTS" = "0" ]; then
        psql_exec "INSERT INTO roles (id, tenant_id, name, description, is_system) VALUES (gen_random_uuid(), '$TENANT_ID', '$role', 'System $role role', true);"
        echo -e "  ${GREEN}✓${NC} Created role: $role"
    else
        echo "  • Role $role already exists"
    fi
done

# --- 2. Register admin user via API (so password is properly hashed) ----------
echo "==> Registering admin user ($ADMIN_EMAIL)..."
REG_RESPONSE=$(curl -sf -X POST "$GATEWAY_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\",\"first_name\":\"Admin\",\"last_name\":\"User\"}" 2>&1) || true

if echo "$REG_RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); exit(0 if d.get('success') else 1)" 2>/dev/null; then
    ADMIN_ID=$(echo "$REG_RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
    echo -e "  ${GREEN}✓${NC} Registered admin user: $ADMIN_ID"
else
    # User might already exist — look up by email
    ADMIN_ID=$(psql_exec "SELECT id FROM users WHERE email='$ADMIN_EMAIL' AND tenant_id='$TENANT_ID';")
    if [ -z "$ADMIN_ID" ]; then
        echo -e "  ${RED}✗${NC} Failed to register or find admin user"
        exit 1
    fi
    echo "  • Admin user already exists: $ADMIN_ID"
fi

# --- 3. Assign org_owner + org_admin roles to admin user ----------------------
echo "==> Assigning admin roles to admin user..."
for role_name in org_owner org_admin; do
    ROLE_ID=$(psql_exec "SELECT id FROM roles WHERE tenant_id='$TENANT_ID' AND name='$role_name';")
    ALREADY=$(psql_exec "SELECT count(*) FROM user_roles WHERE user_id='$ADMIN_ID' AND role_id='$ROLE_ID' AND tenant_id='$TENANT_ID';")
    if [ "$ALREADY" = "0" ]; then
        psql_exec "INSERT INTO user_roles (user_id, role_id, tenant_id, granted_at) VALUES ('$ADMIN_ID', '$ROLE_ID', '$TENANT_ID', now());"
        echo -e "  ${GREEN}✓${NC} Assigned $role_name role"
    else
        echo "  • $role_name role already assigned"
    fi
done

# --- 4. Ensure primary org exists in org_db (tenant_id = org_id) --------------
echo "==> Ensuring primary org exists for tenant..."
ORG_EXISTS=$(podman exec "$DB_CONTAINER" psql -U "$DB_USER" -d org_db -tAc \
    "SELECT count(*) FROM organizations WHERE id='$TENANT_ID';")
if [ "$ORG_EXISTS" = "0" ]; then
    podman exec "$DB_CONTAINER" psql -U "$DB_USER" -d org_db -tAc \
        "INSERT INTO organizations (id, owner_user_id, name, slug, plan, status, max_users, max_storage_bytes, created_at, updated_at)
         VALUES ('$TENANT_ID', '$ADMIN_ID', 'Platform Primary Org', 'platform-primary', 'enterprise', 'active', 100, 10737418240, now(), now());"
    echo -e "  ${GREEN}✓${NC} Created primary org"
else
    echo "  • Primary org already exists"
fi

# --- 5. Verify by logging in -------------------------------------------------
echo "==> Verifying admin login..."
LOGIN=$(curl -sf -X POST "$GATEWAY_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

if echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); exit(0 if d.get('data',{}).get('access_token') else 1)" 2>/dev/null; then
    ROLES_IN_TOKEN=$(echo "$LOGIN" | python3 -c "
import sys,json,base64
token = json.load(sys.stdin)['data']['access_token']
payload = token.split('.')[1]
payload += '=' * (4 - len(payload) % 4)
claims = json.loads(base64.urlsafe_b64decode(payload))
print(', '.join(claims.get('roles', [])))
")
    echo -e "  ${GREEN}✓${NC} Admin login successful — roles in JWT: [$ROLES_IN_TOKEN]"
else
    echo -e "  ${RED}✗${NC} Admin login failed"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Test data seeded successfully.${NC}"
echo "   Admin email:    $ADMIN_EMAIL"
echo "   Admin password: $ADMIN_PASSWORD"
echo "   Tenant ID:      $TENANT_ID"
