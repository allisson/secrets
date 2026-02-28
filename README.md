# 🔐 Secrets

> Last updated: 2026-02-28
>
> A lightweight secrets manager with envelope encryption, transit encryption, API auth, and audit logs.

[![CI](https://github.com/allisson/secrets/workflows/CI/badge.svg)](https://github.com/allisson/secrets/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/allisson/secrets)](https://goreportcard.com/report/github.com/allisson/secrets)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Secrets is inspired by **HashiCorp Vault** ❤️, but it is intentionally **much simpler** and was **not designed to compete with Vault**.

> [!WARNING]
> While in versions `v0.x.y`, this project is not yet recommended for production deployment and the API is not yet stable and is subject to many changes. It will only be recommended for production when it reaches version `v1.0.0`.

## 🚀 Quick Start (Docker-first)

The default way to run Secrets is the published Docker image:

```bash
docker pull allisson/secrets
```

Use pinned tags for reproducible setups. `latest` is available for dev-only fast iteration.

Docs release/API metadata source: `docs/metadata.json`.

Then follow the Docker setup guide in [docs/getting-started/docker.md](docs/getting-started/docker.md).

⚠️ After rotating a master key or KEK, restart API server instances so they load the updated key material.

## 🧭 Choose Your Path

1. 🐳 **Run with Docker image (recommended)**: [docs/getting-started/docker.md](docs/getting-started/docker.md)
2. 💻 **Run locally for development**: [docs/getting-started/local-development.md](docs/getting-started/local-development.md)
3. 📦 **Run with pre-compiled binary**: [docs/getting-started/binary.md](docs/getting-started/binary.md)

Release history:

- All releases: [Release notes](docs/releases/RELEASES.md)

## 📚 Docs Map

- **Start Here**
- 🏁 **Docs index**: [docs/README.md](docs/README.md)
- 🚀 **Getting started (Docker)**: [docs/getting-started/docker.md](docs/getting-started/docker.md)
- 💻 **Getting started (local)**: [docs/getting-started/local-development.md](docs/getting-started/local-development.md)
- 📦 **Getting started (binary)**: [docs/getting-started/binary.md](docs/getting-started/binary.md)
- 🧰 **Troubleshooting**: [docs/operations/troubleshooting/index.md](docs/operations/troubleshooting/index.md)
- ✅ **Smoke test script**: [docs/getting-started/smoke-test.md](docs/getting-started/smoke-test.md)
- 🧪 **CLI commands reference**: [docs/cli-commands.md](docs/cli-commands.md)
- 📦 **All release notes**: [docs/releases/RELEASES.md](docs/releases/RELEASES.md)

- **By Topic**
  - ⚙️ **Environment variables**: [docs/configuration.md](docs/configuration.md)
  - 🏗️ **Architecture concepts**: [docs/concepts/architecture.md](docs/concepts/architecture.md)
  - 🔒 **Security model**: [docs/concepts/security-model.md](docs/concepts/security-model.md)
  - 📘 **Glossary**: [docs/concepts/architecture.md#glossary](docs/concepts/architecture.md#glossary)
  - 🔑 **Key management operations**: [docs/operations/kms/key-management.md](docs/operations/kms/key-management.md)
  - ☁️ **KMS setup guide**: [docs/operations/kms/setup.md](docs/operations/kms/setup.md)
  - ✅ **KMS migration checklist**: [docs/operations/kms/setup.md#migration-checklist](docs/operations/kms/setup.md#migration-checklist)
  - 🔐 **Security hardening**: [docs/operations/deployment/docker-hardened.md](docs/operations/deployment/docker-hardened.md)
  - 📊 **Monitoring and metrics**: [docs/operations/observability/monitoring.md](docs/operations/observability/monitoring.md)
  - 🧯 **Operator drills**: [docs/operations/runbooks/README.md#operator-drills-quarterly](docs/operations/runbooks/README.md#operator-drills-quarterly)
  - 🚀 **Production rollout golden path**: [docs/operations/deployment/production-rollout.md](docs/operations/deployment/production-rollout.md)
  - 🚨 **Incident response guide**: [docs/operations/observability/incident-response.md](docs/operations/observability/incident-response.md)
  - 🏭 **Production deployment**: [docs/operations/deployment/docker-hardened.md](docs/operations/deployment/docker-hardened.md)
  - 🛠️ **Development and testing**: [docs/contributing.md#development-and-testing](docs/contributing.md#development-and-testing)
  - 🗺️ **Docs architecture map**: [docs/contributing.md#docs-architecture-map](docs/contributing.md#docs-architecture-map)
  - 🤝 **Docs contributing**: [docs/contributing.md](docs/contributing.md)

Release note location:

- Project release notes (including documentation changes) are in [CHANGELOG.md](CHANGELOG.md)

- **API Reference**
  - 🔐 **Auth API**: [docs/api/auth/authentication.md](docs/api/auth/authentication.md)
  - 👤 **Clients API**: [docs/api/auth/clients.md](docs/api/auth/clients.md)
  - 📘 **Policy cookbook**: [docs/api/auth/policies.md](docs/api/auth/policies.md)
  - 📦 **Secrets API**: [docs/api/data/secrets.md](docs/api/data/secrets.md)
  - 🚄 **Transit API**: [docs/api/data/transit.md](docs/api/data/transit.md)
  - 🎫 **Tokenization API**: [docs/api/data/tokenization.md](docs/api/data/tokenization.md)
  - 📜 **Audit logs API**: [docs/api/observability/audit-logs.md](docs/api/observability/audit-logs.md)
  - 🧩 **API fundamentals**: [docs/api/fundamentals.md](docs/api/fundamentals.md) - Error triage, capabilities, rate limits, versioning

- **Examples**
- 🧪 **Curl examples**: [docs/examples/curl.md](docs/examples/curl.md)
- 🐍 **Python examples**: [docs/examples/python.md](docs/examples/python.md)
- 🟨 **JavaScript examples**: [docs/examples/javascript.md](docs/examples/javascript.md)
- 🐹 **Go examples**: [docs/examples/go.md](docs/examples/go.md)

All detailed guides include practical use cases and copy/paste-ready examples.

## ✨ What You Get

**Core Cryptography:**

- 🔐 **Envelope encryption** (`Master Key → KEK → DEK → Secret Data`) with [key rotation](docs/operations/kms/key-management.md)
- 🔑 **KMS integration** for master key encryption at rest (Google Cloud KMS, AWS KMS, Azure Key Vault, HashiCorp Vault) - [v0.6.0+](docs/operations/kms/setup.md)
- 🔄 **Dual algorithm support** (AES-GCM and ChaCha20-Poly1305) for envelope encryption

**Authentication & Authorization:**

- 🎫 **Token-based authentication** with Argon2id password hashing (memory-hard, GPU-resistant)
- 🛡️ **Capability-based authorization** with [path-matching policies](docs/api/auth/policies.md) (exact, wildcard, prefix)
- 🎭 **Policy templates** for common personas (read-only, CI writer, key operator, break-glass admin)
- 🚦 **Dual-scope rate limiting** (per-client for authenticated endpoints, per-IP for token issuance)

**Data Services:**

- 📦 **Versioned secrets** by path (`/v1/secrets/*path`) with automatic versioning
- 🚄 **Transit encryption** (`/v1/transit/*`) for encrypt/decrypt as a service with [key rotation](docs/api/data/transit.md#create-vs-rotate)
- 🎫 **Tokenization API** (`/v1/tokenization/*`) with token generation, detokenization, validation, revocation, and TTL expiration

**Security & Compliance:**

- 🔏 **Cryptographic audit log signing** with HMAC-SHA256 for tamper detection - [v0.9.0+](docs/releases/RELEASES.md#090---2026-02-20)
- 📜 **Comprehensive audit logs** with request correlation (`request_id`), filtering, and [integrity verification](docs/cli-commands.md#verify-audit-logs)
- 🧹 **Memory safety** with sensitive key material zeroing in critical paths
- 🔒 **AEAD encryption** for authenticated encryption with associated data

**Operations & Observability:**

- 🗄️ **Dual database support** (PostgreSQL 12+ and MySQL 8.0+) with driver-agnostic migrations
- 📊 **OpenTelemetry metrics** with Prometheus-compatible `/metrics` export
- 🧪 **CLI tooling** (`verify-audit-logs`, `rotate-kek`, `rewrap-deks`, `create-master-key`, `rotate-master-key`)
- 🌐 **CORS support** (configurable, disabled by default)
- 🏥 **Health endpoints** (`/health`, `/ready`) for Docker health checks
- 🧯 **Comprehensive documentation** with [runbooks](docs/operations/runbooks/README.md), [incident response guides](docs/operations/observability/incident-response.md), and [operator drills](docs/operations/runbooks/README.md#operator-drills-quarterly)

## 🌐 API Overview

- Health: `GET /health`
- Readiness: `GET /ready`
- Token issuance: `POST /v1/token`
- Clients: `GET/POST /v1/clients`, `GET/PUT/DELETE /v1/clients/:id`, `POST /v1/clients/:id/unlock`
- Secrets: `POST/GET/DELETE /v1/secrets/*path`
- Transit: `POST /v1/transit/keys`, `POST /v1/transit/keys/:name/rotate`, `POST /v1/transit/keys/:name/encrypt`, `POST /v1/transit/keys/:name/decrypt`, `DELETE /v1/transit/keys/:id` ([create vs rotate](docs/api/data/transit.md#create-vs-rotate), [error matrix](docs/api/data/transit.md#endpoint-error-matrix))
- Tokenization: `POST /v1/tokenization/keys`, `POST /v1/tokenization/keys/:name/rotate`, `DELETE /v1/tokenization/keys/:id`, `POST /v1/tokenization/keys/:name/tokenize`, `POST /v1/tokenization/detokenize`, `POST /v1/tokenization/validate`, `POST /v1/tokenization/revoke`
- Audit logs: `GET /v1/audit-logs`
- Metrics: `GET :8081/metrics` (prom-compatible, available when `METRICS_ENABLED=true`)

## 📄 License

MIT. See `LICENSE`.

## See also

- [Documentation index](docs/README.md)
- [Docker getting started](docs/getting-started/docker.md)
- [API authentication](docs/api/auth/authentication.md)
- [Production operations](docs/operations/deployment/docker-hardened.md)
