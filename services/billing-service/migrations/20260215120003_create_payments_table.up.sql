-- Create payment status enum
CREATE TYPE payment_status AS ENUM ('succeeded', 'pending', 'failed');

-- Payments table
-- CRITICAL: NEVER store card numbers, CVV, or full PANs.
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    invoice_id UUID REFERENCES invoices(id),
    stripe_payment_intent_id VARCHAR(255),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    status payment_status NOT NULL DEFAULT 'pending',
    payment_method_type VARCHAR(50),
    receipt_url TEXT,
    failure_reason TEXT,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant_created
    ON payments (tenant_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_stripe_pi_id
    ON payments (stripe_payment_intent_id) WHERE stripe_payment_intent_id IS NOT NULL;
