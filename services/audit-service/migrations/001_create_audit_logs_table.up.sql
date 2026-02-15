-- Immutable, append-only audit trail with hash chaining
CREATE TYPE event_category AS ENUM ('auth', 'data', 'system', 'security');
CREATE TYPE actor_type AS ENUM ('user', 'system', 'api_key');
CREATE TYPE audit_outcome AS ENUM ('success', 'failure');

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    event_type      VARCHAR(255) NOT NULL,
    event_category  event_category NOT NULL,
    actor_id        VARCHAR(255) NOT NULL,
    actor_type      actor_type NOT NULL DEFAULT 'user',
    resource_type   VARCHAR(255),
    resource_id     VARCHAR(255),
    action          VARCHAR(255) NOT NULL,
    outcome         audit_outcome NOT NULL DEFAULT 'success',
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    metadata        JSONB DEFAULT '{}',
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    previous_hash   VARCHAR(64),
    current_hash    VARCHAR(64) NOT NULL
);

-- Prevent UPDATE and DELETE via a trigger (append-only)
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable — UPDATE and DELETE are not allowed';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER audit_logs_no_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

-- Indexes for efficient querying
CREATE INDEX idx_audit_logs_tenant_timestamp
    ON audit_logs (tenant_id, timestamp DESC);

CREATE INDEX idx_audit_logs_tenant_event_type
    ON audit_logs (tenant_id, event_type);

CREATE INDEX idx_audit_logs_tenant_resource
    ON audit_logs (tenant_id, resource_id);

CREATE INDEX idx_audit_logs_actor
    ON audit_logs (actor_id);

CREATE INDEX idx_audit_logs_current_hash
    ON audit_logs (current_hash);
