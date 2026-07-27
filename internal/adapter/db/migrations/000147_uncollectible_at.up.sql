-- Recovery-rate cohort (QA finding D): the funnel's recovery_rate previously
-- divided ALL-TIME recovered count by the CURRENT uncollectible snapshot — two
-- different populations over two different windows, so the KPI drifted upward
-- as the business aged. A windowed rate needs the write-off moment, which was
-- never stored. Backfill uses updated_at as a best-effort approximation for
-- rows written off before this column existed (an uncollectible invoice is
-- rarely touched after write-off).
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS marked_uncollectible_at TIMESTAMPTZ;
UPDATE invoices SET marked_uncollectible_at = updated_at
 WHERE status = 'uncollectible' AND marked_uncollectible_at IS NULL;
