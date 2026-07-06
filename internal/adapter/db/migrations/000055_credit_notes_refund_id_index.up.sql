-- Refund webhook consumption (Stripe charge.refunded / refund.failed,
-- Razorpay refund.processed / refund.failed) resolves the owning credit note
-- by its gateway refund id (rfnd_* / re_*); index the lookup.
CREATE INDEX IF NOT EXISTS idx_credit_notes_refund_id
    ON credit_notes (refund_id)
    WHERE refund_id IS NOT NULL;
