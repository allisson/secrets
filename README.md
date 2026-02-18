# 🔐 Secrets

> A lightweight secrets manager with envelope encryption, transit encryption, API auth, and audit logs.

[![CI](https://github.com/allisson/secrets/workflows/CI/badge.svg)](https://github.com/allisson/secrets/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/allisson/secrets)](https://goreportcard.com/report/github.com/allisson/secrets)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Secrets is inspired by **HashiCorp Vault** ❤️, but it is intentionally **much simpler** and was **not designed to compete with Vault**.

## 🚀 Quick Start (Docker-first)

The default way to run Secrets is the published Docker image:

```bash
docker pull allisson/secrets:v0.4.0
```

Use pinned tags for reproducible setups. `latest` is also available for fast iteration.

Docs release/API metadata source: `docs/metadata.json`.

Then follow the Docker setup guide in [docs/getting-started/docker.md](docs/getting-started/docker.md).

⚠️ After rotating a master key or KEK, restart API server instances so they load the updated key material.

## 🧭 Choose Your Path

1. 🐳 **Run with Docker image (recommended)**: [docs/getting-started/docker.md](docs/getting-started/docker.md)
2. 💻 **Run locally for development**: [docs/getting-started/local-development.md](docs/getting-started/local-development.md)

## 🆕 What's New in v0.4.0

- 🎫 Tokenization API for format-preserving token workflows (`/v1/tokenization/*`)
- 🧰 New tokenization CLI commands: `create-tokenization-key`, `rotate-tokenization-key`, `clean-expired-tokens`
- 🗄️ Tokenization persistence migrations for PostgreSQL and MySQL (`000002_add_tokenization`)
- 📈 Tokenization business-operation metrics added to observability
- 📘 New release notes: [docs/releases/v0.4.0.md](docs/releases/v0.4.0.md)

## 📚 Docs Map

- **Start Here**
- 🏁 **Docs index**: [docs/README.md](docs/README.md)
- 🚀 **Getting started (Docker)**: [docs/getting-started/docker.md](docs/getting-started/docker.md)
- 💻 **Getting started (local)**: [docs/getting-started/local-development.md](docs/getting-started/local-development.md)
- 🧰 **Troubleshooting**: [docs/getting-started/troubleshooting.md](docs/getting-started/troubleshooting.md)
- ✅ **Smoke test script**: [docs/getting-started/smoke-test.md](docs/getting-started/smoke-test.md)
- 🧪 **CLI commands reference**: [docs/cli/commands.md](docs/cli/commands.md)
- 🚀 **v0.4.0 release notes**: [docs/releases/v0.4.0.md](docs/releases/v0.4.0.md)

- **By Topic**
- ⚙️ **Environment variables**: [docs/configuration/environment-variables.md](docs/configuration/environment-variables.md)
- 🏗️ **Architecture concepts**: [docs/concepts/architecture.md](docs/concepts/architecture.md)
- 🔒 **Security model**: [docs/concepts/security-model.md](docs/concepts/security-model.md)
- 📘 **Glossary**: [docs/concepts/glossary.md](docs/concepts/glossary.md)
- 🔑 **Key management operations**: [docs/operations/key-management.md](docs/operations/key-management.md)
- 📊 **Monitoring and metrics**: [docs/operations/monitoring.md](docs/operations/monitoring.md)
- 🚑 **Failure playbooks**: [docs/operations/failure-playbooks.md](docs/operations/failure-playbooks.md)
- 🏭 **Production deployment**: [docs/operations/production.md](docs/operations/production.md)
- 🛠️ **Development and testing**: [docs/development/testing.md](docs/development/testing.md)
- 🤝 **Docs contributing**: [docs/contributing.md](docs/contributing.md)
- 🗒️ **Docs changelog**: [docs/CHANGELOG.md](docs/CHANGELOG.md)

- **API Reference**
- 🔐 **Auth API**: [docs/api/authentication.md](docs/api/authentication.md)
- 👤 **Clients API**: [docs/api/clients.md](docs/api/clients.md)
- 📘 **Policy cookbook**: [docs/api/policies.md](docs/api/policies.md)
- 🗂️ **Capability matrix**: [docs/api/capability-matrix.md](docs/api/capability-matrix.md)
- 📦 **Secrets API**: [docs/api/secrets.md](docs/api/secrets.md)
- 🚄 **Transit API**: [docs/api/transit.md](docs/api/transit.md)
- 🎫 **Tokenization API**: [docs/api/tokenization.md](docs/api/tokenization.md)
- 📜 **Audit logs API**: [docs/api/audit-logs.md](docs/api/audit-logs.md)
- 🧩 **API versioning policy**: [docs/api/versioning-policy.md](docs/api/versioning-policy.md)

- **Examples**
- 🧪 **Curl examples**: [docs/examples/curl.md](docs/examples/curl.md)
- 🐍 **Python examples**: [docs/examples/python.md](docs/examples/python.md)
- 🟨 **JavaScript examples**: [docs/examples/javascript.md](docs/examples/javascript.md)
- 🐹 **Go examples**: [docs/examples/go.md](docs/examples/go.md)

All detailed guides include practical use cases and copy/paste-ready examples.

## ✨ What You Get

- 🔐 Envelope encryption (`Master Key -> KEK -> DEK -> Secret Data`)
- 🚄 Transit encryption (`/v1/transit/keys/*`) for encrypt/decrypt as a service (decrypt input uses `<version>:<base64-ciphertext>`; see [Transit API docs](docs/api/transit.md), [create vs rotate](docs/api/transit.md#create-vs-rotate), and [error matrix](docs/api/transit.md#endpoint-error-matrix))
- 🎫 Tokenization API (`/v1/tokenization/*`) for token generation, detokenization, validation, and revocation
- 👤 Token-based authentication and policy-based authorization
- 📦 Versioned secrets by path (`/v1/secrets/*path`)
- 📜 Audit logs with request correlation (`request_id`) and filtering
- 📊 OpenTelemetry metrics with Prometheus-compatible `/metrics` export

## 🌐 API Overview

- Health: `GET /health`
- Readiness: `GET /ready`
- Token issuance: `POST /v1/token`
- Clients: `GET/POST /v1/clients`, `GET/PUT/DELETE /v1/clients/:id`
- Secrets: `POST/GET/DELETE /v1/secrets/*path`
- Transit: `POST /v1/transit/keys`, `POST /v1/transit/keys/:name/rotate`, `POST /v1/transit/keys/:name/encrypt`, `POST /v1/transit/keys/:name/decrypt`, `DELETE /v1/transit/keys/:id` ([create vs rotate](docs/api/transit.md#create-vs-rotate), [error matrix](docs/api/transit.md#endpoint-error-matrix))
- Tokenization: `POST /v1/tokenization/keys`, `POST /v1/tokenization/keys/:name/rotate`, `DELETE /v1/tokenization/keys/:id`, `POST /v1/tokenization/keys/:name/tokenize`, `POST /v1/tokenization/detokenize`, `POST /v1/tokenization/validate`, `POST /v1/tokenization/revoke`
- Audit logs: `GET /v1/audit-logs`
- Metrics: `GET /metrics` (available when `METRICS_ENABLED=true`)

## 📄 License

MIT. See `LICENSE`.

## See also

- [Documentation index](docs/README.md)
- [Docker getting started](docs/getting-started/docker.md)
- [API authentication](docs/api/authentication.md)
- [Production operations](docs/operations/production.md)
