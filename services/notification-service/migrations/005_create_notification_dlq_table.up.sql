-- Dead letter queue for failed notification processing
CREATE TABLE notification_dlq (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    event_type  VARCHAR(255) NOT NULL,
    payload     JSONB NOT NULL,
    error       TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retried_at  TIMESTAMPTZ
);

CREATE INDEX idx_notification_dlq_tenant_created
    ON notification_dlq (tenant_id, created_at);
