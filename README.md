# 🔐 Secrets

A lightweight secrets manager with envelope encryption, transit encryption, API auth, and audit logs.

[![CI](https://github.com/allisson/secrets/workflows/CI/badge.svg)](https://github.com/allisson/secrets/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/allisson/secrets)](https://goreportcard.com/report/github.com/allisson/secrets)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Secrets is inspired by **HashiCorp Vault** ❤️, but it is intentionally **much simpler** and was **not designed to compete with Vault**.

> [!WARNING]
> While in versions `v0.x.y`, this project is not yet recommended for production deployment and the API is not yet stable and is subject to many changes. It will only be recommended for production when it reaches version `v1.0.0`.

## ✨ Features

- **Authentication & Authorization:** Token-based auth with Argon2id password hashing and capability-based path-matching policies.
- **KMS Integration:** Native support for Google Cloud KMS, AWS KMS, Azure Key Vault, and HashiCorp Vault.
- **Dual Database Support:** Compatible with PostgreSQL 12+ and MySQL 8.0+ out of the box.
- **Observability:** OpenTelemetry metrics with Prometheus-compatible endpoints.

## 📦 Main Engines

### [Secret Engine](docs/engines/secrets.md)

Provides versioned, encrypted storage for your application secrets using envelope encryption. Keep passwords and API keys secure at rest.

### [Transit Engine](docs/engines/transit.md)

Offers Encryption as a Service (EaaS). Encrypt and decrypt data on the fly without storing the payload in the Secrets database.

### [Tokenization Engine](docs/engines/tokenization.md)

Format-preserving token generation for sensitive values (e.g., credit cards) with deterministic options and lifecycle management.

### [Audit Logs](docs/observability/audit-logs.md)

Tamper-resistant cryptographic audit logs capture capability checks and access attempts for monitoring and compliance.

## 🚀 Quick Start

Choose your preferred deployment method to get started:

1. 🐳 **Run with Docker image (recommended)**: [Docker Guide](docs/getting-started/docker.md)
2. 💻 **Run locally for development**: [Local Development Guide](docs/getting-started/local-development.md)
3. 📦 **Run with pre-compiled binary**: [Binary Guide](docs/getting-started/binary.md)

## 📚 Documentation

See our detailed guides in the `docs/` directory:

- [API Authentication](docs/auth/authentication.md)
- [Client Management](docs/auth/clients.md)
- [Policies Cookbook](docs/auth/policies.md)
- [CLI Commands](docs/cli-commands.md)

## 📄 License

MIT. See `LICENSE`.
