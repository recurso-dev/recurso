DROP INDEX IF EXISTS uq_api_keys_key_lookup;
ALTER TABLE api_keys DROP COLUMN IF EXISTS key_lookup;
