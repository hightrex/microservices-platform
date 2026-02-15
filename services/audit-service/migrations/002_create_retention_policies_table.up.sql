-- Configurable retention policies per event type per tenant
CREATE TABLE retention_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    event_type      VARCHAR(255) NOT NULL,
    retention_days  INT NOT NULL DEFAULT 90,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_retention_policies_tenant_event_type
    ON retention_policies (tenant_id, event_type);
