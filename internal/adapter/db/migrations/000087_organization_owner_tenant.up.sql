-- Give every organization an owning tenant so the /v1/organizations routes can
-- be scoped to the caller. Before this column existed the entire Organizations
-- subsystem was cross-tenant: any authenticated tenant could list or read ANY
-- organization, attach a victim's tenant to their own org, and then read that
-- victim's consolidated MRR. owner_tenant_id is the tenant that created the org;
-- every operation is now gated on it. Existing rows (owner NULL) fail closed —
-- they become inaccessible until an owner is backfilled, which is the safe
-- default for a data-isolation fix.
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS owner_tenant_id UUID REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_organizations_owner_tenant_id ON organizations(owner_tenant_id);
