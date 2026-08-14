-- Invoice status history (lifecycle-history increment 2). An invoice moves
-- through draft → open → paid / past_due / uncollectible / void, but status is
-- mutated from a dozen money-path code paths (settlement, reversal, dunning,
-- write-off, void, finalize). Rather than instrument each one — invasive and
-- risky in the money path — a trigger captures EVERY transition at the source of
-- truth, the invoices row itself. Append-only; no Go/money-path code changes.
-- The DB has no request actor, so no changed_by is recorded — most transitions
-- are automated (settlement, dunning) where there is no user anyway.
CREATE TABLE IF NOT EXISTS invoice_status_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    invoice_id  UUID NOT NULL,
    from_status TEXT,               -- null on the first (creation) row
    to_status   TEXT NOT NULL,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_status_history_invoice
    ON invoice_status_history (invoice_id, changed_at);

-- Records the invoice's status at creation and on every subsequent change.
-- Non-status updates run the check but insert nothing (IS DISTINCT FROM), so
-- the common path stays cheap.
CREATE OR REPLACE FUNCTION record_invoice_status_change() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO invoice_status_history (tenant_id, invoice_id, from_status, to_status, changed_at)
        VALUES (NEW.tenant_id, NEW.id, NULL, NEW.status, COALESCE(NEW.created_at, NOW()));
    ELSIF (TG_OP = 'UPDATE' AND NEW.status IS DISTINCT FROM OLD.status) THEN
        INSERT INTO invoice_status_history (tenant_id, invoice_id, from_status, to_status, changed_at)
        VALUES (NEW.tenant_id, NEW.id, OLD.status, NEW.status, NOW());
    END IF;
    RETURN NULL; -- AFTER trigger: return value is ignored
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS invoice_status_history_trg ON invoices;
CREATE TRIGGER invoice_status_history_trg
    AFTER INSERT OR UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION record_invoice_status_change();
