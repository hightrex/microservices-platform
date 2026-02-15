-- Per-user notification opt-in/opt-out preferences
CREATE TABLE notification_preferences (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    tenant_id   UUID NOT NULL,
    channel     notification_channel NOT NULL,
    event_type  VARCHAR(255) NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_preferences_user_tenant_channel_event
    ON notification_preferences (user_id, tenant_id, channel, event_type);
