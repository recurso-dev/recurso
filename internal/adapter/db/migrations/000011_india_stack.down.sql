ALTER TABLE subscriptions DROP COLUMN IF EXISTS razorpay_subscription_id;

ALTER TABLE invoices DROP COLUMN IF EXISTS ack_no;
ALTER TABLE invoices DROP COLUMN IF EXISTS irn;
ALTER TABLE invoices DROP COLUMN IF EXISTS hsn_code;
ALTER TABLE invoices DROP COLUMN IF EXISTS sgst_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS cgst_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS igst_amount;

ALTER TABLE customers DROP COLUMN IF EXISTS place_of_supply;
ALTER TABLE customers DROP COLUMN IF EXISTS tax_type;
ALTER TABLE customers DROP COLUMN IF EXISTS gstin;
