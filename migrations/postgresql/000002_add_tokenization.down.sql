-- Drop tokenization_tokens table
DROP INDEX IF EXISTS idx_tokenization_tokens_expires_at;
DROP INDEX IF EXISTS idx_tokenization_tokens_created_at;
DROP INDEX IF EXISTS idx_tokenization_tokens_value_hash;
DROP INDEX IF EXISTS idx_tokenization_tokens_key_id;
DROP TABLE IF EXISTS tokenization_tokens;

-- Drop tokenization_keys table
DROP INDEX IF EXISTS idx_tokenization_keys_name;
DROP TABLE IF EXISTS tokenization_keys;
