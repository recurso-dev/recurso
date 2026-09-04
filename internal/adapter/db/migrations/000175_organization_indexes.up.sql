-- Ported from the orphaned top-level migrations/032_organization_indexes.sql,
-- which lived outside the embedded migrations directory and therefore never
-- ran. Both indexes back the organization → tenants join and the owner-email
-- lookup used by the Organizations pages.
CREATE INDEX IF NOT EXISTS idx_tenants_organization_id ON tenants(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_organizations_owner_email ON organizations(owner_email);
