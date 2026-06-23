-- Clear plaintext key values for all keys that have been hashed
UPDATE api_keys SET key_value = '' WHERE key_hash IS NOT NULL AND key_value != '';
