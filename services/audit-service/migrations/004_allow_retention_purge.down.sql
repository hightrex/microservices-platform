-- 004_allow_retention_purge.down.sql
-- Restore the original immutability trigger (no retention bypass).

CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable — UPDATE and DELETE are not allowed';
END;
$$ LANGUAGE plpgsql;
