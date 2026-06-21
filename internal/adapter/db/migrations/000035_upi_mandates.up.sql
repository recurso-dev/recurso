CREATE TABLE IF NOT EXISTS mandates (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    subscription_id UUID REFERENCES subscriptions(id),
    mandate_type VARCHAR(20) NOT NULL DEFAULT 'recurring',
    payment_method VARCHAR(20) NOT NULL DEFAULT 'upi',
    vpa VARCHAR(255),
    razorpay_token_id VARCHAR(100),
    razorpay_subscription_id VARCHAR(100),
    max_amount BIGINT NOT NULL,
    frequency VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'created',
    authorized_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    last_debit_at TIMESTAMPTZ,
    next_debit_at TIMESTAMPTZ,
    pre_debit_notified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mandates_next_debit ON mandates(next_debit_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_mandates_tenant ON mandates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_mandates_customer ON mandates(customer_id);

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS mandate_id UUID REFERENCES mandates(id);
