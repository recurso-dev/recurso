-- Add a 'rejected' terminal status to invoice disputes so resolving one can
-- record an outcome (accepted -> resolved, possibly with a credit note; or
-- rejected), not just a flat "resolved".
ALTER TABLE invoice_disputes DROP CONSTRAINT IF EXISTS invoice_disputes_status_check;
ALTER TABLE invoice_disputes
    ADD CONSTRAINT invoice_disputes_status_check
    CHECK (status IN ('open', 'resolved', 'rejected'));
