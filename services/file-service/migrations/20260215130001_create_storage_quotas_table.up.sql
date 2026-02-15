CREATE TABLE IF NOT EXISTS storage_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    quota_bytes BIGINT NOT NULL DEFAULT 1073741824, -- 1 GB default
    used_bytes BIGINT NOT NULL DEFAULT 0 CHECK (used_bytes >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_quotas_tenant ON storage_quotas (tenant_id);
