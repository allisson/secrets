-- Create tokenization_keys table
CREATE TABLE IF NOT EXISTS tokenization_keys (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    format_type TEXT NOT NULL,
    is_deterministic BOOLEAN NOT NULL,
    dek_id UUID NOT NULL REFERENCES deks(id),
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    UNIQUE (name, version)
);

CREATE INDEX idx_tokenization_keys_name ON tokenization_keys(name) WHERE deleted_at IS NULL;

-- Create tokenization_tokens table
CREATE TABLE IF NOT EXISTS tokenization_tokens (
    id UUID PRIMARY KEY,
    tokenization_key_id UUID NOT NULL REFERENCES tokenization_keys(id),
    token TEXT NOT NULL,
    value_hash TEXT,
    ciphertext BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    UNIQUE (token)
);

CREATE INDEX idx_tokenization_tokens_key_id ON tokenization_tokens(tokenization_key_id);
CREATE INDEX idx_tokenization_tokens_value_hash ON tokenization_tokens(value_hash) WHERE value_hash IS NOT NULL;
CREATE INDEX idx_tokenization_tokens_created_at ON tokenization_tokens(created_at);
CREATE INDEX idx_tokenization_tokens_expires_at ON tokenization_tokens(expires_at) WHERE expires_at IS NOT NULL;
