-- 002_create_sessions_table.up.sql
CREATE TABLE sessions (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_sessions_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX uniq_sessions_refresh_token_hash ON sessions (refresh_token_hash);
CREATE INDEX idx_sessions_user_id_tenant_id ON sessions (user_id, tenant_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);
