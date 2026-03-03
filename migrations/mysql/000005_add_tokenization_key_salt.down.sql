-- Remove salt column from tokenization_keys table
ALTER TABLE tokenization_keys DROP COLUMN salt;
