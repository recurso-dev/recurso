-- Add key_hash and key_prefix columns to api_keys for secure key storage
-- key_prefix stores first 8 chars for DB lookup, key_hash stores bcrypt hash for verification
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_hash TEXT;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_prefix VARCHAR(8);

-- Create index on key_prefix for fast lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix) WHERE is_active = TRUE;

-- Migrate existing plaintext keys: set prefix from existing key_value
-- (Hash migration would need to be done via application code since bcrypt needs a runtime)
UPDATE api_keys SET key_prefix = LEFT(key_value, 8) WHERE key_prefix IS NULL AND key_value IS NOT NULL;
