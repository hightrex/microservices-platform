-- 006_add_tenant_id_to_password_history.down.sql
DROP INDEX IF EXISTS idx_password_history_tenant_user;
ALTER TABLE password_history DROP COLUMN IF EXISTS tenant_id;
