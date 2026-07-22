ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS credit_notes_status_check;

-- Restore the original check constraint
ALTER TABLE credit_notes
    ADD CONSTRAINT credit_notes_status_check CHECK (status IN ('issued', 'used', 'void'));

ALTER TABLE credit_notes 
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS approved_at;
