-- Create clients table
CREATE TABLE IF NOT EXISTS clients (
    id BINARY(16) PRIMARY KEY,
    secret VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL,
    created_at DATETIME(6) NOT NULL
);

-- Create tokens table
CREATE TABLE IF NOT EXISTS tokens (
    id BINARY(16) PRIMARY KEY,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    client_id BINARY(16) NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6),
    created_at DATETIME(6) NOT NULL,
    CONSTRAINT fk_tokens_client_id FOREIGN KEY (client_id) REFERENCES clients(id)
);

CREATE INDEX idx_tokens_client_id ON tokens(client_id);

-- Create policies table
CREATE TABLE IF NOT EXISTS policies (
    id BINARY(16) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    document JSON NOT NULL,
    created_at DATETIME(6) NOT NULL
);

-- Create client_policies table
CREATE TABLE IF NOT EXISTS client_policies (
    client_id BINARY(16) NOT NULL,
    policy_id BINARY(16) NOT NULL,
    PRIMARY KEY (client_id, policy_id),
    CONSTRAINT fk_client_policies_client_id FOREIGN KEY (client_id) REFERENCES clients(id),
    CONSTRAINT fk_client_policies_policy_id FOREIGN KEY (policy_id) REFERENCES policies(id)
);

-- Create keks table
CREATE TABLE IF NOT EXISTS keks (
    id BINARY(16) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    algorithm VARCHAR(255) NOT NULL,
    encrypted_key BLOB NOT NULL,
    nonce BLOB NOT NULL,
    version INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL,
    created_at DATETIME(6) NOT NULL
);

-- Create deks table
CREATE TABLE IF NOT EXISTS deks (
    id BINARY(16) PRIMARY KEY,
    kek_id BINARY(16) NOT NULL,
    algorithm VARCHAR(255) NOT NULL,
    encrypted_key BLOB NOT NULL,
    nonce BLOB NOT NULL,
    created_at DATETIME(6) NOT NULL,
    CONSTRAINT fk_deks_kek_id FOREIGN KEY (kek_id) REFERENCES keks(id)
);

-- Create secrets table
CREATE TABLE IF NOT EXISTS secrets (
    id BINARY(16) PRIMARY KEY,
    path VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME(6) NOT NULL,
    deleted_at DATETIME(6)
);

-- Create secret_versions table
CREATE TABLE IF NOT EXISTS secret_versions (
    id BINARY(16) PRIMARY KEY,
    secret_id BINARY(16) NOT NULL,
    version INTEGER NOT NULL,
    dek_id BINARY(16) NOT NULL,
    ciphertext BLOB NOT NULL,
    nonce BLOB NOT NULL,
    created_at DATETIME(6) NOT NULL,
    UNIQUE KEY uk_secret_versions (secret_id, version),
    CONSTRAINT fk_secret_versions_secret_id FOREIGN KEY (secret_id) REFERENCES secrets(id),
    CONSTRAINT fk_secret_versions_dek_id FOREIGN KEY (dek_id) REFERENCES deks(id)
);

CREATE INDEX idx_secret_versions_secret_id ON secret_versions(secret_id);

-- Create transit_keys table
CREATE TABLE IF NOT EXISTS transit_keys (
    id BINARY(16) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME(6) NOT NULL,
    deleted_at DATETIME(6)
);

-- Create transit_key_versions table
CREATE TABLE IF NOT EXISTS transit_key_versions (
    id BINARY(16) PRIMARY KEY,
    transit_key_id BINARY(16) NOT NULL,
    version INTEGER NOT NULL,
    dek_id BINARY(16) NOT NULL,
    is_primary BOOLEAN NOT NULL,
    created_at DATETIME(6) NOT NULL,
    UNIQUE KEY uk_transit_key_versions (transit_key_id, version),
    CONSTRAINT fk_transit_key_versions_key_id FOREIGN KEY (transit_key_id) REFERENCES transit_keys(id),
    CONSTRAINT fk_transit_key_versions_dek_id FOREIGN KEY (dek_id) REFERENCES deks(id)
);

CREATE INDEX idx_transit_key_versions_key_id ON transit_key_versions(transit_key_id);

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id BINARY(16) PRIMARY KEY,
    previous_hash BLOB,
    entry_hash BLOB NOT NULL,
    actor VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    metadata JSON,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
