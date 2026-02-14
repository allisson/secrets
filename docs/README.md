# 📚 Secrets Documentation

> Last updated: 2026-02-14

Welcome to the full documentation for Secrets. Pick a path and dive in 🚀

## 🧭 Start Here

- 🐳 [getting-started/docker.md](getting-started/docker.md) (recommended)
- 💻 [getting-started/local-development.md](getting-started/local-development.md)
- 🧰 [getting-started/troubleshooting.md](getting-started/troubleshooting.md)
- ✅ [getting-started/smoke-test.md](getting-started/smoke-test.md)

## 🛣️ First-Time Operator Path

1. Start with Docker guide: [getting-started/docker.md](getting-started/docker.md)
2. Validate end-to-end setup: [getting-started/smoke-test.md](getting-started/smoke-test.md)
3. Apply production hardening checklist: [operations/production.md](operations/production.md)

## 📖 Documentation by Topic

- ⚙️ [configuration/environment-variables.md](configuration/environment-variables.md)
- 🏗️ [concepts/architecture.md](concepts/architecture.md)
- 🔒 [concepts/security-model.md](concepts/security-model.md)
- 🔑 [operations/key-management.md](operations/key-management.md)
- 🏭 [operations/production.md](operations/production.md)
- 🛠️ [development/testing.md](development/testing.md)
- 🤝 [contributing.md](contributing.md)
- 🗒️ [CHANGELOG.md](CHANGELOG.md)

## 🌐 API Reference

- 🔐 [api/authentication.md](api/authentication.md)
- 👤 [api/clients.md](api/clients.md)
- 📘 [api/policies.md](api/policies.md)
- 📦 [api/secrets.md](api/secrets.md)
- 🚄 [api/transit.md](api/transit.md)
- 📜 [api/audit-logs.md](api/audit-logs.md)
- 🧱 [api/response-shapes.md](api/response-shapes.md)
- 📄 [openapi.yaml](openapi.yaml)

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
