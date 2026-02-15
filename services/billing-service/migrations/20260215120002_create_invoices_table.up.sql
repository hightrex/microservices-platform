-- Create invoice status enum
CREATE TYPE invoice_status AS ENUM ('draft', 'open', 'paid', 'void', 'uncollectible');

-- Invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    stripe_invoice_id VARCHAR(255),
    invoice_number VARCHAR(50) NOT NULL,
    status invoice_status NOT NULL DEFAULT 'draft',
    amount_due_cents BIGINT NOT NULL CHECK (amount_due_cents >= 0),
    amount_paid_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_paid_cents >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    due_date TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    pdf_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant_created
    ON invoices (tenant_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_stripe_id
    ON invoices (stripe_invoice_id) WHERE stripe_invoice_id IS NOT NULL;
