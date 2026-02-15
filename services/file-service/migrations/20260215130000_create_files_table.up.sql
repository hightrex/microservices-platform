CREATE TYPE file_status AS ENUM ('uploading', 'available', 'deleted', 'quarantined');
CREATE TYPE virus_scan_status AS ENUM ('pending', 'clean', 'infected', 'error');

CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    org_id UUID NOT NULL,
    filename VARCHAR(512) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    s3_key VARCHAR(1024) NOT NULL,
    s3_bucket VARCHAR(255) NOT NULL,
    checksum_sha256 VARCHAR(64),
    status file_status NOT NULL DEFAULT 'uploading',
    virus_scan_status virus_scan_status NOT NULL DEFAULT 'pending',
    thumbnail_s3_key VARCHAR(1024),
    uploaded_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_files_tenant_status ON files (tenant_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_s3_key ON files (s3_key);
CREATE INDEX IF NOT EXISTS idx_files_tenant_user ON files (tenant_id, user_id);
