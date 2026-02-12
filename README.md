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

### 🛡️ Access Control & Authentication

- 👤 **Client Authentication** - API clients with secret-based authentication
- 🎫 **Token Management** - Time-limited tokens with expiration and revocation
- 📋 **Policy-based Authorization** - JSON policy documents for fine-grained access control
- 🔗 **Client-Policy Binding** - Associate multiple policies with each client
- 🔌 **Client Management API** - REST endpoints for CRUD operations on API clients (Create, Read, Update, Delete)

### 🔒 Security & Compliance

- 📜 **Immutable Audit Logs** - Cryptographic hash chaining for tamper-evident logging
- 📊 **Comprehensive Logging** - Track all operations with actor, action, resource, and metadata
- 🧹 **Secure Memory Handling** - Automatic zeroing of sensitive key material
- 💾 **Database Encryption** - All sensitive data encrypted at rest

### 🏗️ Architecture & Design

- 🎯 **Clean Architecture** - Clear separation of domain, use case, repository, and presentation layers
- 🧩 **Domain-Driven Design** - Business logic encapsulated in domain models
- 🗄️ **Multi-Database Support** - PostgreSQL and MySQL with dedicated repository implementations
- 🔑 **Complete Repository Layer** - KEK and DEK repositories with transaction support and database-specific optimizations
- ⚡ **Transaction Management** - ACID guarantees for atomic operations (key rotation, secret updates)
- 💉 **Dependency Injection** - Centralized wiring with lazy initialization
- 🌐 **Gin Web Framework** - High-performance HTTP router (v1.11.0) with custom slog middleware, REST API under `/v1/`, and capability-based authorization

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
│   │   └── http/               # HTTP handlers and middleware
│   ├── crypto/                 # Cryptographic domain module
│   │   ├── domain/             # Entities: Kek, Dek, MasterKey
│   │   ├── service/            # Encryption services
│   │   ├── usecase/            # Business logic orchestration
│   │   └── repository/         # Data access: Kek and Dek repositories (PostgreSQL & MySQL)
│   └── secrets/                # Secrets management module
│       ├── domain/             # Entities: Secret (with versioning)
│       ├── usecase/            # Secret operations (create/update/get/delete)
│       └── repository/         # Secret persistence (PostgreSQL & MySQL)
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

# 9. Start the server
./bin/app server
```

The server will be available at `http://localhost:8080`. Test with:

```bash
curl http://localhost:8080/health
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
    "capabilities": ["read", "write", "delete"]
  },
  {
    "path": "/v1/transit/*",
    "capabilities": ["encrypt", "decrypt"]
  }
]
```

**Note:** Currently only `/v1/clients` endpoints are implemented. The `/v1/secrets` and `/v1/transit` paths shown above are examples for when those APIs become available (see Planned Features section below).

## 🚧 Planned Features

The following API endpoints are planned but not yet implemented. The underlying business logic (domain models, use cases, repositories) exists, but HTTP handlers are under development.

### 📦 Secrets Management API

**Status:** 🚧 Under Development

Secrets are managed with automatic versioning - every update creates a new version while preserving the complete history.

#### Create/Update Secret

Creates a new secret or a new version of an existing secret. Each version is stored as a separate database record with its own Data Encryption Key (DEK).

```bash
POST /v1/secrets
```

**Request Body:**
```json
{
  "path": "/app/production/database-password",
  "value": "super-secret-password"
}
```

**Response (New Secret - Version 1):**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "path": "/app/production/database-password",
  "version": 1,
  "created_at": "2026-02-02T20:13:45Z"
}
```

**Example: Updating a Secret**

When you update an existing secret, a new version is automatically created:

```bash
# First creation (version 1)
curl -X POST http://localhost:8080/v1/secrets \
  -H "Content-Type: application/json" \
  -d '{"path": "/app/prod/api-key", "value": "secret-v1"}'

# Update creates version 2
curl -X POST http://localhost:8080/v1/secrets \
  -H "Content-Type: application/json" \
  -d '{"path": "/app/prod/api-key", "value": "secret-v2"}'

# Another update creates version 3
curl -X POST http://localhost:8080/v1/secrets \
  -H "Content-Type: application/json" \
  -d '{"path": "/app/prod/api-key", "value": "secret-v3"}'
```

**Version Management:**
- ✅ **Automatic Versioning**: Version number auto-increments on each update
- ✅ **Immutable History**: Previous versions remain unchanged in the database
- ✅ **Independent Encryption**: Each version has its own DEK for maximum security
- ✅ **Audit Trail**: Complete history of all secret changes preserved

#### Get Secret

Retrieves and decrypts the **latest version** of a secret at the specified path.

```bash
GET /v1/secrets?path=/app/production/database-password
```

**Response:**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "path": "/app/production/database-password",
  "value": "super-secret-password",
  "version": 3,
  "created_at": "2026-02-02T20:15:30Z"
}
```

**Notes:**
- 🔍 Returns the most recent (highest version number) secret
- 🔓 Automatically decrypts the secret value using the envelope encryption chain
- 📊 Includes version number to track which version is current

#### Delete Secret (Soft Delete)

Performs a soft delete on the **current version** of a secret. The secret is marked as deleted but preserved in the database for audit purposes.

```bash
DELETE /v1/secrets?path=/app/production/database-password
```

**Behavior:**
- 🗑️ Sets the `deleted_at` timestamp on the current version
- 📜 Preserves the secret data for audit trail and compliance
- 🔒 Previous versions remain unaffected
- ⚠️ Deleted secrets cannot be retrieved via the API

**Example:**
```bash
# Delete the current version of a secret
curl -X DELETE "http://localhost:8080/v1/secrets?path=/app/prod/api-key"
```

### 🚄 Transit Encryption API (Encryption-as-a-Service)

**Status:** 🚧 Under Development

#### Create Transit Key

```bash
POST /v1/transit/keys
```

**Request Body:**
```json
{
  "name": "payment-encryption",
  "algorithm": "aes-gcm"
}
```

#### Encrypt Data

```bash
POST /v1/transit/encrypt/{key_name}
```

**Request Body:**
```json
{
  "plaintext": "sensitive-data-to-encrypt"
}
```

**Response:**
```json
{
  "ciphertext": "vault:v1:base64-encoded-ciphertext"
}
```

#### Decrypt Data

```bash
POST /v1/transit/decrypt/{key_name}
```

**Request Body:**
```json
{
  "ciphertext": "vault:v1:base64-encoded-ciphertext"
}
```

**Response:**
```json
{
  "plaintext": "sensitive-data-to-encrypt"
}
```

### 📜 Audit Logs API

**Status:** 🚧 Under Development

#### List Audit Logs

```bash
GET /v1/audit-logs?limit=100&offset=0
```

**Response:**
```json
{
  "logs": [
    {
      "id": "018d7e95-1a23-7890-bcde-f1234567890a",
      "actor": "client-id-or-system",
      "action": "secrets.create",
      "resource": "/app/production/database-password",
      "metadata": {
        "ip": "192.168.1.100",
        "user_agent": "curl/7.68.0"
      },
      "entry_hash": "sha256-hash",
      "previous_hash": "sha256-previous-hash",
      "created_at": "2026-02-02T20:13:45Z"
    }
  ],
  "total": 1234
}
```

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
- **Endpoints**: Health check, secrets, transit, clients, policies, audit logs

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
