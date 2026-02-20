# 🐳 Run with Docker (Recommended)

> Last updated: 2026-02-20

This is the default way to run Secrets.

For release reproducibility, this guide uses the pinned image tag `allisson/secrets:v0.7.0`.
For dev-only fast iteration, you can use `allisson/secrets:latest`.

**⚠️ Security Warning:** This guide is for **development and testing only**. For production deployments, see [Security Hardening Guide](../operations/security-hardening.md) and [Production Deployment Guide](../operations/production.md).

## Current Security Defaults

- `AUTH_TOKEN_EXPIRATION_SECONDS` default is `14400` (4 hours)
- `RATE_LIMIT_ENABLED` default is `true` (per authenticated client)
- `RATE_LIMIT_TOKEN_ENABLED` default is `true` (per IP on `POST /v1/token`)
- `CORS_ENABLED` default is `false`

These defaults were introduced in `v0.5.0` and now include token-endpoint rate limiting in `v0.7.0`.

If upgrading from `v0.6.0`, review [v0.7.0 upgrade guide](../releases/v0.7.0-upgrade.md).

## ⚡ Quickstart Copy Block

Use this minimal flow when you just want to get a working instance quickly:

```bash
docker pull allisson/secrets:v0.7.0
docker network create secrets-net || true

docker run -d --name secrets-postgres --network secrets-net \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=mydb \
  postgres:16-alpine

docker run --rm allisson/secrets:v0.7.0 create-master-key --id default
# copy generated MASTER_KEYS and ACTIVE_MASTER_KEY_ID into .env

docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.7.0 migrate
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.7.0 create-kek --algorithm aes-gcm
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:v0.7.0 server
```

## 1) Pull the image

```bash
docker pull allisson/secrets:v0.7.0
```

## 2) Start PostgreSQL

```bash
docker network create secrets-net

docker run -d --name secrets-postgres --network secrets-net \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=mydb \
  postgres:16-alpine
```

## 3) Generate a master key

```bash
docker run --rm allisson/secrets:v0.7.0 create-master-key --id default
```

Copy the generated values into a local `.env` file.

## 4) Create `.env`

```bash
cat > .env <<'EOF'
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:password@secrets-postgres:5432/mydb?sslmode=disable
DB_MAX_OPEN_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONN_MAX_LIFETIME=5

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
LOG_LEVEL=info

MASTER_KEYS=default:REPLACE_WITH_BASE64_32_BYTE_KEY
ACTIVE_MASTER_KEY_ID=default

AUTH_TOKEN_EXPIRATION_SECONDS=14400

RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20
RATE_LIMIT_TOKEN_ENABLED=true
RATE_LIMIT_TOKEN_REQUESTS_PER_SEC=5.0
RATE_LIMIT_TOKEN_BURST=10

METRICS_ENABLED=true
METRICS_NAMESPACE=secrets
EOF
```

## 5) Run migrations and bootstrap KEK

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.7.0 migrate
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.7.0 create-kek --algorithm aes-gcm
```

## 6) Start the API server

```bash
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:v0.7.0 server
```

## 7) Verify

```bash
curl http://localhost:8080/health
```

Expected:

```json
{"status":"healthy"}
```

## 8) Create first client credentials

Use the CLI command to create your first API client and policy set:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.7.0 create-client \
  --name bootstrap-admin \
  --active \
  --policies '[{"path":"*","capabilities":["read","write","delete","encrypt","decrypt","rotate"]}]' \
  --format json
```

Save the returned `client_id` and one-time `secret` securely. The secret is shown only once.

## 9) First token + first secret

```bash
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
```

Then:

```bash
curl -X POST http://localhost:8080/v1/secrets/app/prod/db-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value":"c3VwZXItc2VjcmV0"}'
```

`value` is base64-encoded plaintext (`super-secret`).

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

For a full end-to-end check, run `docs/getting-started/smoke-test.sh` (usage in `docs/getting-started/smoke-test.md`).

## See also

- [Local development](local-development.md)
- [Smoke test](smoke-test.md)
- [Troubleshooting](troubleshooting.md)
- [Environment variables](../configuration/environment-variables.md)
- [CLI commands reference](../cli/commands.md)
