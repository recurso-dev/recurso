-- Hot-path indexes for the invoices table — the busiest surface in the
-- product, which until now had NO index on tenant, customer, subscription,
-- gateway-payment, or the dunning due-date shape (PG does not auto-index FK
-- columns). Every shape below is a verbatim production query:
--
--   1. WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT/OFFSET — the
--      dashboard invoice list (+ its COUNT) on every page view.
--   2. WHERE gateway_payment_id = $1 ORDER BY created_at DESC LIMIT 1 —
--      EVERY GoCardless/async settlement webhook resolves the invoice this way.
--   3. WHERE status IN ('open','past_due') AND due_date < now — the dunning
--      sweep and retry workers.
--   4/5. customer_id / subscription_id lookups — customer detail pages,
--      renewal + final-usage flows, credit application.
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_created
    ON invoices (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invoices_gateway_payment
    ON invoices (gateway_payment_id)
    WHERE gateway_payment_id IS NOT NULL AND gateway_payment_id <> '';
CREATE INDEX IF NOT EXISTS idx_invoices_due_open
    ON invoices (due_date)
    WHERE status IN ('open', 'past_due');
CREATE INDEX IF NOT EXISTS idx_invoices_customer
    ON invoices (customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_subscription
    ON invoices (subscription_id);
