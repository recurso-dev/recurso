CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    discount_type VARCHAR(20) NOT NULL, -- 'percent' or 'amount'
    discount_value BIGINT NOT NULL,     -- 20 for 20%, 1000 for $10.00
    duration VARCHAR(20) NOT NULL,      -- 'forever', 'once', 'repeating'
    duration_months INT,                -- Nullable, for repeating
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE subscriptions ADD COLUMN coupon_id UUID REFERENCES coupons(id);
