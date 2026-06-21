CREATE TABLE referrals (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    referrer_id UUID NOT NULL REFERENCES customers(id),
    referred_id UUID NOT NULL REFERENCES customers(id),
    code VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    reward_amount BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    qualified_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_referrals_tenant_id ON referrals(tenant_id);
CREATE INDEX idx_referrals_referrer_id ON referrals(referrer_id);
CREATE INDEX idx_referrals_code ON referrals(code);
