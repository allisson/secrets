# 💻 Run Locally (Development)

> Last updated: 2026-02-24

Use this path if you want to modify the source code and run from your workstation.

**⚠️ Security Warning:** This guide is for **development and testing only**. For production deployments, see [Security Hardening Guide](../operations/security/hardening.md) and [Production Deployment Guide](../operations/deployment/production.md).

## Current Security Defaults

- `AUTH_TOKEN_EXPIRATION_SECONDS` default is `14400` (4 hours)
- `RATE_LIMIT_ENABLED` default is `true` (per authenticated client)
- `RATE_LIMIT_TOKEN_ENABLED` default is `true` (per IP on `POST /v1/token`)
- `CORS_ENABLED` default is `false`

These defaults were introduced in `v0.5.0` with token-endpoint rate limiting added in `v0.7.0` (current: v0.12.0).

If upgrading from `v0.6.0`, review [v0.7.0 upgrade guide](../releases/RELEASES.md#070---2026-02-20).

## Prerequisites

- Go 1.25+
- Docker (for local database)

## 1) Clone and install dependencies

```bash
git clone https://github.com/allisson/secrets.git
cd secrets
go mod download
```

## 2) Build

```bash
make build
```

## 3) Generate master key and set `.env`

```bash
./bin/app create-master-key --id default
cp .env.example .env
```

Paste generated `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID` into `.env`.

For production-oriented local parity testing, use KMS mode:

```bash
./bin/app create-master-key --id default --kms-provider=localsecrets --kms-key-uri="base64key://<base64-32-byte-key>"
```

## 4) Start PostgreSQL

```bash
make dev-postgres
```

Default connection in `.env` can be:

```dotenv
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable
```

## 5) Migrate and create KEK

```bash
./bin/app migrate
./bin/app create-kek --algorithm aes-gcm
```

## 6) Start server

```bash
./bin/app server
```

## 7) Create first client credentials

In another terminal, create your first API client and policy set:

```bash
./bin/app create-client \
  --name bootstrap-admin \
  --active \
  --policies '[{"path":"*","capabilities":["read","write","delete","encrypt","decrypt","rotate"]}]' \
  --format json
```

Save the returned `client_id` and one-time `secret` securely.

## 8) Issue token

```bash
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
```

## 9) Smoke test

```bash
curl http://localhost:8080/health
```

## See also

- [Docker getting started](docker.md)
- [Smoke test](smoke-test.md)
- [Troubleshooting](troubleshooting.md)
- [Development and testing](../contributing.md#development-and-testing)
- [CLI commands reference](../cli-commands.md)
