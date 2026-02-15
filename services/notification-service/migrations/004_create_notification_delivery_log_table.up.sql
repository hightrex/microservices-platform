-- Track every delivery attempt with provider responses
CREATE TYPE delivery_status AS ENUM ('delivered', 'failed', 'bounced');

CREATE TABLE notification_delivery_log (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id   UUID NOT NULL REFERENCES notifications(id),
    tenant_id         UUID NOT NULL,
    channel           notification_channel NOT NULL,
    recipient         VARCHAR(500) NOT NULL,
    status            delivery_status NOT NULL,
    error_message     TEXT,
    provider_response JSONB,
    retry_count       INT NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_log_notification
    ON notification_delivery_log (notification_id);

CREATE INDEX idx_delivery_log_tenant_status_created
    ON notification_delivery_log (tenant_id, status, created_at);
