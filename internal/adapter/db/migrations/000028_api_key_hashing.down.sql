-- Rollback key hashing migration
DROP INDEX IF EXISTS idx_api_keys_prefix;
ALTER TABLE api_keys DROP COLUMN IF EXISTS key_hash;
ALTER TABLE api_keys DROP COLUMN IF EXISTS key_prefix;
