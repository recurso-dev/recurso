ALTER TABLE coupons ADD COLUMN tenant_id UUID;
-- We should populate existing coupons if any, but default to NULL or a specific tenant is tricky.
-- For now, let's just add it. If we want to enforce NOT NULL, we'd need a default.
-- Assuming dev env, we can truncate or just allow NULL for legacy.
-- Better approach for dev: Allow NULL initially, then maybe enforce later.
-- Or just make it NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000' (Nil UUID) if no tenants exist.
-- But we have tenants. Let's make it nullable for now to fit existing data, but application will expect it.

-- For uniqueness, if there was a UNIQUE(code), we want UNIQUE(tenant_id, code).
-- DROP INDEX IF EXISTS coupons_code_key; -- if it was named that
-- ALTER TABLE coupons ADD CONSTRAINT coupons_code_tenant_unique UNIQUE (tenant_id, code);
