DROP INDEX IF EXISTS idx_credit_notes_invoice_id;

ALTER TABLE invoices DROP COLUMN IF EXISTS gateway_payment_id;

ALTER TABLE credit_notes DROP COLUMN IF EXISTS refund_message;
ALTER TABLE credit_notes DROP COLUMN IF EXISTS refund_id;
ALTER TABLE credit_notes DROP COLUMN IF EXISTS refund_status;
ALTER TABLE credit_notes DROP COLUMN IF EXISTS type;
