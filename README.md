# 🔐 Secrets

> A production-ready secrets management system implementing envelope encryption with Clean Architecture principles.

[![CI](https://github.com/allisson/secrets/workflows/CI/badge.svg)](https://github.com/allisson/secrets/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/allisson/secrets)](https://goreportcard.com/report/github.com/allisson/secrets)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Secrets is a secure key management and secrets storage system built with Go, designed for applications requiring enterprise-grade cryptographic operations. It implements a three-tier envelope encryption architecture that provides efficient key rotation, fine-grained access control, and comprehensive audit logging.

## 📚 Table of Contents

- [✨ Features](#-features)
- [🏗️ Architecture](#️-architecture)
- [🚀 Quick Start](#-quick-start)
- [📦 Installation](#-installation)
- [⚙️ Configuration](#️-configuration)
- [💻 Usage](#-usage)
- [📖 API Reference](#-api-reference)
- [🚧 Planned Features](#-planned-features)
- [🛠️ Development](#️-development)
- [🧪 Testing](#-testing)
- [🔒 Security](#-security)
- [📄 License](#-license)

## ✨ Features

### 🔑 Core Cryptographic Capabilities

- 🎯 **Envelope Encryption** - Three-tier key hierarchy (Master Key → KEK → DEK → Data) for efficient key rotation
- 🔐 **Multiple Algorithms** - Support for AES-256-GCM and ChaCha20-Poly1305 AEAD encryption
- 🔄 **Key Versioning** - Built-in key rotation with backward compatibility for decryption
- 🚄 **Transit Encryption** - Encrypt/decrypt data without storing it (encryption-as-a-service)

### 📦 Secrets Management

- 📚 **Automatic Versioning** - Every secret update creates a new version, preserving complete history
- 🗂️ **Path-based Organization** - Hierarchical secret organization (e.g., `/app/prod/db-password`)
- 🗑️ **Soft Deletion** - Mark secrets as deleted without losing historical data
- 🔄 **Immutable History** - Each version is a separate database record with its own DEK
- 📝 **Version Tracking** - Retrieve current or specific historical versions of secrets
- 🌐 **REST API** - Full CRUD operations via HTTP endpoints with authentication and authorization

### 🛡️ Access Control & Authentication

- 👤 **Client Authentication** - API clients with secret-based authentication
- 🎫 **Token Management** - Time-limited tokens with expiration, revocation, and client credential exchange
- 📋 **Policy-based Authorization** - JSON policy documents for fine-grained access control
- 🔗 **Client-Policy Binding** - Associate multiple policies with each client
- 🔌 **Client Management API** - REST endpoints for CRUD operations on API clients (Create, Read, Update, Delete)

### 🔒 Security & Compliance

- 📜 **Immutable Audit Logs** - Append-only audit trail for compliance and security monitoring
- 📊 **Comprehensive Logging** - Track all operations with actor, action, resource, and metadata
- 🔍 **Audit Log API** - Query and retrieve audit logs with pagination for compliance reporting and security analysis
- 🧹 **Secure Memory Handling** - Automatic zeroing of sensitive key material
- 💾 **Database Encryption** - All sensitive data encrypted at rest

### 🏗️ Architecture & Design

- 🎯 **Clean Architecture** - Clear separation of domain, use case, repository, and presentation layers
- 🧩 **Domain-Driven Design** - Business logic encapsulated in domain models
- 🗄️ **Multi-Database Support** - PostgreSQL and MySQL with dedicated repository implementations
- 🔑 **Complete Repository Layer** - KEK and DEK repositories with transaction support and database-specific optimizations
- ⚡ **Transaction Management** - ACID guarantees for atomic operations (key rotation, secret updates)
- 💉 **Dependency Injection** - Centralized wiring with lazy initialization
- 🌐 **Gin Web Framework** - High-performance HTTP router (v1.11.0) with custom slog middleware, REST API under `/v1/` (clients, secrets), and capability-based authorization

## 🏗️ Architecture

### 🔐 Envelope Encryption Model

The system implements a complete three-tier envelope encryption architecture with full repository layer support for both PostgreSQL and MySQL:

```
┌─────────────────┐
│   Master Key    │  (Environment/KMS - Root of trust)
│   256-bit AES   │
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│ Key Encryption  │  (Database - Encrypted with master key)
│  Key (KEK)      │  Version: 1, 2, 3... (rotation support)
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│ Data Encryption │  (Database - Per-secret encryption key)
│  Key (DEK)      │  One DEK per secret version
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│ Secret Data     │  (Ciphertext stored with secret)
└─────────────────┘
```

### 🚄 Transit Encryption Model

Transit Encryption extends the envelope encryption hierarchy with a fourth tier for encryption-as-a-service operations:

```
┌─────────────────┐
│   Master Key    │  (Environment/KMS - Root of trust)
│   256-bit AES   │
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│ Key Encryption  │  (Database - Encrypted with master key)
│  Key (KEK)      │  Version: 1, 2, 3... (rotation support)
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│ Data Encryption │  (Database - Per-transit key version)
│  Key (DEK)      │  One DEK per transit key version
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│  Transit Key    │  (Database - Versioned encryption keys)
│  (Versioned)    │  Used for application data encryption
└────────┬────────┘
         │ encrypts
         ▼
┌─────────────────┐
│ Application     │  (NOT stored - returned to client)
│ Data (Plaintext)│  Client handles ciphertext
└─────────────────┘
```

**Key Differences: Secrets Management vs Transit Encryption**

| Aspect | Secrets Management | Transit Encryption |
|--------|-------------------|-------------------|
| **Data Storage** | Encrypted data stored in database | Data NOT stored (encryption-as-a-service) |
| **Use Case** | Long-term secret storage (passwords, API keys) | Encrypt data for external storage (databases, logs) |
| **Key Hierarchy** | Master Key → KEK → DEK → Secret Data | Master Key → KEK → DEK → Transit Key → App Data |
| **Versioning** | Secret versions (each with own DEK) | Transit key versions (each with own DEK) |
| **Client Receives** | Plaintext secret value | Versioned ciphertext string |
| **Ciphertext Format** | Internal (database only) | `"version:base64-ciphertext"` (e.g., `"1:ZW5jcnlwdGVk..."`) |
| **Key Rotation** | Transparent (old versions remain readable) | Transparent (version prefix enables decryption) |
| **Ideal For** | Credentials, tokens, certificates | PII, sensitive logs, database fields |

### 🗄️ Database Schema

The system uses the following core tables for key management and secret storage:

**Key Encryption Keys (KEKs) Table:**
- Stores encrypted KEKs that are encrypted with the master key
- Fields: `id` (UUID), `master_key_id` (TEXT), `algorithm` (TEXT), `encrypted_key` (BYTEA/BLOB), `nonce` (BYTEA/BLOB), `version` (INTEGER), `created_at` (TIMESTAMPTZ/DATETIME)
- Each KEK version enables key rotation without re-encrypting all DEKs

**Data Encryption Keys (DEKs) Table:**
- Stores encrypted DEKs that are encrypted with KEKs
- Fields: `id` (UUID), `kek_id` (UUID FK → keks.id), `algorithm` (TEXT), `encrypted_key` (BYTEA/BLOB), `nonce` (BYTEA/BLOB), `created_at` (TIMESTAMPTZ/DATETIME)
- One DEK per secret version for cryptographic isolation
- Foreign key relationship ensures DEKs are always associated with valid KEKs

**Secrets Table:**
- Stores encrypted secret data with automatic versioning
- Fields: `id` (UUID), `path` (TEXT), `version` (INTEGER), `dek_id` (UUID FK → deks.id), `ciphertext` (BYTEA/BLOB), `nonce` (BYTEA/BLOB), `created_at` (TIMESTAMPTZ/DATETIME), `deleted_at` (TIMESTAMPTZ/DATETIME)
- Each version is a separate record with its own DEK
- Unique constraint on (path, version) ensures version consistency

**Database-Specific Handling:**
- **PostgreSQL**: Native UUID type, BYTEA for binary data, TIMESTAMPTZ for timestamps
- **MySQL**: BINARY(16) for UUIDs (with marshal/unmarshal), BLOB for binary data, DATETIME for timestamps

### 📚 Secret Versioning Model

Every time a secret is created or updated, a new version is created as a separate database record:

```
Path: /app/prod/api-key

┌─────────────────────────────────────────────────────────┐
│ Version 1 (Initial Creation)                            │
│ ID: 018d7e95-1a23...  Path: /app/prod/api-key          │
│ Version: 1            DEK ID: 018d7e95-2b34...          │
│ Created: 2026-02-01   Deleted: null                     │
└─────────────────────────────────────────────────────────┘
         │
         ▼ UPDATE (creates new version)
┌─────────────────────────────────────────────────────────┐
│ Version 2 (First Update)                                │
│ ID: 018d7e96-3c45...  Path: /app/prod/api-key          │
│ Version: 2            DEK ID: 018d7e96-4d56...          │
│ Created: 2026-02-02   Deleted: null                     │
└─────────────────────────────────────────────────────────┘
         │
         ▼ UPDATE (creates new version)
┌─────────────────────────────────────────────────────────┐
│ Version 3 (Second Update)                               │
│ ID: 018d7e97-5e67...  Path: /app/prod/api-key          │
│ Version: 3            DEK ID: 018d7e97-6f78...          │
│ Created: 2026-02-03   Deleted: null    ← CURRENT       │
└─────────────────────────────────────────────────────────┘
```

**Versioning Behavior:**
- 📝 **New Secret**: First creation sets version to 1
- 🔄 **Update Secret**: Creates a new record with version incremented (2, 3, 4...)
- 📖 **Get Secret**: Returns the highest version number (current)
- 🗑️ **Delete Secret**: Marks only the current version as deleted
- 🔒 **Independent Encryption**: Each version has its own DEK for security isolation
- 📜 **Immutable History**: Previous versions remain unchanged for audit trail

**Key Advantages:**
- ✅ Complete audit trail of all secret changes
- ✅ Ability to investigate security incidents using historical data
- ✅ Compliance with data retention policies
- ✅ Each version cryptographically isolated with its own DEK
- ✅ Safe rollback capability (future feature)

### ✅ Key Benefits

- ⚡ **Fast Key Rotation**: Rotate master keys or KEKs without re-encrypting all secrets
- 🔒 **Per-Secret Security**: Each secret version has its own DEK for maximum isolation
- 🎨 **Algorithm Flexibility**: Different encryption algorithms per key tier
- 📈 **Scalability**: Minimal performance impact from key rotation
- 📚 **Version Control**: Automatic versioning preserves complete secret history
- 🔄 **Backward Compatibility**: Old secret versions remain readable after KEK rotation

### 📁 Project Structure

```
secrets/
├── cmd/app/                    # Application entry point
│   └── main.go                 # CLI with server and migrate commands
├── internal/
│   ├── app/                    # Dependency injection container
│   ├── config/                 # Configuration management
│   ├── database/               # Database connection and transactions
│   ├── errors/                 # Standardized domain errors
│   ├── http/                   # HTTP server infrastructure
│   ├── httputil/               # HTTP utilities (JSON responses)
│   ├── validation/             # Custom validation rules
│   ├── testutil/               # Test utilities
│   ├── auth/                   # Authentication & authorization module
│   │   ├── domain/             # Entities: Client, Token, AuditLog, PolicyDocument
│   │   ├── service/            # Token and secret hashing services
│   │   ├── usecase/            # Client, Token, and AuditLog business logic
│   │   ├── repository/         # Data access (PostgreSQL & MySQL)
│   │   └── http/               # HTTP presentation layer
│   │       ├── client_handler.go         # Client management handlers
│   │       ├── client_handler_test.go    # Client handler tests
│   │       ├── token_handler.go          # Token issuance handlers
│   │       ├── token_handler_test.go     # Token handler tests
│   │       ├── middleware.go             # Auth & authz middleware
│   │       ├── middleware_test.go        # Middleware tests
│   │       ├── context.go                # Context helpers
│   │       ├── test_helpers.go           # Shared test utilities
│   │       ├── dto/                      # Data Transfer Objects
│   │       │   ├── request.go            # Request DTOs + validation
│   │       │   ├── request_test.go       # Request DTO tests
│   │       │   ├── response.go           # Response DTOs + mapping
│   │       │   └── response_test.go      # Response DTO tests
│   │       └── mocks/                    # Manual mocks
│   │           └── token_usecase.go      # Mock TokenUseCase
│   ├── crypto/                 # Cryptographic domain module
│   │   ├── domain/             # Entities: Kek, Dek, MasterKey
│   │   ├── service/            # Encryption services
│   │   ├── usecase/            # Business logic orchestration
│   │   └── repository/         # Data access: Kek and Dek repositories (PostgreSQL & MySQL)
│   └── secrets/                # Secrets management module
│       ├── domain/             # Entities: Secret (with versioning)
│       ├── usecase/            # Secret operations (create/update/get/delete)
│       ├── repository/         # Secret persistence (PostgreSQL & MySQL)
│       └── http/               # HTTP presentation layer
│           ├── secret_handler.go         # Secret management handlers
│           ├── secret_handler_test.go    # Secret handler tests
│           ├── test_helpers.go           # Shared test utilities
│           └── dto/                      # Data Transfer Objects
│               ├── request.go            # Request DTOs + validation
│               ├── request_test.go       # Request DTO tests
│               ├── response.go           # Response DTOs + mapping
│               └── response_test.go      # Response DTO tests
│   └── transit/                # Transit encryption module (encryption-as-a-service)
│       ├── domain/             # Entities: TransitKey, EncryptedBlob (versioning)
│       ├── usecase/            # Transit operations (create/rotate/encrypt/decrypt)
│       ├── repository/         # Transit key persistence (PostgreSQL & MySQL)
│       └── http/               # HTTP presentation layer
│           ├── transit_key_handler.go    # Transit key management handlers
│           ├── transit_key_handler_test.go  # Transit key handler tests
│           ├── crypto_handler.go         # Encrypt/decrypt handlers
│           ├── crypto_handler_test.go    # Crypto handler tests
│           ├── test_helpers.go           # Shared test utilities
│           └── dto/                      # Data Transfer Objects
│               ├── request.go            # Request DTOs + validation
│               ├── request_test.go       # Request DTO tests
│               ├── response.go           # Response DTOs + mapping
│               └── response_test.go      # Response DTO tests
├── migrations/
│   ├── postgresql/             # PostgreSQL migrations
│   └── mysql/                  # MySQL migrations
├── Dockerfile                  # Multi-stage Docker build
├── Makefile                    # Development and build commands
└── docker-compose.test.yml     # Test database setup
```

## 🚀 Quick Start

### ✅ Prerequisites

- Go 1.25 or higher
- PostgreSQL 12+ or MySQL 8.0+
- Docker (optional, for containerized databases)

### 🔧 Initial Setup Workflow

Follow these steps to get the system up and running:

```bash
# 1. Clone the repository
git clone https://github.com/allisson/secrets.git
cd secrets

# 2. Install dependencies
go mod download

# 3. Build the application
make build

# 4. Generate a master key
./bin/app create-master-key --id default

# 5. Copy the output and create .env file
cp .env.example .env
# Edit .env and paste the MASTER_KEYS and ACTIVE_MASTER_KEY_ID values

# 6. Start the database (using Docker)
make dev-postgres

# 7. Run database migrations
./bin/app migrate

# 8. Create the initial Key Encryption Key (KEK)
./bin/app create-kek

# 9. Create a bootstrap admin client
./bin/app create-client --name "bootstrap-admin" \
  --policies '[{"path":"*","capabilities":["read","write","delete","encrypt","decrypt","rotate"]}]'
# Save the client_id and secret from the output

# 9a. (Optional) List all clients to verify creation
# First, obtain a token using the client credentials from step 9
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "<client-id-from-step-9>",
    "client_secret": "<secret-from-step-9>"
  }'
# Save the token from the response

# List all clients
curl http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <token-from-previous-request>"
# You should see your bootstrap-admin client in the response

# 10. Start the server
./bin/app server
```

The server will be available at `http://localhost:8080`. Test with:

```bash
# Health check
curl http://localhost:8080/health

# Obtain an authentication token (use client_id and secret from step 9)
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "<client-id-from-step-9>",
    "client_secret": "<secret-from-step-9>"
  }'
# Save the token from the response

# Create your first secret (use the token from previous step)
# Note: value must be base64-encoded
curl -X POST http://localhost:8080/v1/secrets/app/production/database-password \
  -H "Authorization: Bearer <token-from-previous-step>" \
  -H "Content-Type: application/json" \
  -d '{"value":"bXktc3VwZXItc2VjcmV0LXBhc3N3b3Jk"}'

# Retrieve the secret
curl http://localhost:8080/v1/secrets/app/production/database-password \
  -H "Authorization: Bearer <token-from-previous-step>"

# (Optional) Test transit encryption for encryption-as-a-service
# 11. Create a transit encryption key
curl -X POST http://localhost:8080/v1/transit/keys \
  -H "Authorization: Bearer <token-from-previous-step>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-app-encryption",
    "algorithm": "aes-gcm"
  }'

# Encrypt data (server does NOT store the data)
# Note: plaintext must be base64-encoded
curl -X POST http://localhost:8080/v1/transit/keys/my-app-encryption/encrypt \
  -H "Authorization: Bearer <token-from-previous-step>" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c2Vuc2l0aXZlLWRhdGE="}'
# Save the ciphertext from response (format: "1:base64...")

# Decrypt data (returns base64-encoded plaintext)
curl -X POST http://localhost:8080/v1/transit/keys/my-app-encryption/decrypt \
  -H "Authorization: Bearer <token-from-previous-step>" \
  -H "Content-Type: application/json" \
  -d '{"ciphertext":"<ciphertext-from-previous-step>"}'
```

### 📦 Installation

```bash
# Clone the repository
git clone https://github.com/allisson/secrets.git
cd secrets

# Install dependencies
go mod download

# Build the application
make build

# Generate a master key using the built-in command
./bin/app create-master-key

# Copy the output environment variables to your .env file
cp .env.example .env

# Edit .env and paste the MASTER_KEYS and ACTIVE_MASTER_KEY_ID values
```

### 🐳 Running with Docker

```bash
# Start PostgreSQL database
make dev-postgres

# Build the application
make build

# Generate master key and copy to .env
./bin/app create-master-key --id default

# Edit .env file and paste the generated MASTER_KEYS and ACTIVE_MASTER_KEY_ID
nano .env

# Run database migrations
make run-migrate

# Create initial KEK
./bin/app create-kek

# Start the server
make run-server
```

The server will be available at `http://localhost:8080`.

### 🐳 Running with Docker Compose

```bash
# Build Docker image
make docker-build

# Run migrations
make docker-run-migrate

# Start server
make docker-run-server
```

## ⚙️ Configuration

Configuration is managed through environment variables. Create a `.env` file in the project root:

```bash
# Database Configuration
DB_DRIVER=postgres                          # Database driver: postgres or mysql
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/secrets?sslmode=disable
DB_MAX_OPEN_CONNECTIONS=25                  # Maximum open database connections
DB_MAX_IDLE_CONNECTIONS=5                   # Maximum idle database connections
DB_CONN_MAX_LIFETIME=5                      # Connection max lifetime (minutes)

# Server Configuration
SERVER_HOST=0.0.0.0                         # HTTP server bind address
SERVER_PORT=8080                            # HTTP server port

# Logging
LOG_LEVEL=info                              # Log level: debug, info, warn, error

# Master Keys (Envelope Encryption)
MASTER_KEYS=default:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=  # Format: id:base64key
ACTIVE_MASTER_KEY_ID=default                # ID of active master key for new KEKs

# Authentication
AUTH_TOKEN_EXPIRATION_SECONDS=86400         # Token lifetime in seconds (default: 86400 = 24 hours)

# Worker Configuration (for future async operations)
WORKER_INTERVAL=5                           # Worker polling interval (seconds)
WORKER_BATCH_SIZE=10                        # Batch size for processing
WORKER_MAX_RETRIES=3                        # Maximum retry attempts
WORKER_RETRY_INTERVAL=1                     # Retry interval (minutes)
```

### 🔑 Master Key Configuration

Master keys are the root of trust in the envelope encryption hierarchy. They are stored in environment variables:

- 🔑 **Format**: `MASTER_KEYS=id1:base64key1,id2:base64key2`
- 📏 **Key Size**: Each key must be exactly 32 bytes (256 bits), base64-encoded
- ⭐ **Active Key**: `ACTIVE_MASTER_KEY_ID` specifies which key encrypts new KEKs
- 🔄 **Rotation**: Add a new key to `MASTER_KEYS`, update `ACTIVE_MASTER_KEY_ID`, and rotate KEKs

#### Generate Master Key

The recommended way to generate a master key is using the built-in command:

```bash
# Generate with default ID (master-key-YYYY-MM-DD)
./bin/app create-master-key

# Generate with custom ID
./bin/app create-master-key --id prod-master-key-2025
```

**Output:**
```bash
# Master Key Configuration
# Copy these environment variables to your .env file or secrets manager

MASTER_KEYS="prod-master-key-2025:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y="
ACTIVE_MASTER_KEY_ID="prod-master-key-2025"

# For multiple master keys (key rotation), use comma-separated format:
# MASTER_KEYS="prod-master-key-2025:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=,new-key:base64-encoded-new-key"
# ACTIVE_MASTER_KEY_ID="new-key"
```

**Alternative Method:**

You can also generate a master key using OpenSSL:

```bash
# Generate a new 256-bit key
openssl rand -base64 32

# Set in environment
MASTER_KEYS=default:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=,backup:xYz123...
ACTIVE_MASTER_KEY_ID=default
```

## 💻 Usage

### 🔑 Generate Master Key

Before setting up the system, generate a cryptographically secure master key:

```bash
# Build the application
make build

# Generate master key with default ID
./bin/app create-master-key

# Or generate with custom ID
./bin/app create-master-key --id prod-master-key-2025
./bin/app create-master-key -i prod-master-key-2025  # Short flag
```

**Output:**
```
# Master Key Configuration
# Copy these environment variables to your .env file or secrets manager

MASTER_KEYS="prod-master-key-2025:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y="
ACTIVE_MASTER_KEY_ID="prod-master-key-2025"

# For multiple master keys (key rotation), use comma-separated format:
# MASTER_KEYS="prod-master-key-2025:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=,new-key:base64-encoded-new-key"
# ACTIVE_MASTER_KEY_ID="new-key"
```

**Security Notes:**
- 🔐 The command generates a cryptographically secure 32-byte (256-bit) key using `crypto/rand`
- 🧹 The key is automatically zeroed from memory after encoding
- 💾 Store the output securely in a secrets manager or encrypted vault
- ⚠️ Never commit master keys to version control
- 🏢 For production, consider using a proper KMS (AWS KMS, HashiCorp Vault, etc.)

### 🗄️ Database Migrations

```bash
# Run migrations
make run-migrate

# Or using Docker
make docker-run-migrate
```

### 🔑 Key Encryption Key (KEK) Management

#### Create Initial KEK

Before starting the server, you need to create the initial Key Encryption Key (KEK):

```bash
# Build the application
make build

# Create KEK with default algorithm (AES-GCM)
./bin/app create-kek

# Or specify ChaCha20-Poly1305 algorithm
./bin/app create-kek --algorithm chacha20-poly1305

# Or using short flag
./bin/app create-kek --alg chacha20-poly1305
```

**Requirements:**
- Database must be migrated (run `make run-migrate` first)
- Environment variables must be set: `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID`
- Should only be run once during initial setup

**Example:**
```bash
# Set up environment variables
export MASTER_KEYS="default:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y="
export ACTIVE_MASTER_KEY_ID="default"
export DB_DRIVER="postgres"
export DB_CONNECTION_STRING="postgres://user:password@localhost:5432/secrets?sslmode=disable"

# Run migrations
make run-migrate

# Create initial KEK
./bin/app create-kek --algorithm aes-gcm
```

**Output:**
```json
{"time":"2026-02-05T00:14:23Z","level":"INFO","msg":"creating new KEK","algorithm":"aes-gcm"}
{"time":"2026-02-05T00:14:23Z","level":"INFO","msg":"master key chain loaded","active_master_key_id":"default"}
{"time":"2026-02-05T00:14:23Z","level":"INFO","msg":"KEK created successfully","algorithm":"aes-gcm","master_key_id":"default"}
```

#### Rotate KEK

Rotate the Key Encryption Key to create a new version and mark the previous one as inactive. This is recommended every 90 days or when suspecting key compromise.

```bash
# Rotate KEK with default algorithm (AES-GCM)
./bin/app rotate-kek

# Or specify ChaCha20-Poly1305 algorithm
./bin/app rotate-kek --algorithm chacha20-poly1305

# Or using short flag
./bin/app rotate-kek --alg chacha20-poly1305
```

**Requirements:**
- An active KEK must already exist (run `create-kek` first)
- Database must be accessible
- Environment variables must be set: `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID`

**When to Rotate:**
- ⏰ **Regularly**: Every 90 days as a security best practice
- 🚨 **Security Incident**: When suspecting KEK compromise
- 🔄 **Algorithm Change**: When upgrading to a different encryption algorithm
- 🔑 **Master Key Rotation**: After rotating master keys

**Example:**
```bash
# Rotate to a new KEK version
./bin/app rotate-kek --algorithm aes-gcm
```

**Output:**
```json
{"time":"2026-02-05T00:14:23Z","level":"INFO","msg":"rotating KEK","algorithm":"aes-gcm"}
{"time":"2026-02-05T00:14:23Z","level":"INFO","msg":"master key chain loaded","active_master_key_id":"default"}
{"time":"2026-02-05T00:14:23Z","level":"INFO","msg":"KEK rotated successfully","algorithm":"aes-gcm","master_key_id":"default"}
```

**Important Notes:**
- ✅ Rotation is atomic - either fully succeeds or fully fails
- ✅ Old KEK remains in the database for decrypting existing DEKs
- ✅ New secrets will use the new KEK version
- ✅ Backward compatibility is maintained - existing secrets remain readable

### 🚀 Starting the Server

```bash
# Development mode
make run-server

# Production with Docker
make docker-run-server
```

### ❤️ Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2026-02-02T20:13:45Z"
}
```

## 📖 API Reference

### 🔐 Authentication

All API endpoints (except `/health` and `/ready`) require authentication using client tokens:

```bash
# Include token in Authorization header
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/clients
```

### 🎫 Token Issuance

Before using authenticated endpoints, clients must obtain a token using their client credentials.

#### Issue Token

Exchanges client credentials for a time-limited authentication token.

```bash
POST /v1/token
```

**Authentication:** Not required (this IS the authentication endpoint)

**Request Body:**
```json
{
  "client_id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "client_secret": "sec_1234567890abcdef"
}
```

**Response (201 Created):**
```json
{
  "token": "tok_abcdef1234567890...",
  "expires_at": "2026-02-13T20:13:45Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "018d7e95-1a23-7890-bcde-f1234567890a",
    "client_secret": "sec_1234567890abcdef"
  }'
```

**Error Responses:**
- `401 Unauthorized` - Invalid client credentials
- `403 Forbidden` - Client is not active
- `422 Unprocessable Entity` - Invalid request format or missing fields

**Token Details:**
- ⏱️ **Expiration**: Tokens expire after 24 hours (configurable via `AUTH_TOKEN_EXPIRATION_SECONDS`, default: 86400)
- 🔄 **Renewal**: Request a new token before expiration using the same credentials
- 🔒 **Storage**: Store tokens securely and never commit them to version control

### 👤 Client Management

Client management endpoints provide CRUD operations for API clients. All endpoints require authentication with a valid Bearer token and appropriate capabilities.

#### Authentication

All client management endpoints require authentication using Bearer tokens:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/clients
```

**Note:** The "Bearer" prefix is case-insensitive (`bearer`, `Bearer`, `BEARER` all work).

#### Create Client

Creates a new API client with a name and policy document.

```bash
POST /v1/clients
```

**Authentication:** Required  
**Authorization:** `WriteCapability` for path `/v1/clients`

**Request Body:**
```json
{
  "name": "production-app",
  "is_active": true,
  "policies": [
    {
      "path": "/v1/clients/*",
      "capabilities": ["read", "write"]
    },
    {
      "path": "/v1/secrets/*",
      "capabilities": ["read"]
    }
  ]
}
```

**Response (201 Created):**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "secret": "sec_1234567890abcdef"
}
```

**Important:** The client secret is only returned during creation and cannot be retrieved later. Store it securely.

**Example:**
```bash
curl -X POST http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-app",
    "is_active": true,
    "policies": [
      {"path": "*", "capabilities": ["read"]}
    ]
  }'
```

#### Get Client

Retrieves a client by ID. The client secret is never returned.

```bash
GET /v1/clients/:id
```

**Authentication:** Required  
**Authorization:** `ReadCapability` for path `/v1/clients/:id`

**Response (200 OK):**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "name": "production-app",
  "is_active": true,
  "policies": [
    {
      "path": "/v1/clients/*",
      "capabilities": ["read", "write"]
    }
  ],
  "created_at": "2026-02-12T20:13:45Z"
}
```

**Example:**
```bash
curl http://localhost:8080/v1/clients/018d7e95-1a23-7890-bcde-f1234567890a \
  -H "Authorization: Bearer <token>"
```

#### Update Client

Updates an existing client's name, active status, or policy document.

```bash
PUT /v1/clients/:id
```

**Authentication:** Required  
**Authorization:** `WriteCapability` for path `/v1/clients/:id`

**Request Body:**
```json
{
  "name": "production-app-updated",
  "is_active": true,
  "policies": [
    {
      "path": "/v1/clients/*",
      "capabilities": ["read", "write", "delete"]
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "name": "production-app-updated",
  "is_active": true,
  "policies": [
    {
      "path": "/v1/clients/*",
      "capabilities": ["read", "write", "delete"]
    }
  ],
  "created_at": "2026-02-12T20:13:45Z"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/v1/clients/018d7e95-1a23-7890-bcde-f1234567890a \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "updated-name",
    "is_active": true,
    "policies": [
      {"path": "*", "capabilities": ["read", "write"]}
    ]
  }'
```

#### Delete Client

Deletes a client. This operation is permanent and cannot be undone.

```bash
DELETE /v1/clients/:id
```

**Authentication:** Required  
**Authorization:** `DeleteCapability` for path `/v1/clients/:id`

**Response (204 No Content):**  
Empty body

**Example:**
```bash
curl -X DELETE http://localhost:8080/v1/clients/018d7e95-1a23-7890-bcde-f1234567890a \
  -H "Authorization: Bearer <admin-token>"
```

#### List Clients

Lists all API clients with pagination support. Results are ordered by ID in descending order (newest first).

```bash
GET /v1/clients
```

**Authentication:** Required  
**Authorization:** `ReadCapability` for path `/v1/clients`

**Query Parameters:**
- `offset` (optional) - Starting position for pagination (default: 0, must be >= 0)
- `limit` (optional) - Number of clients to return (default: 50, max: 100, must be >= 1)

**Response (200 OK):**
```json
{
  "clients": [
    {
      "id": "018d7e97-5e67-7890-bcde-f1234567890c",
      "name": "production-app",
      "is_active": true,
      "policies": [
        {
          "path": "/v1/clients/*",
          "capabilities": ["read", "write"]
        }
      ],
      "created_at": "2026-02-13T10:30:00Z"
    },
    {
      "id": "018d7e95-1a23-7890-bcde-f1234567890a",
      "name": "staging-app",
      "is_active": true,
      "policies": [
        {
          "path": "/v1/secrets/*",
          "capabilities": ["decrypt"]
        }
      ],
      "created_at": "2026-02-12T20:13:45Z"
    }
  ]
}
```

**Example - Default Pagination:**
```bash
# Get first 50 clients (default)
curl http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <token>"
```

**Example - Custom Pagination:**
```bash
# Get 20 clients starting from offset 10
curl http://localhost:8080/v1/clients?offset=10&limit=20 \
  -H "Authorization: Bearer <token>"

# Get next page
curl http://localhost:8080/v1/clients?offset=30&limit=20 \
  -H "Authorization: Bearer <token>"
```

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `ReadCapability` for `/v1/clients`
- `422 Unprocessable Entity` - Invalid query parameters (negative offset, limit out of range, non-numeric values)

**Important Notes:**
- 🔢 **Ordering**: Clients are ordered by ID DESC (newest first) using UUIDv7's time-based properties
- 🔒 **Security**: Client secrets are never returned in list responses
- 📄 **Pagination**: Simple offset/limit pagination without total count metadata
- ⚠️ **Limits**: Maximum limit is 100 clients per request. Use offset for pagination.

#### Policy Document Structure

Policy documents define what paths and capabilities a client has access to.

**Structure:**
```json
[
  {
    "path": "<path-pattern>",
    "capabilities": ["<capability1>", "<capability2>"]
  }
]
```

**Path Matching Patterns:**
- **Exact match:** `/v1/clients/018d7e95-1a23-7890-bcde-f1234567890a` (matches only this exact path)
- **Wildcard:** `*` (matches all paths)
- **Prefix:** `/v1/clients/*` (matches all paths starting with `/v1/clients/`)

**Available Capabilities:**
- `read` - View resources
- `write` - Create/update resources
- `delete` - Delete resources
- `encrypt` - Encrypt data (transit encryption)
- `decrypt` - Decrypt data (transit encryption)
- `rotate` - Rotate keys

**Example Policy - Read-Only Access to Client Management:**
```json
[
  {
    "path": "/v1/clients/*",
    "capabilities": ["read"]
  }
]
```

**Example Policy - Admin Access:**
```json
[
  {
    "path": "*",
    "capabilities": ["read", "write", "delete", "encrypt", "decrypt", "rotate"]
  }
]
```

**Note:** The wildcard `*` policy grants access to all current and future API endpoints.

**Example Policy - Multiple Paths with Different Capabilities:**
```json
[
  {
    "path": "/v1/clients/*",
    "capabilities": ["read", "write"]
  },
  {
    "path": "/v1/secrets/*",
    "capabilities": ["encrypt", "decrypt", "delete"]
  },
  {
    "path": "/v1/transit/*",
    "capabilities": ["encrypt", "decrypt"]
  }
]
```

**Example Policy - Application with Secrets Access:**
```json
[
  {
    "path": "/v1/clients/*",
    "capabilities": ["read"]
  },
  {
    "path": "/v1/secrets/app/production/*",
    "capabilities": ["encrypt", "decrypt"]
  }
]
```

**Example Policy - Read-Only Secrets Access:**
```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["decrypt"]
  }
]
```

**Example Policy - Full Secrets Management:**
```json
[
  {
    "path": "/v1/secrets/*",
    "capabilities": ["encrypt", "decrypt", "delete"]
  }
]
```

#### Transit Encryption Policy Examples

The following examples demonstrate policy configurations for transit encryption (encryption-as-a-service) use cases. Transit encryption allows applications to encrypt/decrypt data without storing it server-side.

**Example Policy - Encrypt-Only Access (Data Producer):**

Use case: Application that encrypts sensitive data before storing in external database, but never needs to decrypt.

```json
[
  {
    "path": "/v1/transit/keys/user-pii/encrypt",
    "capabilities": ["encrypt"]
  }
]
```

**Example Policy - Decrypt-Only Access (Data Consumer):**

Use case: Analytics service that reads encrypted data from database and decrypts for processing, but cannot encrypt new data.

```json
[
  {
    "path": "/v1/transit/keys/user-pii/decrypt",
    "capabilities": ["decrypt"]
  }
]
```

**Example Policy - Full Transit Key Management (Admin):**

Use case: Security team managing transit encryption keys with full control over creation, rotation, and deletion.

```json
[
  {
    "path": "/v1/transit/keys",
    "capabilities": ["write"]
  },
  {
    "path": "/v1/transit/keys/*/rotate",
    "capabilities": ["rotate"]
  },
  {
    "path": "/v1/transit/keys/*",
    "capabilities": ["read", "write", "delete", "encrypt", "decrypt"]
  }
]
```

**Example Policy - Multiple Transit Keys with Different Permissions:**

Use case: Application with separate encryption keys for different data types (PII, payment data, logs) with granular access control.

```json
[
  {
    "path": "/v1/transit/keys/user-pii/encrypt",
    "capabilities": ["encrypt"]
  },
  {
    "path": "/v1/transit/keys/user-pii/decrypt",
    "capabilities": ["decrypt"]
  },
  {
    "path": "/v1/transit/keys/payment-data/encrypt",
    "capabilities": ["encrypt"]
  },
  {
    "path": "/v1/transit/keys/audit-logs/encrypt",
    "capabilities": ["encrypt"]
  }
]
```

**Example Policy - Separation of Duties (SOC 2 Compliance):**

Use case: Enforce separation of duties where different clients handle encryption vs. decryption to meet compliance requirements.

**Client 1 (Encryption Service):**
```json
[
  {
    "path": "/v1/transit/keys/compliance-data/encrypt",
    "capabilities": ["encrypt"]
  }
]
```

**Client 2 (Decryption Service):**
```json
[
  {
    "path": "/v1/transit/keys/compliance-data/decrypt",
    "capabilities": ["decrypt"]
  }
]
```

**Client 3 (Key Administrator):**
```json
[
  {
    "path": "/v1/transit/keys",
    "capabilities": ["write"]
  },
  {
    "path": "/v1/transit/keys/*/rotate",
    "capabilities": ["rotate"]
  },
  {
    "path": "/v1/transit/keys/*",
    "capabilities": ["read", "delete"]
  }
]
```

**Note:** Transit Encryption API (`/v1/transit/*`) is fully implemented and production-ready. Audit Logs API (`/v1/audit-logs`) is planned but not yet implemented (see Planned Features section below).

---

## 📡 API Reference

### 🚨 Breaking Change: Base64 Encoding Required

**IMPORTANT:** All plaintext data in API requests and responses must now be base64-encoded:

**Secrets Management:**
- `POST /v1/secrets/*path` - Request `value` field must be base64-encoded
- `GET /v1/secrets/*path` - Response `value` field is base64-encoded

**Transit Encryption:**
- `POST /v1/transit/keys/:name/encrypt` - Request `plaintext` field must be base64-encoded
- `POST /v1/transit/keys/:name/decrypt` - Response `plaintext` field is base64-encoded

**Migration Example:**
```bash
# Old API (no longer works):
curl -d '{"value":"my-secret"}' ...

# New API (base64-encoded):
echo -n "my-secret" | base64  # Output: bXktc2VjcmV0
curl -d '{"value":"bXktc2VjcmV0"}' ...
```

This change improves binary data handling and ensures consistent encoding across all endpoints.

---

### 📦 Secrets Management

The Secrets Management API provides secure storage and retrieval of sensitive data using envelope encryption. All endpoints require authentication and appropriate capabilities.

#### 🔒 Security Warnings

**CRITICAL SECURITY CONSIDERATIONS:**
- ⚠️ **HTTPS Required**: ALWAYS use HTTPS in production. Secrets are returned as plaintext in GET responses
- ⚠️ **No Logging**: DO NOT log API response bodies containing secret values
- ⚠️ **Memory Handling**: Client applications MUST zero secret values from memory after use
- ⚠️ **Access Control**: Use fine-grained policies to restrict secret access to authorized clients only
- ⚠️ **Audit Trail**: All secret operations are logged for compliance and security monitoring

#### Authentication & Authorization

All secret endpoints require:
- **Authentication**: Valid Bearer token in `Authorization` header
- **Authorization**: Appropriate capability for the operation:
  - `POST` requires `EncryptCapability`
  - `GET` requires `DecryptCapability`
  - `DELETE` requires `DeleteCapability`

#### Create or Update Secret

Creates a new secret (version 1) or updates an existing secret (creates new version).

```bash
POST /v1/secrets/*path
```

**Authentication:** Required  
**Authorization:** `EncryptCapability` for path `/v1/secrets/*path`

**Request Body:**
```json
{
  "value": "bXktc2VjcmV0LXZhbHVl"
}
```

**Notes:**
- The secret path is part of the URL (e.g., `/v1/secrets/app/production/db-password`)
- The `value` field MUST be base64-encoded
- First creation sets version to 1
- Subsequent updates create new versions (2, 3, 4...)
- Each version is encrypted with its own Data Encryption Key (DEK)

**Response (201 Created):**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "path": "app/production/db-password",
  "version": 1,
  "created_at": "2026-02-12T20:13:45Z"
}
```

**Security Note:** The response excludes the plaintext value for security. The secret is encrypted and stored in the database.

**Example - Create New Secret:**
```bash
# First, base64-encode your secret
echo -n "super-secret-password-v1" | base64
# Output: c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYx

curl -X POST http://localhost:8080/v1/secrets/app/production/database-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "value": "c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYx"
  }'
```

**Example - Update Existing Secret (Creates Version 2):**
```bash
# Base64-encode the new value
echo -n "super-secret-password-v2" | base64
# Output: c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYy

curl -X POST http://localhost:8080/v1/secrets/app/production/database-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "value": "c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYy"
  }'
```

**Response:**
```json
{
  "id": "018d7e96-4d56-7890-bcde-f1234567890b",
  "path": "app/production/database-password",
  "version": 2,
  "created_at": "2026-02-12T21:30:15Z"
}
```

**Path Organization:**
- Use hierarchical paths for organization: `/app/env/service/credential`
- Examples:
  - `/app/production/database-password`
  - `/infrastructure/aws/access-key`
  - `/services/payment/api-token`

#### Get Secret

Retrieves and decrypts a secret by its path. Returns the latest version by default, or a specific version if the `version` query parameter is provided.

```bash
GET /v1/secrets/*path
GET /v1/secrets/*path?version=N
```

**Authentication:** Required  
**Authorization:** `DecryptCapability` for path `/v1/secrets/*path`

**Query Parameters:**
- `version` (optional) - Specific version number to retrieve (positive integer)
  - Omit to get the latest version
  - Must be a valid unsigned integer (1, 2, 3...)
  - Returns 422 if invalid format

**Response (200 OK):**
```json
{
  "id": "018d7e96-4d56-7890-bcde-f1234567890b",
  "path": "app/production/database-password",
  "value": "c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYy",
  "version": 2,
  "created_at": "2026-02-12T21:30:15Z"
}
```

**Security Note:** The `value` field contains the base64-encoded plaintext secret. Decode and handle with extreme care.

**Example - Get Latest Version:**
```bash
curl http://localhost:8080/v1/secrets/app/production/database-password \
  -H "Authorization: Bearer <token>"

# Response includes base64-encoded value:
# {"id":"...","path":"...","value":"c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYy","version":2,"created_at":"..."}

# Decode the value:
echo "c3VwZXItc2VjcmV0LXBhc3N3b3JkLXYy" | base64 -d
# Output: super-secret-password-v2
```

**Example - Get Specific Version:**
```bash
# Get version 1 (original secret)
curl http://localhost:8080/v1/secrets/app/production/database-password?version=1 \
  -H "Authorization: Bearer <token>"

# Get version 2 (first update)
curl http://localhost:8080/v1/secrets/app/production/database-password?version=2 \
  -H "Authorization: Bearer <token>"
```

**Use Cases for Version Retrieval:**
- 🔍 **Audit & Investigation**: Review historical secret values during security incidents
- 🔄 **Rollback**: Retrieve previous version to restore after problematic update
- 📊 **Compliance**: Access historical data for regulatory requirements
- 🐛 **Debugging**: Compare current vs. previous versions to identify issues

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `DecryptCapability` for the path
- `404 Not Found` - Secret not found at path, or version doesn't exist
- `422 Unprocessable Entity` - Invalid version parameter (not a positive integer)

**Important Notes:**
- 🔓 Secrets are automatically decrypted using the envelope encryption chain (Master Key → KEK → DEK → Secret)
- 📊 Each version has independent encryption with its own DEK for maximum security isolation
- 🗑️ Deleted secrets (soft delete) cannot be retrieved via the API
- ⚡ Version retrieval has the same performance as latest version (single database query)

#### Delete Secret

Performs a soft delete on the current version of a secret. The secret is marked as deleted but preserved in the database for audit purposes.

```bash
DELETE /v1/secrets/*path
```

**Authentication:** Required  
**Authorization:** `DeleteCapability` for path `/v1/secrets/*path`

**Response (204 No Content):**  
Empty body (HTTP status code only)

**Example:**
```bash
curl -X DELETE http://localhost:8080/v1/secrets/app/production/database-password \
  -H "Authorization: Bearer <token>"
```

**Behavior:**
- 🗑️ Sets the `deleted_at` timestamp on the **current version only**
- 📜 Preserves the encrypted secret data for audit trail and compliance
- 🔒 Previous versions remain unaffected and accessible (if not deleted)
- ⚠️ Deleted secrets cannot be retrieved via GET endpoint
- 💾 Data remains in database for forensic analysis and compliance requirements

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `DeleteCapability` for the path
- `404 Not Found` - Secret not found at path

**Important Notes:**
- ✅ This is a **soft delete** - data is NOT physically removed from the database
- ✅ Hard deletion (physical removal) is not currently supported via API
- ✅ Database administrators can recover soft-deleted secrets if needed
- ✅ Consider key rotation policies if secrets are suspected to be compromised

### 🚄 Transit Encryption API (Encryption-as-a-Service)

The Transit Encryption API provides encryption-as-a-service, allowing applications to encrypt and decrypt data without storing it server-side. This is ideal for encrypting sensitive data before storing it in external systems (databases, logs, object storage) while maintaining centralized key management.

#### 🔒 Security Warnings

**CRITICAL SECURITY CONSIDERATIONS:**
- ⚠️ **HTTPS Required**: ALWAYS use HTTPS in production. Plaintext is transmitted in API requests/responses
- ⚠️ **No Server Storage**: Transit encrypted data is NOT stored server-side. Client applications must store ciphertext
- ⚠️ **Memory Handling**: Client applications MUST zero plaintext from memory after encryption/decryption
- ⚠️ **Access Control**: Use fine-grained policies to restrict encryption/decryption access (`EncryptCapability`, `DecryptCapability`)
- ⚠️ **Key Rotation**: Regularly rotate transit keys. Old versions remain functional for backward compatibility
- ⚠️ **Audit Trail**: All transit operations are logged for compliance and security monitoring

#### Authentication & Authorization

All transit endpoints require:
- **Authentication**: Valid Bearer token in `Authorization` header
- **Authorization**: Appropriate capability for the operation:
  - `POST /v1/transit/keys` requires `WriteCapability`
  - `POST /v1/transit/keys/:name/rotate` requires `RotateCapability`
  - `DELETE /v1/transit/keys/:id` requires `DeleteCapability`
  - `POST /v1/transit/keys/:name/encrypt` requires `EncryptCapability`
  - `POST /v1/transit/keys/:name/decrypt` requires `DecryptCapability`

#### 🔑 Ciphertext Format

Transit encryption uses a versioned ciphertext format that enables transparent key rotation:

**Format:** `"version:base64-ciphertext"`

**Example:** `"1:ZW5jcnlwdGVkLWRhdGEtd2l0aC1ub25jZS1hbmQtYXV0aA=="`

**Components:**
- `version` - Transit key version number (integer: 1, 2, 3...)
- `:` - Delimiter
- `base64-ciphertext` - Base64-encoded AEAD output (nonce + ciphertext + authentication tag)

**Key Rotation Behavior:**
- Encryption always uses the **latest** transit key version
- Decryption automatically uses the version specified in the ciphertext prefix
- Old ciphertext remains decryptable after key rotation (backward compatibility)
- Applications don't need to modify code or re-encrypt data during key rotation

#### 📦 Binary Data Handling

**All plaintext data MUST be base64-encoded before sending to the API:**
- Request `plaintext` fields require base64 encoding
- Response `plaintext` fields are base64 encoded
- Ciphertext is already in versioned format: `"version:base64"`

**Examples:**

**Text Data (UTF-8):**
```bash
# Plain text "hello world"
echo -n "hello world" | base64  # Output: aGVsbG8gd29ybGQ=

# Send to API (plaintext must be base64-encoded)
curl -X POST http://localhost:8080/v1/transit/keys/mykey/encrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"aGVsbG8gd29ybGQ="}'

# Response: {"ciphertext":"1:ZXhhbXBsZS1jaXBoZXJ0ZXh0","version":1}

# Decrypt (response plaintext is base64-encoded)
curl -X POST http://localhost:8080/v1/transit/keys/mykey/decrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"ciphertext":"1:ZXhhbXBsZS1jaXBoZXJ0ZXh0"}'

# Response: {"plaintext":"aGVsbG8gd29ybGQ=","version":1}
# Decode: echo "aGVsbG8gd29ybGQ=" | base64 -d  # Output: hello world
```

**Binary Data:**
```bash
# Binary file (e.g., image)
base64 image.png > image_b64.txt

# Send to API
curl -X POST http://localhost:8080/v1/transit/keys/mykey/encrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"plaintext\":\"$(cat image_b64.txt)\"}"
```

#### Create Transit Key

Creates a new named transit encryption key for encryption-as-a-service operations.

```bash
POST /v1/transit/keys
```

**Authentication:** Required  
**Authorization:** `WriteCapability` for path `/v1/transit/keys`

**Request Body:**
```json
{
  "name": "payment-encryption",
  "algorithm": "aes-gcm"
}
```

**Fields:**
- `name` (required) - Unique name for the transit key (1-255 characters, alphanumeric and hyphens)
- `algorithm` (required) - Encryption algorithm: `"aes-gcm"` or `"chacha20-poly1305"`

**Response (201 Created):**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "name": "payment-encryption",
  "algorithm": "aes-gcm",
  "version": 1,
  "created_at": "2026-02-13T20:13:45Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/transit/keys \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-pii-encryption",
    "algorithm": "aes-gcm"
  }'
```

**Use Cases:**
- 💳 **Payment Data**: Encrypt credit card numbers before storing in database
- 👤 **PII Protection**: Encrypt names, emails, SSNs for GDPR/CCPA compliance
- 📊 **Audit Logs**: Encrypt sensitive log entries while maintaining searchability on metadata
- 🗄️ **Database Fields**: Application-level encryption for specific columns (defense in depth)
- 🌐 **API Responses**: Encrypt sensitive fields in API responses for client-side storage

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `WriteCapability` for the path
- `409 Conflict` - Transit key with this name already exists
- `422 Unprocessable Entity` - Invalid request format or unsupported algorithm

**Important Notes:**
- ✅ Transit key names must be unique across the system
- ✅ Algorithm choice affects performance and compatibility (AES-GCM for hardware acceleration, ChaCha20 for software-only environments)
- ✅ Transit keys are versioned - rotation creates new versions while preserving old ones
- ✅ Each transit key version has its own Data Encryption Key (DEK) for cryptographic isolation

#### Rotate Transit Key

Rotates a transit key by creating a new version. Old versions remain active for decryption, but new encryptions use the latest version.

```bash
POST /v1/transit/keys/:name/rotate
```

**Authentication:** Required  
**Authorization:** `RotateCapability` for path `/v1/transit/keys/:name/rotate`

**Request Body:** Empty (no request body required)

**Response (200 OK):**
```json
{
  "id": "018d7e96-4d56-7890-bcde-f1234567890b",
  "name": "payment-encryption",
  "algorithm": "aes-gcm",
  "version": 2,
  "created_at": "2026-02-14T10:30:00Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-encryption/rotate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json"
```

**When to Rotate:**
- ⏰ **Regularly**: Every 90 days as a security best practice
- 🚨 **Security Incident**: When suspecting key compromise
- 📋 **Compliance**: Per regulatory requirements (PCI-DSS, HIPAA, etc.)
- 📊 **Volume Threshold**: After encrypting a certain volume of data (e.g., 1 billion operations)

**Rotation Behavior:**
- ✅ New version is created and becomes the active version for encryption
- ✅ Old versions remain available for decryption (backward compatibility)
- ✅ Ciphertext encrypted with old versions remains valid indefinitely
- ✅ No re-encryption of existing data required
- ✅ Atomic operation - either fully succeeds or fully fails

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `RotateCapability` for the path
- `404 Not Found` - Transit key not found

**Important Notes:**
- 🔄 **Zero Downtime**: Rotation does not interrupt encryption/decryption operations
- 📦 **No Re-encryption Needed**: Existing ciphertext remains valid and decryptable
- 🗑️ **Soft Delete Protection**: Cannot rotate deleted transit keys
- ⚡ **Performance**: No performance impact on existing ciphertext decryption

#### Delete Transit Key

Performs a soft delete on a transit key. The key is marked as deleted but preserved in the database for decrypting existing ciphertext.

```bash
DELETE /v1/transit/keys/:id
```

**Authentication:** Required  
**Authorization:** `DeleteCapability` for path `/v1/transit/keys/:id`

**URL Parameters:**
- `id` (required) - UUID of the transit key (not the name)

**Response (204 No Content):**  
Empty body (HTTP status code only)

**Example:**
```bash
curl -X DELETE http://localhost:8080/v1/transit/keys/018d7e95-1a23-7890-bcde-f1234567890a \
  -H "Authorization: Bearer <token>"
```

**Behavior:**
- 🗑️ Marks all versions of the transit key as deleted
- 🔒 Prevents new encryption operations with this key
- ✅ Allows decryption of existing ciphertext (for data recovery)
- 📜 Preserves key material for audit trail and compliance
- 💾 Data remains in database for forensic analysis

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `DeleteCapability` for the path
- `404 Not Found` - Transit key not found

**Important Notes:**
- ✅ This is a **soft delete** - key material is NOT physically removed
- ⚠️ **Decryption Still Works**: Existing ciphertext remains decryptable after deletion
- ⚠️ **Encryption Blocked**: New encryption operations fail with 404 Not Found
- ✅ Hard deletion (physical removal) is not currently supported via API

#### Encrypt Data

Encrypts plaintext data using a named transit key. The encrypted data is returned to the client and NOT stored server-side.

```bash
POST /v1/transit/keys/:name/encrypt
```

**Authentication:** Required  
**Authorization:** `EncryptCapability` for path `/v1/transit/keys/:name/encrypt`

**URL Parameters:**
- `name` (required) - Transit key name (not UUID)

**Request Body:**
```json
{
  "plaintext": "aGVsbG8gd29ybGQ="
}
```

**Fields:**
- `plaintext` (required) - Base64-encoded data to encrypt

**Response (200 OK):**
```json
{
  "ciphertext": "1:ZW5jcnlwdGVkLWRhdGEtd2l0aC1ub25jZS1hbmQtYXV0aA==",
  "version": 1
}
```

**Fields:**
- `ciphertext` - Versioned ciphertext string (format: `"version:base64-ciphertext"`)
- `version` - Transit key version used for encryption (for informational purposes)

**Example - Text Data:**
```bash
# First, base64-encode your plaintext
echo -n "sensitive-data" | base64
# Output: c2Vuc2l0aXZlLWRhdGE=

# Encrypt text "sensitive-data"
curl -X POST http://localhost:8080/v1/transit/keys/user-pii-encryption/encrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "plaintext": "c2Vuc2l0aXZlLWRhdGE="
  }'

# Response: {"ciphertext":"1:ZW5jcnlwdGVkLWRhdGEtd2l0aC1ub25jZS1hbmQtYXV0aA==","version":1}
```

**Example - Binary Data:**
```bash
# Encrypt binary file
FILE_B64=$(base64 -i secret.pdf)
curl -X POST http://localhost:8080/v1/transit/keys/document-encryption/encrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d "{\"plaintext\":\"$FILE_B64\"}"
```

**Use Cases:**
- 💳 **Tokenization**: Encrypt credit card numbers before database storage
- 👤 **GDPR Compliance**: Encrypt PII (names, emails, addresses) in user records
- 📊 **Sensitive Logs**: Encrypt log entries containing credentials or API keys
- 🗄️ **Column-Level Encryption**: Protect specific database columns (SSN, medical records)
- 🌐 **Client-Side Storage**: Encrypt data before sending to client browser (localStorage, IndexedDB)

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `EncryptCapability` for the path
- `404 Not Found` - Transit key not found or deleted
- `422 Unprocessable Entity` - Invalid request format or missing plaintext

**Important Notes:**
- ✅ **Always Uses Latest Version**: Encryption uses the current active version of the transit key
- 🔐 **AEAD Protection**: Authenticated encryption prevents tampering
- 📦 **No Server Storage**: Plaintext and ciphertext are never persisted server-side
- 🔄 **Idempotent**: Same plaintext encrypts to different ciphertext each time (random nonce)
- 🧹 **Memory Safety**: Plaintext is zeroed from server memory after encryption

#### Decrypt Data

Decrypts ciphertext using a named transit key. The version is automatically extracted from the ciphertext prefix.

```bash
POST /v1/transit/keys/:name/decrypt
```

**Authentication:** Required  
**Authorization:** `DecryptCapability` for path `/v1/transit/keys/:name/decrypt`

**URL Parameters:**
- `name` (required) - Transit key name (must match the key used for encryption)

**Request Body:**
```json
{
  "ciphertext": "1:ZW5jcnlwdGVkLWRhdGEtd2l0aC1ub25jZS1hbmQtYXV0aA=="
}
```

**Fields:**
- `ciphertext` (required) - Versioned ciphertext string from encrypt operation (format: `"version:base64-ciphertext"`)

**Response (200 OK):**
```json
{
  "plaintext": "aGVsbG8gd29ybGQ=",
  "version": 1
}
```

**Fields:**
- `plaintext` - Base64-encoded decrypted data
- `version` - Transit key version used for decryption (extracted from ciphertext prefix)

**Example - Text Data:**
```bash
# Decrypt ciphertext
curl -X POST http://localhost:8080/v1/transit/keys/user-pii-encryption/decrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "1:ZW5jcnlwdGVkLWRhdGEtd2l0aC1ub25jZS1hbmQtYXV0aA=="
  }'

# Response: {"plaintext":"c2Vuc2l0aXZlLWRhdGE=","version":1}

# Decode the base64 plaintext
echo "c2Vuc2l0aXZlLWRhdGE=" | base64 -d  # Output: sensitive-data
```

**Example - Binary Data:**
```bash
# Decrypt and save to file
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/transit/keys/document-encryption/decrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"ciphertext":"2:..."}')

PLAINTEXT=$(echo "$RESPONSE" | jq -r '.plaintext')
echo "$PLAINTEXT" | base64 --decode > decrypted_secret.pdf
```

**Version Handling:**
- 🔄 **Automatic Version Selection**: Version is parsed from ciphertext prefix (e.g., `"1:"`, `"2:"`, `"3:"`)
- ✅ **Backward Compatibility**: Old ciphertext remains decryptable after key rotation
- 🔑 **Multiple Versions**: System maintains all historical transit key versions for decryption
- 📦 **No Client Changes**: Applications don't need code changes during key rotation

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `DecryptCapability` for the path
- `404 Not Found` - Transit key not found
- `422 Unprocessable Entity` - Invalid ciphertext format, corrupted data, or authentication failure

**Important Notes:**
- ✅ **Version Agnostic**: Works with any version of the transit key (supports key rotation)
- 🔐 **AEAD Verification**: Decryption fails if ciphertext has been tampered with
- 📦 **No Server Storage**: Plaintext and ciphertext are never persisted server-side
- 🗑️ **Deleted Keys**: Decryption still works for soft-deleted transit keys
- 🧹 **Memory Safety**: Plaintext is zeroed from server memory after response is sent

#### Client Library Examples

The following examples demonstrate how to use the Transit Encryption API from different programming languages.

##### Python Example

```python
import requests
import base64
import json

class TransitClient:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json"
        }
    
    def create_key(self, name, algorithm="aes-gcm"):
        """Create a new transit encryption key."""
        url = f"{self.base_url}/v1/transit/keys"
        data = {"name": name, "algorithm": algorithm}
        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        return response.json()
    
    def encrypt(self, key_name, plaintext):
        """Encrypt plaintext data.
        
        Args:
            key_name: Name of the transit key
            plaintext: bytes or str to encrypt
        
        Returns:
            Versioned ciphertext string (e.g., "1:base64...")
        """
        url = f"{self.base_url}/v1/transit/keys/{key_name}/encrypt"
        
        # Convert to bytes if string
        if isinstance(plaintext, str):
            plaintext = plaintext.encode('utf-8')
        
        # Base64 encode for JSON
        plaintext_b64 = base64.b64encode(plaintext).decode('utf-8')
        
        data = {"plaintext": plaintext_b64}
        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        
        return response.json()["ciphertext"]
    
    def decrypt(self, key_name, ciphertext):
        """Decrypt ciphertext data.
        
        Args:
            key_name: Name of the transit key
            ciphertext: Versioned ciphertext string from encrypt()
        
        Returns:
            Decrypted plaintext as bytes
        """
        url = f"{self.base_url}/v1/transit/keys/{key_name}/decrypt"
        
        data = {"ciphertext": ciphertext}
        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        
        # Decode base64 plaintext
        plaintext_b64 = response.json()["plaintext"]
        return base64.b64decode(plaintext_b64)

# Usage Example
client = TransitClient("http://localhost:8080", "your-token-here")

# Create transit key
key_info = client.create_key("python-app-encryption", "aes-gcm")
print(f"Created key: {key_info['name']} (version {key_info['version']})")

# Encrypt text data
plaintext = "sensitive user data"
ciphertext = client.encrypt("python-app-encryption", plaintext)
print(f"Ciphertext: {ciphertext}")

# Decrypt data
decrypted = client.decrypt("python-app-encryption", ciphertext)
print(f"Decrypted: {decrypted.decode('utf-8')}")

# Encrypt binary data (e.g., image file)
with open("photo.jpg", "rb") as f:
    image_data = f.read()

encrypted_image = client.encrypt("python-app-encryption", image_data)

# Store encrypted_image in database
# ...

# Later: decrypt and save
decrypted_image = client.decrypt("python-app-encryption", encrypted_image)
with open("photo_decrypted.jpg", "wb") as f:
    f.write(decrypted_image)
```

##### JavaScript/Node.js Example

```javascript
const axios = require('axios');

class TransitClient {
  constructor(baseURL, token) {
    this.client = axios.create({
      baseURL: baseURL,
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    });
  }

  async createKey(name, algorithm = 'aes-gcm') {
    /**
     * Create a new transit encryption key.
     */
    const response = await this.client.post('/v1/transit/keys', {
      name: name,
      algorithm: algorithm
    });
    return response.data;
  }

  async encrypt(keyName, plaintext) {
    /**
     * Encrypt plaintext data.
     * 
     * @param {string} keyName - Name of the transit key
     * @param {string|Buffer} plaintext - Data to encrypt
     * @returns {string} Versioned ciphertext string (e.g., "1:base64...")
     */
    // Convert to Buffer if string
    const buffer = Buffer.isBuffer(plaintext) 
      ? plaintext 
      : Buffer.from(plaintext, 'utf-8');
    
    // Base64 encode for JSON
    const plaintextB64 = buffer.toString('base64');
    
    const response = await this.client.post(
      `/v1/transit/keys/${keyName}/encrypt`,
      { plaintext: plaintextB64 }
    );
    
    return response.data.ciphertext;
  }

  async decrypt(keyName, ciphertext) {
    /**
     * Decrypt ciphertext data.
     * 
     * @param {string} keyName - Name of the transit key
     * @param {string} ciphertext - Versioned ciphertext from encrypt()
     * @returns {Buffer} Decrypted plaintext as Buffer
     */
    const response = await this.client.post(
      `/v1/transit/keys/${keyName}/decrypt`,
      { ciphertext: ciphertext }
    );
    
    // Decode base64 plaintext
    const plaintextB64 = response.data.plaintext;
    return Buffer.from(plaintextB64, 'base64');
  }
}

// Usage Example
(async () => {
  const client = new TransitClient('http://localhost:8080', 'your-token-here');

  // Create transit key
  const keyInfo = await client.createKey('nodejs-app-encryption', 'aes-gcm');
  console.log(`Created key: ${keyInfo.name} (version ${keyInfo.version})`);

  // Encrypt text data
  const plaintext = 'sensitive user data';
  const ciphertext = await client.encrypt('nodejs-app-encryption', plaintext);
  console.log(`Ciphertext: ${ciphertext}`);

  // Decrypt data
  const decrypted = await client.decrypt('nodejs-app-encryption', ciphertext);
  console.log(`Decrypted: ${decrypted.toString('utf-8')}`);

  // Encrypt binary data (e.g., PDF file)
  const fs = require('fs').promises;
  const pdfData = await fs.readFile('document.pdf');
  
  const encryptedPDF = await client.encrypt('nodejs-app-encryption', pdfData);
  
  // Store encryptedPDF in database
  // await database.save({ file: encryptedPDF });
  
  // Later: decrypt and save
  const decryptedPDF = await client.decrypt('nodejs-app-encryption', encryptedPDF);
  await fs.writeFile('document_decrypted.pdf', decryptedPDF);
})();
```

##### Go Example

```go
package main

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

type TransitClient struct {
    BaseURL string
    Token   string
    Client  *http.Client
}

func NewTransitClient(baseURL, token string) *TransitClient {
    return &TransitClient{
        BaseURL: baseURL,
        Token:   token,
        Client:  &http.Client{},
    }
}

func (c *TransitClient) CreateKey(name, algorithm string) (map[string]interface{}, error) {
    // Create a new transit encryption key
    reqBody := map[string]string{
        "name":      name,
        "algorithm": algorithm,
    }
    
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", c.BaseURL+"/v1/transit/keys", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.Token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.Client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result, nil
}

func (c *TransitClient) Encrypt(keyName string, plaintext []byte) (string, error) {
    // Encrypt plaintext data
    // Base64 encode for JSON (Go's json.Marshal does this automatically for []byte)
    plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)
    
    reqBody := map[string]string{
        "plaintext": plaintextB64,
    }
    
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }
    
    url := fmt.Sprintf("%s/v1/transit/keys/%s/encrypt", c.BaseURL, keyName)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.Token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.Client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    return result["ciphertext"].(string), nil
}

func (c *TransitClient) Decrypt(keyName, ciphertext string) ([]byte, error) {
    // Decrypt ciphertext data
    reqBody := map[string]string{
        "ciphertext": ciphertext,
    }
    
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    url := fmt.Sprintf("%s/v1/transit/keys/%s/decrypt", c.BaseURL, keyName)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.Token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.Client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    // Decode base64 plaintext
    plaintextB64 := result["plaintext"].(string)
    return base64.StdEncoding.DecodeString(plaintextB64)
}

func main() {
    client := NewTransitClient("http://localhost:8080", "your-token-here")
    
    // Create transit key
    keyInfo, err := client.CreateKey("go-app-encryption", "aes-gcm")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created key: %s (version %.0f)\n", keyInfo["name"], keyInfo["version"])
    
    // Encrypt text data
    plaintext := []byte("sensitive user data")
    ciphertext, err := client.Encrypt("go-app-encryption", plaintext)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Ciphertext: %s\n", ciphertext)
    
    // Decrypt data
    decrypted, err := client.Decrypt("go-app-encryption", ciphertext)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Decrypted: %s\n", string(decrypted))
    
    // Encrypt binary data (e.g., image file)
    imageData, err := os.ReadFile("photo.jpg")
    if err != nil {
        panic(err)
    }
    
    encryptedImage, err := client.Encrypt("go-app-encryption", imageData)
    if err != nil {
        panic(err)
    }
    
    // Store encryptedImage in database
    // db.Save(encryptedImage)
    
    // Later: decrypt and save
    decryptedImage, err := client.Decrypt("go-app-encryption", encryptedImage)
    if err != nil {
        panic(err)
    }
    
    if err := os.WriteFile("photo_decrypted.jpg", decryptedImage, 0644); err != nil {
        panic(err)
    }
}
```

### 📜 Audit Logs API

The Audit Logs API provides immutable, queryable records of all operations performed in the system. Every authenticated API request automatically generates an audit log entry for compliance, security monitoring, and forensic investigation.

#### 🔒 Security & Compliance Features

**Audit Log Characteristics:**
- 📝 **Automatic Logging**: All authenticated operations are automatically logged via authorization middleware
- 🔒 **Immutable Records**: Audit logs cannot be modified or deleted after creation (append-only)
- 🔍 **Comprehensive Tracking**: Captures request ID, client ID, capability, path, and custom metadata
- 📊 **Compliance Ready**: Meets requirements for SOC 2, GDPR, HIPAA, and PCI-DSS auditing
- 🕵️ **Forensic Investigation**: Complete audit trail for security incident analysis
- ⏱️ **Time-Ordered**: Logs are ordered by ID descending (newest first) using UUIDv7's time-based properties
- 🚫 **Non-Blocking**: Audit log failures don't block API operations (logged as errors)

#### Authentication & Authorization

All audit log endpoints require:
- **Authentication**: Valid Bearer token in `Authorization` header
- **Authorization**: `ReadCapability` for path `/v1/audit-logs`

#### What Gets Logged

Audit logs automatically capture every authorization attempt (both successful and denied):
- ✅ **Client Operations**: Create, read, update, delete API clients
- ✅ **Secret Operations**: Encrypt (create/update), decrypt (read), delete secrets
- ✅ **Transit Operations**: Create transit keys, rotate keys, encrypt, decrypt data
- ✅ **Token Operations**: Token issuance (authentication events)
- ✅ **Audit Log Access**: Reading audit logs themselves
- ✅ **Request Context**: Request ID, client ID, capability used, resource path
- ✅ **Authorization Outcome**: Whether access was allowed or denied
- ✅ **Client Information**: IP address and user agent from HTTP request

**Metadata Structure:**

Each audit log entry includes metadata with the following fields:
```json
{
  "allowed": true,           // Authorization outcome (true = success, false = denied)
  "ip": "192.168.1.100",     // Client IP address
  "user_agent": "curl/7.68.0" // HTTP User-Agent header
}
```

#### Use Cases

- 🔍 **Security Monitoring**: Detect anomalous access patterns and unauthorized access attempts
- 📋 **Compliance Auditing**: Generate reports for regulatory compliance (SOC 2, GDPR, HIPAA, PCI-DSS)
- 🐛 **Debugging**: Trace request flows using request IDs across distributed systems
- 🚨 **Incident Response**: Investigate security breaches with complete operation history
- 📊 **Analytics**: Analyze API usage patterns and client behavior
- 🔐 **Access Reviews**: Verify clients are accessing only authorized resources
- ⚠️ **Failed Access Detection**: Identify clients attempting unauthorized operations

#### List Audit Logs

Retrieves audit log entries with pagination support. Results are ordered by ID in descending order (newest first).

```bash
GET /v1/audit-logs
```

**Authentication:** Required  
**Authorization:** `ReadCapability` for path `/v1/audit-logs`

**Query Parameters:**
- `offset` (optional) - Starting position for pagination (default: 0, must be >= 0)
- `limit` (optional) - Number of logs to return (default: 50, min: 1, max: 100)

**Response (200 OK):**
```json
{
  "audit_logs": [
    {
      "id": "018d7e97-5e67-7890-bcde-f1234567890c",
      "request_id": "018d7e97-5e66-7890-bcde-f1234567890b",
      "client_id": "018d7e95-1a23-7890-bcde-f1234567890a",
      "capability": "decrypt",
      "path": "/v1/secrets/app/production/database-password",
      "metadata": {
        "allowed": true,
        "ip": "192.168.1.100",
        "user_agent": "python-requests/2.28.1"
      },
      "created_at": "2026-02-13T20:15:30Z"
    },
    {
      "id": "018d7e97-4d56-7890-bcde-f1234567890b",
      "request_id": "018d7e97-4d55-7890-bcde-f1234567890a",
      "client_id": "018d7e95-1a23-7890-bcde-f1234567890a",
      "capability": "encrypt",
      "path": "/v1/secrets/app/production/api-key",
      "metadata": {
        "allowed": true,
        "ip": "192.168.1.100",
        "user_agent": "curl/7.68.0"
      },
      "created_at": "2026-02-13T20:13:45Z"
    },
    {
      "id": "018d7e97-3c45-7890-bcde-f1234567890a",
      "request_id": "018d7e97-3c44-7890-bcde-f1234567890b",
      "client_id": "018d7e96-2b34-7890-bcde-f1234567890b",
      "capability": "write",
      "path": "/v1/clients",
      "metadata": {
        "allowed": true,
        "ip": "192.168.1.50",
        "user_agent": "Go-http-client/1.1"
      },
      "created_at": "2026-02-13T19:45:20Z"
    },
    {
      "id": "018d7e97-2b34-7890-bcde-f1234567890a",
      "request_id": "018d7e97-2b33-7890-bcde-f1234567890b",
      "client_id": "018d7e96-1a23-7890-bcde-f1234567890c",
      "capability": "read",
      "path": "/v1/clients/018d7e95-1a23-7890-bcde-f1234567890a",
      "metadata": {
        "allowed": false,
        "ip": "192.168.1.200",
        "user_agent": "curl/7.68.0"
      },
      "created_at": "2026-02-13T19:30:15Z"
    }
  ]
}
```

**Field Descriptions:**
- `id` - Unique audit log entry ID (UUIDv7, time-ordered)
- `request_id` - Request ID for distributed tracing (matches `X-Request-Id` response header)
- `client_id` - ID of the authenticated client that performed the operation
- `capability` - Capability used for authorization (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`)
- `path` - Resource path that was accessed (e.g., `/v1/secrets/app/prod/db-password`)
- `metadata` - Additional context:
  - `allowed` - Authorization outcome (true = granted, false = denied)
  - `ip` - Client IP address (extracted via `c.ClientIP()`)
  - `user_agent` - HTTP User-Agent header
- `created_at` - Timestamp when the operation occurred (UTC, ISO 8601 format)

**Example - Default Pagination:**
```bash
# Get first 50 audit logs (default)
curl http://localhost:8080/v1/audit-logs \
  -H "Authorization: Bearer <token>"
```

**Example - Custom Pagination:**
```bash
# Get 20 audit logs starting from offset 10
curl http://localhost:8080/v1/audit-logs?offset=10&limit=20 \
  -H "Authorization: Bearer <token>"

# Get next page
curl http://localhost:8080/v1/audit-logs?offset=30&limit=20 \
  -H "Authorization: Bearer <token>"

# Get maximum allowed logs per request (100)
curl http://localhost:8080/v1/audit-logs?offset=0&limit=100 \
  -H "Authorization: Bearer <token>"
```

**Example - Filter Failed Authorization Attempts:**
```bash
# Get recent logs and filter for denied access (client-side filtering)
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq '.audit_logs[] | select(.metadata.allowed == false)'
```

**Example - Track Specific Client Activity:**
```bash
# Get all operations by a specific client (client-side filtering)
CLIENT_ID="018d7e95-1a23-7890-bcde-f1234567890a"
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq --arg cid "$CLIENT_ID" '.audit_logs[] | select(.client_id == $cid)'
```

**Error Responses:**
- `401 Unauthorized` - Invalid or missing authentication token
- `403 Forbidden` - Client lacks `ReadCapability` for `/v1/audit-logs`
- `422 Unprocessable Entity` - Invalid query parameters:
  - Negative offset value (e.g., `offset=-1`)
  - Limit less than 1 or greater than 100 (e.g., `limit=0` or `limit=200`)
  - Non-numeric offset or limit values (e.g., `offset=abc`)

**Important Notes:**
- 🔢 **Ordering**: Audit logs are ordered by ID DESC (newest first) using UUIDv7's time-based properties
- 📄 **Pagination**: Simple offset/limit pagination without total count metadata (for performance)
- 🔒 **Immutability**: Audit logs cannot be modified or deleted via API (append-only design)
- 📦 **Empty Results**: Returns `{"audit_logs": []}` when no logs exist (not null)
- ⚠️ **Limits**: Maximum limit is 100 logs per request to prevent performance degradation
- 🔍 **Metadata Flexibility**: Metadata structure is consistent but extensible for future enhancements
- 🌐 **Request Tracing**: Use `request_id` to correlate audit logs with application logs and distributed traces
- 📊 **Authorization Tracking**: Both successful (`allowed: true`) and failed (`allowed: false`) attempts are logged
- 🚫 **Non-Blocking**: Audit log creation failures are logged as errors but don't block API responses

**Compliance Considerations:**
- ✅ **Data Retention**: Audit logs are retained indefinitely for compliance (no automatic deletion)
- ✅ **Access Control**: Only clients with `ReadCapability` for `/v1/audit-logs` can view logs
- ✅ **Complete Audit Trail**: All authorization attempts are logged, including denied operations
- ✅ **Time Accuracy**: All timestamps use UTC for consistent cross-timezone auditing
- ✅ **Tamper Resistance**: Append-only design prevents modification of historical logs
- ✅ **Request Correlation**: Request IDs enable end-to-end tracing across systems

**Monitoring & Alerting Use Cases:**

**Detect Failed Authorization Attempts:**
```bash
# Monitor for repeated failed access attempts (potential breach)
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq '.audit_logs[] | select(.metadata.allowed == false) | {client_id, path, ip: .metadata.ip, time: .created_at}'
```

**Track High-Privilege Operations:**
```bash
# Identify all delete and rotate operations (sensitive capabilities)
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq '.audit_logs[] | select(.capability == "delete" or .capability == "rotate")'
```

**Analyze Access Patterns:**
```bash
# Get summary of operations by capability
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq '[.audit_logs[] | .capability] | group_by(.) | map({capability: .[0], count: length})'
```

**Trace Specific Request:**
```bash
# Find all operations related to a specific request ID
REQUEST_ID="018d7e97-5e66-7890-bcde-f1234567890b"
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq --arg rid "$REQUEST_ID" '.audit_logs[] | select(.request_id == $rid)'
```

**Security Alert: Detect IP Changes for Client:**
```bash
# Identify if a client is accessing from multiple IPs (potential credential theft)
CLIENT_ID="018d7e95-1a23-7890-bcde-f1234567890a"
curl -s http://localhost:8080/v1/audit-logs?limit=100 \
  -H "Authorization: Bearer <token>" | \
  jq --arg cid "$CLIENT_ID" '[.audit_logs[] | select(.client_id == $cid) | .metadata.ip] | unique'
```

## 🚧 Planned Features

The following API endpoints are planned but not yet implemented.

## 🛠️ Development

### 🎯 CLI Commands

The application provides several CLI commands for managing the system:

```bash
# View all available commands
./bin/app --help

# View help for a specific command
./bin/app create-master-key --help
```

**Available Commands:**

| Command | Description | Usage |
|---------|-------------|-------|
| `server` | Start the HTTP server | `./bin/app server` |
| `migrate` | Run database migrations | `./bin/app migrate` |
| `create-master-key` | Generate a new Master Key | `./bin/app create-master-key [--id <key-id>]` |
| `create-kek` | Create initial Key Encryption Key | `./bin/app create-kek [--algorithm <alg>]` |
| `rotate-kek` | Rotate the Key Encryption Key | `./bin/app rotate-kek [--algorithm <alg>]` |
| `create-client` | Create a new authentication client | `./bin/app create-client --name <name> [--policies <json>]` |
| `update-client` | Update an existing client | `./bin/app update-client --id <uuid> --name <name> [--policies <json>]` |

**Command Details:**

#### `create-master-key` - Generate Master Key
```bash
# Generate with auto-generated ID (master-key-YYYY-MM-DD)
./bin/app create-master-key

# Generate with custom ID
./bin/app create-master-key --id prod-master-key-2025
./bin/app create-master-key -i prod-master-key-2025  # Short flag
```
- **Purpose**: Generate cryptographically secure 32-byte master key
- **Output**: Environment variables ready to copy to `.env` file
- **Security**: Key is zeroed from memory after generation
- **When to run**: Before initial setup or during master key rotation

#### `migrate` - Database Migrations
```bash
./bin/app migrate
```
- **Purpose**: Run database schema migrations
- **Requirements**: Database connection configured in environment
- **When to run**: After database creation, before first use
- **Supported databases**: PostgreSQL, MySQL

#### `create-kek` - Create Initial KEK
```bash
# Default algorithm (AES-GCM)
./bin/app create-kek

# Specify algorithm
./bin/app create-kek --algorithm chacha20-poly1305
./bin/app create-kek --alg aes-gcm  # Short flag
```
- **Purpose**: Create the first Key Encryption Key
- **Requirements**: Database migrated, master keys configured
- **When to run**: Once during initial setup
- **Algorithms**: `aes-gcm`, `chacha20-poly1305`

#### `rotate-kek` - Rotate KEK
```bash
# Default algorithm (AES-GCM)
./bin/app rotate-kek

# Specify algorithm
./bin/app rotate-kek --algorithm chacha20-poly1305
./bin/app rotate-kek --alg aes-gcm  # Short flag
```
- **Purpose**: Create new KEK version and mark old as inactive
- **Requirements**: Active KEK already exists
- **When to run**: Every 90 days or after security incident
- **Effect**: New secrets use new KEK, old secrets remain readable

#### `server` - Start HTTP Server
```bash
./bin/app server
```
- **Purpose**: Start the HTTP API server
- **Requirements**: Database migrated, KEK created
- **Port**: Configured via `SERVER_PORT` environment variable (default: 8080)
- **Endpoints**: Health check (`/health`, `/ready`), token issuance (`/v1/token`), client management (`/v1/clients`), secrets management (`/v1/secrets`), transit encryption (`/v1/transit`)

#### `create-client` - Create Authentication Client
```bash
# Interactive mode (prompts for policies)
./bin/app create-client --name "my-app"

# Non-interactive mode with JSON policies
./bin/app create-client --name "my-app" \
  --policies '[{"path":"*","capabilities":["read","write"]}]'

# Create inactive client
./bin/app create-client --name "my-app" --active=false \
  --policies '[{"path":"/v1/secrets/*","capabilities":["read"]}]'

# JSON output format (for automation)
./bin/app create-client --name "my-app" --format json \
  --policies '[{"path":"*","capabilities":["read"]}]'
```
- **Purpose**: Create new API client with policies
- **Output**: Client ID and secret (shown only once)
- **When to run**: To create new API clients for applications
- **Modes**: Interactive (prompts for input) or non-interactive (JSON policies via flag)

#### `update-client` - Update Authentication Client
```bash
# Update client configuration
./bin/app update-client \
  --id "018d7e95-1a23-7890-bcde-f1234567890a" \
  --name "updated-name" \
  --policies '[{"path":"*","capabilities":["read"]}]'

# Deactivate a client
./bin/app update-client --id "..." --name "..." --active=false \
  --policies '[{"path":"*","capabilities":["read"]}]'
```
- **Purpose**: Update existing client's name, active status, or policies
- **When to run**: To modify client permissions or deactivate compromised clients
- **Note**: Does not change the client secret (cannot be retrieved/changed)

### 🔨 Build Commands

```bash
make build              # Build the application binary
make run-server         # Build and run HTTP server
make run-migrate        # Build and run database migrations
make clean              # Remove build artifacts
```

### ✅ Code Quality

```bash
make lint               # Run golangci-lint with auto-fix
make test               # Run all tests with coverage
make test-coverage      # View coverage report in browser
```

### 🗄️ Database Management

```bash
# Start test databases
make test-db-up

# Run tests with real databases
make test

# Stop test databases
make test-db-down

# Or run everything in one command
make test-with-db
```

### 🎯 Running a Single Test

```bash
# Run specific test function
go test -v -race -run TestKekUseCase_Create ./internal/crypto/usecase

# Run specific subtest
go test -v -race -run "TestKekUseCase_Create/Success" ./internal/crypto/usecase

# Run all tests in a package
go test -v -race ./internal/crypto/usecase
```

## 🧪 Testing

The project uses real databases (PostgreSQL and MySQL) for integration testing instead of mocks, ensuring tests accurately reflect production behavior. All repository implementations (KEK and DEK) include comprehensive test coverage for both databases, including transaction handling, foreign key constraints, and database-specific binary data storage.

### 📝 Test Structure

```go
func TestKekUseCase_Create(t *testing.T) {
    t.Run("Success_CreateKekWithAESGCM", func(t *testing.T) {
        // Setup mocks
        mockRepo := mocks.NewMockKekRepository(t)
        
        // Create test data
        masterKey := &cryptoDomain.MasterKey{
            ID:  "test-master-key",
            Key: make([]byte, 32),
        }
        
        // Setup expectations
        mockRepo.EXPECT().Create(ctx, mock.Anything).Return(nil).Once()
        
        // Execute
        err := useCase.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)
        
        // Assert
        assert.NoError(t, err)
})
```

### 📊 Test Coverage

Run tests with coverage report:

```bash
make test-coverage
```

Current coverage is tracked in `coverage.out` and viewable in HTML format.

## 🔒 Security

### 🛡️ Security Best Practices

1. 🔐 **Master Key Storage**: Store master keys in a secure KMS (AWS KMS, HashiCorp Vault, etc.) in production
2. 🔄 **Key Rotation**: Regularly rotate master keys and KEKs (recommended: every 90 days)
3. 🛡️ **Access Control**: Use policies to implement principle of least privilege
4. 📜 **Audit Logs**: Monitor audit logs for suspicious activity
5. 🔒 **TLS/SSL**: Always use HTTPS in production
6. 💾 **Database Encryption**: Enable database-level encryption at rest

### ⚠️ Threat Model

- 💾 **Compromised Database**: Secrets remain encrypted; attacker needs master key
- 🔑 **Compromised Master Key**: Rotate master key and re-encrypt KEKs
- 🔐 **Compromised KEK**: Rotate KEK and re-encrypt affected DEKs
- 🎯 **Compromised DEK**: Only affects single secret version; rotate secret
- 📜 **Tampered Audit Logs**: Hash chain provides tamper detection

### ✨ Security Features

- 🔒 **AEAD Encryption**: Authenticated encryption prevents tampering
- 🧹 **Memory Zeroing**: Sensitive key material cleared from memory after use
- ⏱️ **Time-limited Tokens**: Tokens expire after configurable period
- 🚫 **Token Revocation**: Tokens can be revoked before expiration
- 🗑️ **Soft Deletion**: Secrets marked as deleted but retained for audit

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Allisson Azevedo

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 🙏 Acknowledgments

This project leverages these excellent Go libraries:

- [gin-gonic/gin](https://github.com/gin-gonic/gin) - High-performance HTTP web framework
- [google/uuid](https://github.com/google/uuid) - UUID generation with UUIDv7 support
- [jellydator/validation](https://github.com/jellydator/validation) - Advanced input validation
- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [allisson/go-env](https://github.com/allisson/go-env) - Environment configuration
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Database migrations
- [stretchr/testify](https://github.com/stretchr/testify) - Testing framework

---

**Built with 💙 security and scalability in mind by [Allisson Azevedo](https://github.com/allisson)**
