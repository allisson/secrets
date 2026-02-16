# 📚 Secrets Documentation

> Last updated: 2026-02-16

Welcome to the full documentation for Secrets. Pick a path and dive in 🚀

## 🧭 Start Here

- 🐳 [getting-started/docker.md](getting-started/docker.md) (recommended)
- 💻 [getting-started/local-development.md](getting-started/local-development.md)
- 🧰 [getting-started/troubleshooting.md](getting-started/troubleshooting.md)
- ✅ [getting-started/smoke-test.md](getting-started/smoke-test.md)
- 🧪 [cli/commands.md](cli/commands.md)

## 🛣️ First-Time Operator Path

1. Start with Docker guide: [getting-started/docker.md](getting-started/docker.md)
2. Validate end-to-end setup: [getting-started/smoke-test.md](getting-started/smoke-test.md)
3. Apply production hardening checklist: [operations/production.md](operations/production.md)

## 📖 Documentation by Topic

- ⚙️ [configuration/environment-variables.md](configuration/environment-variables.md)
- 🏗️ [concepts/architecture.md](concepts/architecture.md)
- 🔒 [concepts/security-model.md](concepts/security-model.md)
- 📘 [concepts/glossary.md](concepts/glossary.md)
- 🔑 [operations/key-management.md](operations/key-management.md)
- 📊 [operations/monitoring.md](operations/monitoring.md)
- 🏭 [operations/production.md](operations/production.md)
- 🚑 [operations/failure-playbooks.md](operations/failure-playbooks.md)
- 🛠️ [development/testing.md](development/testing.md)
- 🤝 [contributing.md](contributing.md)
- 🗒️ [CHANGELOG.md](CHANGELOG.md)

## 🧭 Docs Freshness SLA

| Area | Primary owner | Review cadence |
| --- | --- | --- |
| Getting started | Maintainers | Monthly |
| API reference | Maintainers + feature PR author | Every behavior change + monthly |
| Operations runbooks | Maintainers + on-call | Monthly and after incidents |
| Examples | Maintainers | Monthly and when API contract changes |
| Concepts/architecture | Maintainers | Quarterly |

## 🌐 API Reference

- 🔐 [api/authentication.md](api/authentication.md)
- 👤 [api/clients.md](api/clients.md)
- 📘 [api/policies.md](api/policies.md)
- 📦 [api/secrets.md](api/secrets.md)
- 🚄 [api/transit.md](api/transit.md)
- 📜 [api/audit-logs.md](api/audit-logs.md)
- 🧱 [api/response-shapes.md](api/response-shapes.md)
- 🧩 [api/versioning-policy.md](api/versioning-policy.md)
- 📄 [openapi.yaml](openapi.yaml)

OpenAPI scope note:

- `openapi.yaml` is a baseline subset for common API flows in `v0.3.0`
- Full endpoint behavior is documented in the endpoint pages under `docs/api/`

## 🚀 Releases

- 📦 [releases/v0.3.0.md](releases/v0.3.0.md)
- 📦 [releases/v0.2.0.md](releases/v0.2.0.md) (historical)
- 📦 [releases/v0.1.0.md](releases/v0.1.0.md) (historical)

## 🧠 ADRs

- 🧾 [adr/0001-envelope-encryption-model.md](adr/0001-envelope-encryption-model.md)
- 🧾 [adr/0002-transit-versioned-ciphertext-contract.md](adr/0002-transit-versioned-ciphertext-contract.md)

## 🖥️ Supported Platforms

- ✅ Linux and macOS environments for local development and operations
- ✅ Docker-based runtime recommended for all environments
- ✅ CI validates with Go `1.25.5`, PostgreSQL `16-alpine`, and MySQL `8.0`
- ℹ️ Project compatibility targets include PostgreSQL `12+` and MySQL `8.0+`

## 💡 Practical Examples

- 🧪 [examples/curl.md](examples/curl.md)
- 🐍 [examples/python.md](examples/python.md)
- 🟨 [examples/javascript.md](examples/javascript.md)
- 🐹 [examples/go.md](examples/go.md)

## 🧩 Positioning

Secrets is inspired by HashiCorp Vault, but it is much simpler and intentionally focused on core use cases. It is not designed to compete with Vault.

## See also

- [Docker getting started](getting-started/docker.md)
- [Architecture](concepts/architecture.md)
- [Authentication API](api/authentication.md)
- [Production operations](operations/production.md)
