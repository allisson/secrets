# 🐳 Run with Docker (Recommended)

> Last updated: 2026-02-14

This is the default way to run Secrets.

## ⚡ Quickstart Copy Block

Use this minimal flow when you just want to get a working instance quickly:

```bash
docker pull allisson/secrets:latest
docker network create secrets-net || true

docker run -d --name secrets-postgres --network secrets-net \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=mydb \
  postgres:16-alpine

docker run --rm allisson/secrets:latest create-master-key --id default
# copy generated MASTER_KEYS and ACTIVE_MASTER_KEY_ID into .env

docker run --rm --network secrets-net --env-file .env allisson/secrets:latest migrate
docker run --rm --network secrets-net --env-file .env allisson/secrets:latest create-kek --algorithm aes-gcm
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:latest server
```

## 1) Pull the image

```bash
docker pull allisson/secrets:latest
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
docker run --rm allisson/secrets:latest create-master-key --id default
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

AUTH_TOKEN_EXPIRATION_SECONDS=86400
EOF
```

## 5) Run migrations and bootstrap KEK

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets:latest migrate
docker run --rm --network secrets-net --env-file .env allisson/secrets:latest create-kek --algorithm aes-gcm
```

## 6) Start the API server

```bash
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:latest server
```

## 7) Verify

```bash
curl http://localhost:8080/health
```

Expected:

```json
{"status":"healthy"}
```

## 8) First token + first secret

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
