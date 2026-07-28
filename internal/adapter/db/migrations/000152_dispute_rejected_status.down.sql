-- Revert: any 'rejected' disputes must be reclassified before the tighter
-- constraint can hold again.
UPDATE invoice_disputes SET status = 'resolved' WHERE status = 'rejected';
ALTER TABLE invoice_disputes DROP CONSTRAINT IF EXISTS invoice_disputes_status_check;
ALTER TABLE invoice_disputes
    ADD CONSTRAINT invoice_disputes_status_check
    CHECK (status IN ('open', 'resolved'));
