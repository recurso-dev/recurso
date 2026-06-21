CREATE TABLE credit_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    invoice_id UUID REFERENCES invoices(id), -- Optional: if tied to a specific invoice
    reference VARCHAR(255),
    amount BIGINT NOT NULL, -- Stored in cents
    balance BIGINT NOT NULL, -- Remaining usable balance
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL, -- 'issued', 'used', 'void'
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_credit_notes_tenant_id ON credit_notes(tenant_id);
CREATE INDEX idx_credit_notes_customer_id ON credit_notes(customer_id);
