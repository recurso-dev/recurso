-- Refund support for credit notes (v1).
-- type: 'adjustment' (spendable credit, previous behavior) or 'refund'
-- refund_status: 'none' | 'pending' | 'processed' | 'refund_failed' | 'manual_required'
-- refund_id: gateway-side refund id (Stripe re_*, Razorpay rfnd_*)
ALTER TABLE credit_notes ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'adjustment';
ALTER TABLE credit_notes ADD COLUMN IF NOT EXISTS refund_status VARCHAR(30) NOT NULL DEFAULT 'none';
ALTER TABLE credit_notes ADD COLUMN IF NOT EXISTS refund_id VARCHAR(255);
ALTER TABLE credit_notes ADD COLUMN IF NOT EXISTS refund_message TEXT NOT NULL DEFAULT '';

-- Gateway-side payment identifier that settled the invoice (Stripe pi_*/ch_*,
-- Razorpay pay_*). Populated by the payment-success webhook handlers; refunds
-- are issued against this id. NULL/empty for offline payments and invoices
-- paid before this column existed (those refunds are flagged manual_required).
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS gateway_payment_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_credit_notes_invoice_id ON credit_notes(invoice_id);
