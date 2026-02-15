DROP TRIGGER IF EXISTS audit_logs_no_delete ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_modification();
DROP TABLE IF EXISTS audit_logs;
DROP TYPE IF EXISTS audit_outcome;
DROP TYPE IF EXISTS actor_type;
DROP TYPE IF EXISTS event_category;
