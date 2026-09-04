DROP INDEX IF EXISTS idx_unbilled_charges_status;
DROP INDEX IF EXISTS idx_unbilled_charges_subscription_id;
DROP TABLE IF EXISTS unbilled_charges;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_terms;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS payment_terms;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS billing_anchor_day;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS billing_anchor_type;
