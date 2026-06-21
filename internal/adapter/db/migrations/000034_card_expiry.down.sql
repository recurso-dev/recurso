DROP TABLE IF EXISTS card_expiry_notifications;
DROP INDEX IF EXISTS idx_customers_card_expiry;
ALTER TABLE customers DROP COLUMN IF EXISTS card_exp_year;
ALTER TABLE customers DROP COLUMN IF EXISTS card_exp_month;
ALTER TABLE customers DROP COLUMN IF EXISTS card_last4;
ALTER TABLE customers DROP COLUMN IF EXISTS card_brand;
