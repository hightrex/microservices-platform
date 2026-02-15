-- Usage records table for metered billing
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    metric_name VARCHAR(100) NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    aggregated BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_tenant_metric_time
    ON usage_records (tenant_id, metric_name, recorded_at);

CREATE INDEX IF NOT EXISTS idx_usage_subscription_aggregated
    ON usage_records (subscription_id, aggregated);
