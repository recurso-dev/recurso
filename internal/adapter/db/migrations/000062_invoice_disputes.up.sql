-- Invoice disputes / queries raised by customers from the billing portal.
--
-- Minimal v1: a customer can raise a single OPEN dispute (a "query") against
-- one of their own invoices. An admin later resolves it with an optional note.
-- Status transitions are open -> resolved (one-way for now).
CREATE TABLE IF NOT EXISTS invoice_disputes (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- At most one OPEN dispute per invoice (re-raising updates the existing one).
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_disputes_one_open
    ON invoice_disputes (invoice_id)
    WHERE status = 'open';

-- Admin list is tenant-scoped and filtered by status.
CREATE INDEX IF NOT EXISTS idx_invoice_disputes_tenant_status
    ON invoice_disputes (tenant_id, status);

-- Portal reads the customer's own disputes.
CREATE INDEX IF NOT EXISTS idx_invoice_disputes_customer
    ON invoice_disputes (customer_id);
