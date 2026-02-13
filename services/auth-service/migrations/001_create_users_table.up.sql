-- 001_create_users_table.up.sql
CREATE TYPE user_status AS ENUM ('active', 'disabled', 'locked');

CREATE TABLE users (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    status user_status NOT NULL DEFAULT 'active',
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_secret VARCHAR(255),
    failed_login_count INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uniq_users_tenant_id_email ON users (tenant_id, email);
CREATE INDEX idx_users_tenant_id_status ON users (tenant_id, status);

-- Enforce lowercase email
ALTER TABLE users ADD CONSTRAINT chk_users_email_lowercase CHECK (email = lower(email));
