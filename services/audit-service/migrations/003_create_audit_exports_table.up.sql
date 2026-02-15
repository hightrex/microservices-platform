-- Track audit export jobs
CREATE TYPE export_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE export_format AS ENUM ('csv', 'json');

CREATE TABLE audit_exports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    requested_by    UUID NOT NULL,
    start_date      TIMESTAMPTZ NOT NULL,
    end_date        TIMESTAMPTZ NOT NULL,
    format          export_format NOT NULL DEFAULT 'json',
    status          export_status NOT NULL DEFAULT 'pending',
    file_id         UUID,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_audit_exports_tenant_created
    ON audit_exports (tenant_id, created_at DESC);
