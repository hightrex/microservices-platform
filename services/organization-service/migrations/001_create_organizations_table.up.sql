-- Organizations table
-- Note: the organization id IS the tenant_id for all downstream services
CREATE TYPE org_plan AS ENUM ('free', 'starter', 'business', 'enterprise');
CREATE TYPE org_status AS ENUM ('active', 'suspended', 'deleted');

CREATE TABLE IF NOT EXISTS organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(255) NOT NULL,
    owner_user_id   UUID NOT NULL,
    plan            org_plan NOT NULL DEFAULT 'free',
    status          org_status NOT NULL DEFAULT 'active',
    settings        JSONB NOT NULL DEFAULT '{}',
    max_users       INT NOT NULL DEFAULT 5,
    max_storage_bytes BIGINT NOT NULL DEFAULT 1073741824, -- 1 GB
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_status ON organizations(status);
CREATE INDEX idx_organizations_owner ON organizations(owner_user_id);
