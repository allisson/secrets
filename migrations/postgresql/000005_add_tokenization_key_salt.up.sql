-- Add salt column to tokenization_keys table
ALTER TABLE tokenization_keys ADD COLUMN salt BYTEA;
