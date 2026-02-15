CREATE TYPE access_action AS ENUM ('upload', 'download', 'delete', 'view');

CREATE TABLE IF NOT EXISTS file_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    action access_action NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_file_access_log_file ON file_access_log (file_id);
CREATE INDEX IF NOT EXISTS idx_file_access_log_tenant ON file_access_log (tenant_id, created_at);
