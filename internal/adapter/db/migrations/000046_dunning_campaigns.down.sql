DROP INDEX IF EXISTS idx_dunning_executions_due;
DROP INDEX IF EXISTS idx_dunning_executions_invoice;
DROP TABLE IF EXISTS dunning_campaign_executions;
DROP TABLE IF EXISTS dunning_campaign_steps;
DROP TABLE IF EXISTS dunning_campaigns;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_wall_active;
