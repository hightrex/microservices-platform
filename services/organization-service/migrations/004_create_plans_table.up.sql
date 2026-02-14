CREATE TABLE IF NOT EXISTS plans (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(50) NOT NULL UNIQUE,
    display_name            VARCHAR(100) NOT NULL,
    max_users               INT NOT NULL DEFAULT 5,
    max_storage_bytes       BIGINT NOT NULL DEFAULT 1073741824,
    max_api_calls_per_minute INT NOT NULL DEFAULT 60,
    available_modules       TEXT[] NOT NULL DEFAULT '{}',
    price_monthly_cents     INT NOT NULL DEFAULT 0,
    price_annual_cents      INT NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default plans
INSERT INTO plans (id, name, display_name, max_users, max_storage_bytes, max_api_calls_per_minute, available_modules, price_monthly_cents, price_annual_cents, is_active)
VALUES
    (gen_random_uuid(), 'free', 'Free', 5, 1073741824, 60,
     ARRAY['audit_logging']::TEXT[],
     0, 0, true),
    (gen_random_uuid(), 'starter', 'Starter', 25, 10737418240, 300,
     ARRAY['notifications', 'audit_logging']::TEXT[],
     2900, 29000, true),
    (gen_random_uuid(), 'business', 'Business', 100, 107374182400, 1000,
     ARRAY['notifications', 'billing', 'file_management', 'audit_logging', 'analytics']::TEXT[],
     9900, 99000, true),
    (gen_random_uuid(), 'enterprise', 'Enterprise', 1000, 1099511627776, 5000,
     ARRAY['notifications', 'billing', 'file_management', 'audit_logging', 'analytics']::TEXT[],
     29900, 299000, true);
