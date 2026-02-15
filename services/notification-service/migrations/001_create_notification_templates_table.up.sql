-- Notification templates for multi-channel delivery
CREATE TYPE notification_channel AS ENUM ('email', 'sms', 'in_app', 'webhook');

CREATE TABLE notification_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(255) NOT NULL,
    channel         notification_channel NOT NULL,
    subject_template TEXT,
    body_template   TEXT NOT NULL,
    template_data_schema JSONB DEFAULT '{}',
    language        VARCHAR(10) NOT NULL DEFAULT 'en',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID
);

CREATE UNIQUE INDEX idx_templates_tenant_name_channel
    ON notification_templates (tenant_id, name, channel);

CREATE INDEX idx_templates_tenant_active
    ON notification_templates (tenant_id, is_active);
