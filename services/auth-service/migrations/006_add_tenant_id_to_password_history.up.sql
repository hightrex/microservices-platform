-- 006_add_tenant_id_to_password_history.up.sql
-- Add tenant_id to password_history for multi-tenancy isolation.
ALTER TABLE password_history ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';

-- Backfill tenant_id from the users table so existing records are properly scoped
UPDATE password_history ph
SET tenant_id = u.tenant_id
FROM users u
WHERE ph.user_id = u.id;

-- Remove the default after backfilling
ALTER TABLE password_history ALTER COLUMN tenant_id DROP DEFAULT;

-- Add index for tenant-scoped queries
CREATE INDEX idx_password_history_tenant_user ON password_history (tenant_id, user_id);
