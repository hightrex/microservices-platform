-- 004_allow_retention_purge.up.sql
-- Allow the retention worker to delete expired audit logs by checking a session variable.
-- Normal deletes remain blocked; only sessions that SET LOCAL app.retention_purge = 'true'
-- inside a transaction are permitted to delete.

CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    -- Allow retention-based purging when the session variable is set within a transaction
    IF TG_OP = 'DELETE' AND current_setting('app.retention_purge', true) = 'true' THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'Audit logs are immutable — UPDATE and DELETE are not allowed';
END;
$$ LANGUAGE plpgsql;
