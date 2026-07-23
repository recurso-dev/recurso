-- Ledger-backed credits, increment 2: give adjustment credits an optional
-- expiry, and a terminal 'expired' status the sweep sets when they lapse.
ALTER TABLE credit_notes ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

-- Widen the status CHECK to admit 'expired'. Kept NOT VALID (like 000126) so it
-- tolerates any legacy rows without validating the whole table under a lock.
ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS credit_notes_status_check;
ALTER TABLE credit_notes ADD CONSTRAINT credit_notes_status_check
    CHECK (status IN ('issued', 'used', 'void', 'pending_approval', 'rejected', 'expired')) NOT VALID;

-- The expiry sweep claims only still-spendable ('issued') dated credits; a
-- partial index keeps that poll cheap regardless of how many notes exist.
CREATE INDEX IF NOT EXISTS idx_credit_notes_due_expiry
    ON credit_notes (expires_at)
    WHERE expires_at IS NOT NULL AND status = 'issued';
