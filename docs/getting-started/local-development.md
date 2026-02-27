# 💻 Run Locally (Development)

> Last updated: 2026-02-26

Use this path if you want to modify the source code and run from your workstation.

**⚠️ Security Warning:** This guide is for **development and testing only**. For production deployments, see [Security Hardening Guide](../operations/deployment/docker-hardened.md) and [Production Deployment Guide](../operations/deployment/docker-hardened.md).

## Current Security Defaults

- `AUTH_TOKEN_EXPIRATION_SECONDS` default is `14400` (4 hours)
- `RATE_LIMIT_ENABLED` default is `true` (per authenticated client)
- `RATE_LIMIT_TOKEN_ENABLED` default is `true` (per IP on `POST /v1/token`)
- `CORS_ENABLED` default is `false`

These defaults were introduced in `v0.5.0` with token-endpoint rate limiting added in `v0.7.0` (see `docs/metadata.json` for latest).

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

KMS mode is required as of v0.19.0. For local development, use the `localsecrets` provider:

```bash
# Generate a KMS encryption key (32 random bytes, base64-encoded)
KMS_KEY=$(openssl rand -base64 32)

# Create master key with KMS encryption
./bin/app create-master-key --id default \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://${KMS_KEY}"

# Copy example environment file
cp .env.example .env
```

The command output will include:

- `KMS_PROVIDER` and `KMS_KEY_URI` (already set if you used the command above)
- `MASTER_KEYS` - paste this into your `.env` file
- `ACTIVE_MASTER_KEY_ID` - paste this into your `.env` file

Your `.env` file should look like:

```dotenv
KMS_PROVIDER=localsecrets
KMS_KEY_URI=base64key://<generated-key>
MASTER_KEYS=default:<kms-encrypted-value>
ACTIVE_MASTER_KEY_ID=default
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
- [Troubleshooting](../operations/troubleshooting/index.md)
- [Development and testing](../contributing.md#development-and-testing)
- [CLI commands reference](../cli-commands.md)
