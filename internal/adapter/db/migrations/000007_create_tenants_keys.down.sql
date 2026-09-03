-- 000007 only ADDED columns to tenants (owned by 000001) and created api_keys.
-- The previous down dropped the tenants table itself, which fails on every
-- database because plans/subscriptions/invoices reference it, so `migrate
-- down` could never pass this version.
DROP TABLE IF EXISTS api_keys;
ALTER TABLE tenants DROP COLUMN IF EXISTS updated_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS email;
