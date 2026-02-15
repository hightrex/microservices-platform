-- Create subscription status enum
CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'canceled', 'paused', 'trialing');

-- Subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    plan_id UUID NOT NULL REFERENCES plans(id),
    stripe_subscription_id VARCHAR(255),
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    cancel_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    trial_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_id
    ON subscriptions (tenant_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_id
    ON subscriptions (stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;

-- Only one active/trialing subscription per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_active_tenant
    ON subscriptions (tenant_id) WHERE status IN ('active', 'trialing', 'past_due');
