-- In-app notification storage and delivery audit trail
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed', 'read');

CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    channel     notification_channel NOT NULL,
    event_type  VARCHAR(255) NOT NULL,
    subject     TEXT,
    body        TEXT NOT NULL,
    status      notification_status NOT NULL DEFAULT 'pending',
    metadata    JSONB DEFAULT '{}',
    sent_at     TIMESTAMPTZ,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_tenant_status
    ON notifications (user_id, tenant_id, status);

CREATE INDEX idx_notifications_tenant_created
    ON notifications (tenant_id, created_at DESC);
