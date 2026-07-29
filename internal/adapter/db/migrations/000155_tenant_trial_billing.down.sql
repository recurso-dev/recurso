DROP INDEX IF EXISTS idx_tenants_trial_ends_at;
DROP INDEX IF EXISTS idx_tenants_billing_status;
ALTER TABLE tenants
    DROP COLUMN IF EXISTS plan_tier,
    DROP COLUMN IF EXISTS billing_status,
    DROP COLUMN IF EXISTS trial_ends_at;
