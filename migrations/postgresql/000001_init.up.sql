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
    master_key_id TEXT NOT NULL,
    algorithm TEXT NOT NULL,
    encrypted_key BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    version INTEGER NOT NULL,
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
    path TEXT NOT NULL,
    version INTEGER NOT NULL,
    dek_id UUID NOT NULL REFERENCES deks(id),
    ciphertext BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    UNIQUE (path, version)
);

-- Create transit_keys table
CREATE TABLE IF NOT EXISTS transit_keys (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    dek_id UUID NOT NULL REFERENCES deks(id),
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    UNIQUE (name, version)
);

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
