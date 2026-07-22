ALTER TABLE credit_notes 
    ADD COLUMN created_by UUID REFERENCES users(id),
    ADD COLUMN approved_by UUID REFERENCES users(id),
    ADD COLUMN approved_at TIMESTAMP WITH TIME ZONE;

-- Enforce the allowed credit-note statuses for NEW/updated rows, but add the
-- constraint NOT VALID so it does NOT re-check pre-existing rows. Production
-- carried legacy credit notes with statuses 'open'/'applied'; a plain (validating)
-- CHECK rejects them — which is exactly what left this migration dirty at version
-- 126 and blocked every migration after it (incl. subscriptions.entity_id in
-- 000130), causing the 2026-07-22 subscriptions outage. NOT VALID enforces the
-- correct set going forward while tolerating legacy data, so a fresh restore of
-- production data can never re-hit this edge case. (Legacy rows are normalized
-- separately, as a deliberate data decision — not silently by this migration.)
ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS credit_notes_status_check;
ALTER TABLE credit_notes ADD CONSTRAINT credit_notes_status_check
    CHECK (status IN ('issued', 'used', 'void', 'pending_approval', 'rejected')) NOT VALID;
