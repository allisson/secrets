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

- 📚 **Versioned Secrets** - Store secrets with full version history and rollback capability
- 🗂️ **Path-based Organization** - Hierarchical secret organization (e.g., `/app/prod/db-password`)
- 🗑️ **Soft Deletion** - Mark secrets as deleted without losing historical data

### 🛡️ Access Control & Authentication

- 👤 **Client Authentication** - API clients with secret-based authentication
- 🎫 **Token Management** - Time-limited tokens with expiration and revocation
- 📋 **Policy-based Authorization** - JSON policy documents for fine-grained access control
- 🔗 **Client-Policy Binding** - Associate multiple policies with each client

### 🔒 Security & Compliance

- 📜 **Immutable Audit Logs** - Cryptographic hash chaining for tamper-evident logging
- 📊 **Comprehensive Logging** - Track all operations with actor, action, resource, and metadata
- 🧹 **Secure Memory Handling** - Automatic zeroing of sensitive key material
- 💾 **Database Encryption** - All sensitive data encrypted at rest

### 🏗️ Architecture & Design

- 🎯 **Clean Architecture** - Clear separation of domain, use case, repository, and presentation layers
- 🧩 **Domain-Driven Design** - Business logic encapsulated in domain models
- 🗄️ **Multi-Database Support** - PostgreSQL and MySQL with dedicated repository implementations
- ⚡ **Transaction Management** - ACID guarantees for atomic operations
- 💉 **Dependency Injection** - Centralized wiring with lazy initialization
- 🌐 **RESTful API** - JSON-based HTTP API with standard status codes

## 🏗️ Architecture

### 🔐 Envelope Encryption Model

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

### ✅ Key Benefits

- ⚡ **Fast Key Rotation**: Rotate master keys or KEKs without re-encrypting all secrets
- 🔒 **Per-Secret Security**: Each secret version has its own DEK
- 🎨 **Algorithm Flexibility**: Different encryption algorithms per key tier
- 📈 **Scalability**: Minimal performance impact from key rotation

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
│   └── crypto/                 # Cryptographic domain module
│       ├── domain/             # Entities: Kek, Dek, MasterKey
│       ├── service/            # Encryption services
│       ├── usecase/            # Business logic orchestration
│       └── repository/         # Data access (PostgreSQL & MySQL)
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

### 📦 Installation

```bash
# Clone the repository
git clone https://github.com/allisson/secrets.git
cd secrets

# Install dependencies
go mod download

# Generate a master key (base64-encoded 32-byte key)
openssl rand -base64 32

# Create .env file from example
cp .env.example .env

# Edit .env and set your MASTER_KEYS and database connection
# MASTER_KEYS=default:<your-base64-key>
```

### 🐳 Running with Docker

```bash
# Start PostgreSQL database
make dev-postgres

# Run database migrations
make run-migrate

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

Example:
```bash
# Generate a new 256-bit key
openssl rand -base64 32

# Set in environment
MASTER_KEYS=default:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=,backup:xYz123...
ACTIVE_MASTER_KEY_ID=default
```

## 💻 Usage

### 🗄️ Database Migrations

```bash
# Run migrations
make run-migrate

# Or using Docker
make docker-run-migrate
```

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

All API endpoints (except `/health`) require authentication using client tokens:

```bash
# Include token in Authorization header
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/secrets
```

### 🔑 Key Management Operations

#### Create Initial KEK

```bash
POST /api/keks/create
```

Creates the first Key Encryption Key using the active master key.

**Request Body:**
```json
{
  "algorithm": "aes-gcm"  # Options: "aes-gcm", "chacha20-poly1305"
}
```

#### Rotate KEK

```bash
POST /api/keks/rotate
```

Creates a new KEK version and marks the previous one as inactive.

**Request Body:**
```json
{
  "algorithm": "aes-gcm"
}
```

### 📦 Secrets Operations

#### Create/Update Secret

```bash
POST /api/secrets
```

**Request Body:**
```json
{
  "path": "/app/production/database-password",
  "value": "super-secret-password"
}
```

**Response:**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "path": "/app/production/database-password",
  "version": 1,
  "created_at": "2026-02-02T20:13:45Z"
}
```

#### Get Secret

```bash
GET /api/secrets?path=/app/production/database-password
```

**Response:**
```json
{
  "id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "path": "/app/production/database-password",
  "value": "super-secret-password",
  "version": 2,
  "created_at": "2026-02-02T20:13:45Z"
}
```

#### Get Secret Version

```bash
GET /api/secrets/versions/{version_id}
```

#### List Secret Versions

```bash
GET /api/secrets/{secret_id}/versions
```

#### Delete Secret (Soft Delete)

```bash
DELETE /api/secrets/{secret_id}
```

### 🚄 Transit Encryption (Encryption-as-a-Service)

#### Create Transit Key

```bash
POST /api/transit/keys
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
POST /api/transit/encrypt/{key_name}
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
POST /api/transit/decrypt/{key_name}
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

### 👤 Client Management

#### Create Client

```bash
POST /api/clients
```

**Request Body:**
```json
{
  "name": "production-app",
  "secret": "client-secret-value"
}
```

#### Create Token

```bash
POST /api/tokens
```

**Request Body:**
```json
{
  "client_id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "client_secret": "client-secret-value",
  "expires_in": 3600  # Expiration in seconds
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-02-02T21:13:45Z"
}
```

### 📋 Policy Management

#### Create Policy

```bash
POST /api/policies
```

**Request Body:**
```json
{
  "name": "read-production-secrets",
  "document": {
    "version": "1",
    "statements": [
      {
        "effect": "allow",
        "actions": ["secrets:read"],
        "resources": ["/app/production/*"]
      }
    ]
  }
}
```

#### Attach Policy to Client

```bash
POST /api/clients/{client_id}/policies/{policy_id}
```

### 📜 Audit Logs

#### List Audit Logs

```bash
GET /api/audit-logs?limit=100&offset=0
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

The project uses real databases (PostgreSQL and MySQL) for integration testing instead of mocks, ensuring tests accurately reflect production behavior.

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

- [google/uuid](https://github.com/google/uuid) - UUID generation with UUIDv7 support
- [jellydator/validation](https://github.com/jellydator/validation) - Advanced input validation
- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [allisson/go-env](https://github.com/allisson/go-env) - Environment configuration
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Database migrations
- [stretchr/testify](https://github.com/stretchr/testify) - Testing framework

---

**Built with 💙 security and scalability in mind by [Allisson Azevedo](https://github.com/allisson)**
