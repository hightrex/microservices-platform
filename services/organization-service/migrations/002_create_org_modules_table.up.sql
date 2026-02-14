CREATE TYPE module_name AS ENUM (
    'notifications',
    'billing',
    'file_management',
    'audit_logging',
    'analytics'
);

CREATE TABLE IF NOT EXISTS org_modules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    module_name module_name NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    config      JSONB NOT NULL DEFAULT '{}',
    enabled_at  TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_org_modules_org_module ON org_modules(org_id, module_name);
