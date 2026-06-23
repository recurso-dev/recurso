ALTER TABLE customers ADD COLUMN IF NOT EXISTS card_token_id VARCHAR(100);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS card_fingerprint VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_customers_card_fingerprint ON customers(card_fingerprint) WHERE card_fingerprint IS NOT NULL;
