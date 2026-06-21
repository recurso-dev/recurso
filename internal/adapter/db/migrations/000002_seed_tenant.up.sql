-- Seed the Default Tenant
INSERT INTO tenants (id, name, api_key_hash)
VALUES (
    '00000000-0000-0000-0000-000000000001', 
    'Recurso Default Tenant', 
    'mock_hash_123'
) ON CONFLICT (id) DO NOTHING;

-- Seed the requested Test User (as a Customer for now, or Admin if we had that)
-- If we treat swapnil.go20@gmail.com as a Customer:
-- INSERT INTO customers (id, tenant_id, email, name, ledger_account_id) ... 
-- Skipping explicit Customer insert to allow API testing to do it.
