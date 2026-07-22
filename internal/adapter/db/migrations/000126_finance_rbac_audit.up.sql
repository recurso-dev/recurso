ALTER TABLE credit_notes 
    ADD COLUMN created_by UUID REFERENCES users(id),
    ADD COLUMN approved_by UUID REFERENCES users(id),
    ADD COLUMN approved_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE credit_notes DROP CONSTRAINT IF EXISTS credit_notes_status_check;
ALTER TABLE credit_notes ADD CONSTRAINT credit_notes_status_check 
    CHECK (status IN ('issued', 'used', 'void', 'pending_approval', 'rejected'));
