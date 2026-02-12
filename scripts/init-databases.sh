#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
NC='\033[0m'

DB_USER=${POSTGRES_USER:-postgres}
DB_HOST=localhost
DB_PORT=5432

# Wait for Postgres
echo "Waiting for Postgres..."
until podman exec platform-postgres pg_isready -U "$DB_USER"; do
  sleep 1
done

# databases to create
DBS=(
    "auth_db"
    "org_db"
    "notification_db"
    "billing_db"
    "file_db"
    "audit_db"
    "analytics_db"
)

for db in "${DBS[@]}"; do
    if ! podman exec platform-postgres psql -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$db"; then
        echo -e "Creating database: ${GREEN}$db${NC}"
        podman exec platform-postgres createdb -U "$DB_USER" "$db"
    else
        echo "Database $db already exists"
    fi
done
