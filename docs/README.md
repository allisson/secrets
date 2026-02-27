# 📚 Secrets Documentation

> Last updated: 2026-02-26

Metadata source for release/API labels: `docs/metadata.json`

> [!WARNING]
> While in versions `v0.x.y`, this project is not yet recommended for production deployment and the API is not yet stable and is subject to many changes. It will only be recommended for production when it reaches version `v1.0.0`.

Welcome to the full documentation for Secrets. Pick a path and dive in 🚀

## 🧭 Start Here

- 🐳 [getting-started/docker.md](getting-started/docker.md) (recommended)
- 💻 [getting-started/local-development.md](getting-started/local-development.md)
- 🧭 [getting-started/day-0-walkthrough.md](getting-started/day-0-walkthrough.md)
- 🧰 [operations/troubleshooting/index.md](operations/troubleshooting/index.md)
- ✅ [getting-started/smoke-test.md](getting-started/smoke-test.md)
- 🧪 [cli-commands.md](cli-commands.md)

## 🛣️ First-Time Operator Path

1. Start with Docker guide: [getting-started/docker.md](getting-started/docker.md)
2. Validate end-to-end setup: [getting-started/smoke-test.md](getting-started/smoke-test.md)
3. Follow rollout runbook: [operations/deployment/production-rollout.md](operations/deployment/production-rollout.md)
4. Apply production hardening checklist: [operations/deployment/docker-hardened.md](operations/deployment/docker-hardened.md)
5. Use runbook hub for rollout and incidents: [operations/runbooks/README.md](operations/runbooks/README.md)

## 👥 Persona Paths

