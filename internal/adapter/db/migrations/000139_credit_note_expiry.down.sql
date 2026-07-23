DROP INDEX IF EXISTS idx_credit_notes_due_expiry;
ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS credit_notes_status_check;
ALTER TABLE credit_notes ADD CONSTRAINT credit_notes_status_check
    CHECK (status IN ('issued', 'used', 'void', 'pending_approval', 'rejected')) NOT VALID;
ALTER TABLE credit_notes DROP COLUMN IF EXISTS expires_at;
