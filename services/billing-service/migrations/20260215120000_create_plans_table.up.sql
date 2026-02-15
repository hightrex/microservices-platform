-- Create billing interval enum
CREATE TYPE billing_interval AS ENUM ('month', 'year');

-- Plans table (mirrors Stripe products/prices locally)
CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    stripe_price_id VARCHAR(255),
    billing_interval billing_interval NOT NULL DEFAULT 'month',
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    features JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_stripe_price_id
    ON plans (stripe_price_id) WHERE stripe_price_id IS NOT NULL;

-- Seed default plans
INSERT INTO plans (name, billing_interval, price_cents, currency, features, is_active) VALUES
    ('Starter', 'month', 0, 'usd', '{"max_users": 5, "max_storage_gb": 1, "modules": ["auth", "org"]}', true),
    ('Professional', 'month', 2900, 'usd', '{"max_users": 25, "max_storage_gb": 10, "modules": ["auth", "org", "notifications", "files"]}', true),
    ('Enterprise', 'month', 9900, 'usd', '{"max_users": -1, "max_storage_gb": 100, "modules": ["auth", "org", "notifications", "files", "audit", "billing"]}', true),
    ('Professional', 'year', 29000, 'usd', '{"max_users": 25, "max_storage_gb": 10, "modules": ["auth", "org", "notifications", "files"]}', true),
    ('Enterprise', 'year', 99000, 'usd', '{"max_users": -1, "max_storage_gb": 100, "modules": ["auth", "org", "notifications", "files", "audit", "billing"]}', true);