- 👷 [Operator](personas/README.md#operator-path)
- 👨‍💻 [Developer](personas/README.md#developer-path)
- 🛡️ [Security Engineer](personas/README.md#security-engineer-path)

## 📖 Documentation by Topic

**Configuration & Concepts:**

- ⚙️ [configuration.md](configuration.md)
- 🏗️ [concepts/architecture.md](concepts/architecture.md)
- 🔒 [concepts/security-model.md](concepts/security-model.md)
- 📘 [concepts/architecture.md#glossary](concepts/architecture.md#glossary)

**Operations: KMS & Key Management:**

- ☁️ [operations/kms/setup.md](operations/kms/setup.md) - KMS setup and migration
- 🔑 [operations/kms/key-management.md](operations/kms/key-management.md)

**Operations: Security:**

- 🛡️ [operations/deployment/docker-hardened.md](operations/deployment/docker-hardened.md) - Includes trusted proxy configuration

**Operations: Observability:**

- 📊 [operations/observability/monitoring.md](operations/observability/monitoring.md)
- 🚑 [operations/observability/incident-response.md](operations/observability/incident-response.md)

**Operations: Deployment:**

- 🚀 [operations/deployment/production-rollout.md](operations/deployment/production-rollout.md)
- 🏭 [operations/deployment/docker-hardened.md](operations/deployment/docker-hardened.md) - Includes known limitations

**Operations: Runbooks:**

- 🧭 [operations/runbooks/README.md](operations/runbooks/README.md) - Runbook hub
- ⚡ [operations/runbooks/README.md#operator-quick-card](operations/runbooks/README.md#operator-quick-card)
- 🧯 [operations/runbooks/README.md#operator-drills-quarterly](operations/runbooks/README.md#operator-drills-quarterly)
- 🧪 [operations/runbooks/policy-smoke-tests.md](operations/runbooks/policy-smoke-tests.md)

**Development:**

- 🤝 [contributing.md](contributing.md) - Includes testing, docs architecture map, release checklist, and documentation management

## 🧭 Docs Freshness SLA

| Area | Primary owner | Review cadence |
| --- | --- | --- |
| Getting started | Maintainers | Monthly |
| API reference | Maintainers + feature PR author | Every behavior change + monthly |
| Operations runbooks | Maintainers + on-call | Monthly and after incidents |
| Examples | Maintainers | Monthly and when API contract changes |
| Concepts/architecture | Maintainers | Quarterly |

## 🌐 API Reference

- 🔐 [api/auth/authentication.md](api/auth/authentication.md)
- 👤 [api/auth/clients.md](api/auth/clients.md)
- 📘 [api/auth/policies.md](api/auth/policies.md)
- 📦 [api/data/secrets.md](api/data/secrets.md)
- 🚄 [api/data/transit.md](api/data/transit.md)
- 🎫 [api/data/tokenization.md](api/data/tokenization.md)
- 📜 [api/observability/audit-logs.md](api/observability/audit-logs.md)
- 🧱 [api/observability/response-shapes.md](api/observability/response-shapes.md)
- 🧩 [api/fundamentals.md](api/fundamentals.md) - Error triage, capabilities, rate limits, versioning
- 📄 [openapi.yaml](openapi.yaml)

## 🔎 Search Aliases

- `401 403 429 decision tree incident` -> [operations/observability/incident-response.md](operations/observability/incident-response.md)
- `first 15 minutes incident playbook` -> [operations/observability/incident-response.md](operations/observability/incident-response.md)
- `trusted proxy retry-after token 429` -> [operations/deployment/docker-hardened.md](operations/deployment/docker-hardened.md)
- `known limitations` -> [operations/deployment/docker-hardened.md](operations/deployment/docker-hardened.md)
- `examples` -> [examples/README.md](examples/README.md)

OpenAPI scope note:

- `openapi.yaml` is a baseline subset for common API flows in the current release (v0.19.0, see `docs/metadata.json`)
- Full endpoint behavior is documented in the endpoint pages under `docs/api/`
- Tokenization endpoints are included in `openapi.yaml` for the current release

## 🚀 Releases

- 📦 [releases/RELEASES.md](releases/RELEASES.md) - All release notes

## 🧠 Architecture Decision Records

This section documents key architectural decisions with their context, rationale, and trade-offs:

- 🧾 [ADR 0001: Envelope Encryption Model](adr/0001-envelope-encryption-model.md) - Master Key → KEK → DEK → Secret Data hierarchy
- 🧾 [ADR 0002: Transit Versioned Ciphertext Contract](adr/0002-transit-versioned-ciphertext-contract.md) - `<version>:<base64-ciphertext>` format
- 🧾 [ADR 0003: Capability-Based Authorization Model](adr/0003-capability-based-authorization-model.md) - Fine-grained access control with path matching
- 🧾 [ADR 0004: Dual Database Support](adr/0004-dual-database-support.md) - PostgreSQL and MySQL compatibility
- 🧾 [ADR 0005: Context-Based Transaction Management](adr/0005-context-based-transaction-management.md) - Go context for transaction propagation
- 🧾 [ADR 0006: Dual-Scope Rate Limiting Strategy](adr/0006-dual-scope-rate-limiting-strategy.md) - Per-client and per-IP rate limiting
- 🧾 [ADR 0007: Path-Based API Versioning](adr/0007-path-based-api-versioning.md) - `/v1/*` API versioning strategy
- 🧾 [ADR 0008: Gin Web Framework with Custom Middleware](adr/0008-gin-web-framework-with-custom-middleware.md) - HTTP framework and middleware strategy
- 🧾 [ADR 0009: UUIDv7 for Identifiers](adr/0009-uuidv7-for-identifiers.md) - Time-ordered UUID strategy for database IDs
- 🧾 [ADR 0010: Argon2id for Client Secret Hashing](adr/0010-argon2id-for-client-secret-hashing.md) - Memory-hard password hashing algorithm
- 🧾 [ADR 0011: HMAC-SHA256 Cryptographic Signing for Audit Log Integrity](adr/0011-hmac-sha256-audit-log-signing.md) - Tamper detection for audit logs

## 🖥️ Supported Platforms

- ✅ Linux and macOS environments for local development and operations
- ✅ Docker-based runtime recommended for all environments
- ✅ CI validates with Go `1.25.5`, PostgreSQL `16-alpine`, and MySQL `8.0`
- ℹ️ Project compatibility targets include PostgreSQL `12+` and MySQL `8.0+`

## 💡 Practical Examples

- 🧭 [examples/README.md](examples/README.md) - Code examples overview and version compatibility
- 🧪 [examples/curl.md](examples/curl.md)
- 🐍 [examples/python.md](examples/python.md)
- 🟨 [examples/javascript.md](examples/javascript.md)
- 🐹 [examples/go.md](examples/go.md)

## 🧩 Positioning

Secrets is inspired by HashiCorp Vault, but it is much simpler and intentionally focused on core use cases. It is not designed to compete with Vault.

## See also

- [Docker getting started](getting-started/docker.md)
- [Architecture](concepts/architecture.md)
- [Authentication API](api/auth/authentication.md)
- [Production operations](operations/deployment/docker-hardened.md)
