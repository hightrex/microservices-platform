CREATE TYPE member_status AS ENUM ('invited', 'active', 'removed');

CREATE TABLE IF NOT EXISTS org_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    role        VARCHAR(50) NOT NULL DEFAULT 'member',
    invited_by  UUID,
    invited_at  TIMESTAMPTZ,
    joined_at   TIMESTAMPTZ,
    status      member_status NOT NULL DEFAULT 'invited',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_org_members_org_user ON org_members(org_id, user_id);
CREATE INDEX idx_org_members_org_status ON org_members(org_id, status);
