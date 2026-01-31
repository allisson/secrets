-- Create clients table
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY,
    secret TEXT NOT NULL,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- Create tokens table
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    client_id UUID NOT NULL REFERENCES clients(id),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_tokens_client_id ON tokens(client_id);

-- Create policies table
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    document JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- Create client_policies table
CREATE TABLE IF NOT EXISTS client_policies (
    client_id UUID NOT NULL REFERENCES clients(id),
    policy_id UUID NOT NULL REFERENCES policies(id),
    PRIMARY KEY (client_id, policy_id)
);

-- Create keks table
CREATE TABLE IF NOT EXISTS keks (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    algorithm TEXT NOT NULL,
    encrypted_key BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    version INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- Create deks table
CREATE TABLE IF NOT EXISTS deks (
    id UUID PRIMARY KEY,
    kek_id UUID NOT NULL REFERENCES keks(id),
    algorithm TEXT NOT NULL,
    encrypted_key BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- Create secrets table
CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);

-- Create secret_versions table
CREATE TABLE IF NOT EXISTS secret_versions (
    id UUID PRIMARY KEY,
    secret_id UUID NOT NULL REFERENCES secrets(id),
    version INTEGER NOT NULL,
    dek_id UUID NOT NULL REFERENCES deks(id),
    ciphertext BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (secret_id, version)
);

CREATE INDEX idx_secret_versions_secret_id ON secret_versions(secret_id);

-- Create transit_keys table
CREATE TABLE IF NOT EXISTS transit_keys (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);

-- Create transit_keys table
CREATE TABLE IF NOT EXISTS transit_key_versions (
    id UUID PRIMARY KEY,
    transit_key_id UUID NOT NULL REFERENCES transit_keys(id),
    version INTEGER NOT NULL,
    dek_id UUID NOT NULL REFERENCES deks(id),
    is_primary BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (transit_key_id, version)
);

CREATE INDEX idx_transit_key_versions_key_id ON transit_key_versions(transit_key_id);

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    previous_hash BYTEA,
    entry_hash BYTEA NOT NULL,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
