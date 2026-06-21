CREATE TABLE gifts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    plan_id UUID NOT NULL REFERENCES plans(id),
    buyer_customer_id UUID NOT NULL REFERENCES customers(id),
    recipient_email VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    redeemed_by_customer_id UUID REFERENCES customers(id),
    redeemed_at TIMESTAMP WITH TIME ZONE,
    duration_months INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_gifts_tenant_id ON gifts(tenant_id);
CREATE INDEX idx_gifts_code ON gifts(code);
CREATE INDEX idx_gifts_buyer_customer_id ON gifts(buyer_customer_id);
