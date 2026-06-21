ALTER TABLE customers ADD COLUMN referral_code VARCHAR(20);
CREATE INDEX idx_customers_referral_code ON customers(referral_code);
