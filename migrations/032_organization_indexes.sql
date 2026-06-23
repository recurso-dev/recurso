CREATE INDEX IF NOT EXISTS idx_tenants_organization_id ON tenants(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_organizations_owner_email ON organizations(owner_email);
