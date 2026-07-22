ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS credit_notes_status_check;

-- Restore the original check constraint, NOT VALID so rolling back never fails on
-- rows whose status was added since (pending_approval/rejected) or on legacy values.
ALTER TABLE credit_notes
    ADD CONSTRAINT credit_notes_status_check CHECK (status IN ('issued', 'used', 'void')) NOT VALID;

ALTER TABLE credit_notes 
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS approved_at;
