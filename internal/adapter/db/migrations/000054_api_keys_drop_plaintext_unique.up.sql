-- Since API keys became bcrypt-hashed (000028), key_value is always stored
-- as the empty string, so its UNIQUE constraint makes the SECOND tenant
-- registration in any database fail. The hash is the real credential;
-- uniqueness of plaintext is meaningless now.
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_key_value_key;
