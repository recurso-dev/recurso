-- Customers: Add GST fields
ALTER TABLE customers ADD COLUMN IF NOT EXISTS gstin VARCHAR(50);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS tax_type VARCHAR(20) DEFAULT 'consumer'; -- 'business', 'consumer'
ALTER TABLE customers ADD COLUMN IF NOT EXISTS place_of_supply VARCHAR(50); -- State Code e.g. 'TN', 'KA'

-- Invoices: Add Tax Breakdown and Compliance fields
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS igst_amount BIGINT DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS cgst_amount BIGINT DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS sgst_amount BIGINT DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS hsn_code VARCHAR(20);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS irn VARCHAR(255); -- Invoice Reference Number (E-invoice)
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS ack_no VARCHAR(255); -- Ack Number (E-invoice)

-- Subscriptions: Add Razorpay Subscription ID
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS razorpay_subscription_id VARCHAR(100);
CREATE INDEX idx_subscriptions_razorpay_id ON subscriptions(razorpay_subscription_id);
