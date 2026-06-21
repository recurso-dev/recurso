-- Card expiry notification: store card details and track notifications
ALTER TABLE customers ADD COLUMN IF NOT EXISTS card_brand VARCHAR(20);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS card_last4 VARCHAR(4);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS card_exp_month INTEGER;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS card_exp_year INTEGER;

-- Index for scheduler query: find cards expiring in a given month/year
CREATE INDEX IF NOT EXISTS idx_customers_card_expiry ON customers(card_exp_year, card_exp_month) WHERE card_exp_year IS NOT NULL AND card_exp_month IS NOT NULL;

-- Track which expiry notifications have been sent (dedup)
CREATE TABLE IF NOT EXISTS card_expiry_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    card_exp_month INTEGER NOT NULL,
    card_exp_year INTEGER NOT NULL,
    card_last4 VARCHAR(4) NOT NULL,
    notification_sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, card_exp_month, card_exp_year)
);
