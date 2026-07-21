-- EU e-invoicing (Track C, increment 2b). Delivery to a Peppol Access Point can
-- fail transiently, so a stored document needs retry state to be redriven by a
-- background worker — mirroring the India IRN retry, but scoped to the EU record.
--
-- A row's document is generated once and is immutable, so the retriable failure
-- is *transmission*: status='failed' WITH a non-empty document. Generation
-- failures (bad data — missing VAT id, invalid country) are also status='failed'
-- but leave next_retry_at NULL, so the worker never picks them up (retrying a
-- deterministic validation failure is pointless); they surface for manual fix.

ALTER TABLE eu_einvoices
    -- Peppol routing needs the recipient's participant identifier; store it so a
    -- redrive can re-transmit the stored document without re-deriving it from the
    -- customer.
    ADD COLUMN IF NOT EXISTS recipient_vat_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0,
    -- Due time for the next delivery attempt. NULL = not scheduled for retry
    -- (delivered, or a non-retriable generation failure). TIMESTAMPTZ so the
    -- worker's UTC claim comparison can never skew by the server's offset.
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

-- Partial index over just the redrive-eligible rows keeps the worker's claim
-- query cheap regardless of how many delivered/never-scheduled records pile up.
CREATE INDEX IF NOT EXISTS idx_eu_einvoices_retry_due
    ON eu_einvoices (next_retry_at)
    WHERE next_retry_at IS NOT NULL;
