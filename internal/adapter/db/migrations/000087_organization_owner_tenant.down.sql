DROP INDEX IF EXISTS idx_organizations_owner_tenant_id;
ALTER TABLE organizations DROP COLUMN IF EXISTS owner_tenant_id;
