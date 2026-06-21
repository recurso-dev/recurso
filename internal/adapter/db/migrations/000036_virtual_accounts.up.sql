CREATE TABLE IF NOT EXISTS virtual_accounts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    invoice_id UUID REFERENCES invoices(id),
    account_number VARCHAR(50),
    ifsc_code VARCHAR(20),
    bank_name VARCHAR(100),
    beneficiary_name VARCHAR(255),
    razorpay_va_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active',
    amount_expected BIGINT,
    amount_received BIGINT DEFAULT 0,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS offline_payments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    invoice_id UUID REFERENCES invoices(id),
    payment_type VARCHAR(20) NOT NULL,
    amount BIGINT NOT NULL,
    currency CHAR(3) DEFAULT 'INR',
    reference_number VARCHAR(100),
    notes TEXT,
    recorded_by VARCHAR(255),
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_virtual_accounts_tenant ON virtual_accounts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_virtual_accounts_razorpay ON virtual_accounts(razorpay_va_id);
CREATE INDEX IF NOT EXISTS idx_offline_payments_tenant ON offline_payments(tenant_id);
