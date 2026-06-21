-- Smart Dunning: track RL action selection on invoices
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS dunning_action_id VARCHAR(20);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS dunning_context_key VARCHAR(100);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS last_payment_error VARCHAR(100);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS dunning_managed_by VARCHAR(20) DEFAULT 'scheduler';

-- Index for worker/scheduler partition queries
CREATE INDEX IF NOT EXISTS idx_invoices_dunning_managed ON invoices(dunning_managed_by) WHERE status IN ('open', 'past_due');
