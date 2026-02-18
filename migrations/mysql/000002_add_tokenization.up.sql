-- Create tokenization_keys table
CREATE TABLE IF NOT EXISTS tokenization_keys (
    id BINARY(16) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    format_type VARCHAR(50) NOT NULL,
    is_deterministic BOOLEAN NOT NULL,
    dek_id BINARY(16) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    deleted_at DATETIME(6),
    UNIQUE KEY unique_name_version (name, version),
    CONSTRAINT fk_tokenization_keys_dek_id FOREIGN KEY (dek_id) REFERENCES deks(id),
    INDEX idx_tokenization_keys_name (name, deleted_at)
);

-- Create tokenization_tokens table
CREATE TABLE IF NOT EXISTS tokenization_tokens (
    id BINARY(16) PRIMARY KEY,
    tokenization_key_id BINARY(16) NOT NULL,
    token VARCHAR(255) NOT NULL,
    value_hash VARCHAR(64),
    ciphertext BLOB NOT NULL,
    nonce BLOB NOT NULL,
    metadata JSON,
    created_at DATETIME(6) NOT NULL,
    expires_at DATETIME(6),
    revoked_at DATETIME(6),
    UNIQUE KEY unique_token (token),
    CONSTRAINT fk_tokenization_tokens_key_id FOREIGN KEY (tokenization_key_id) REFERENCES tokenization_keys(id),
    INDEX idx_tokenization_tokens_key_id (tokenization_key_id),
    INDEX idx_tokenization_tokens_value_hash (value_hash),
    INDEX idx_tokenization_tokens_created_at (created_at),
    INDEX idx_tokenization_tokens_expires_at (expires_at)
);
