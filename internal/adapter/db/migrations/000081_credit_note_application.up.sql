-- ENG-153: apply spendable adjustment credit-note balances to invoices at
-- billing time. The invoice keeps its GROSS total; credit_applied records how
-- much account credit settled it, so amount due = total - amount_paid -
-- credit_applied. credit_note_applications is the audit trail of which credit
-- note settled which invoice for how much (and lets ENG-154 post the ledger
-- entry for the applied credit).

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS credit_applied BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS credit_note_applications (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    credit_note_id UUID NOT NULL REFERENCES credit_notes(id),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cn_applications_invoice ON credit_note_applications(invoice_id);
CREATE INDEX IF NOT EXISTS idx_cn_applications_credit_note ON credit_note_applications(credit_note_id);
