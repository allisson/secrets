# 🐳 Run with Docker (Recommended)

> Last updated: 2026-02-21

This is the default way to run Secrets.

This guide uses the latest Docker image (`allisson/secrets`).

**⚠️ Security Warning:** This guide is for **development and testing only**. For production deployments, see [Security Hardening Guide](../operations/security/hardening.md) and [Production Deployment Guide](../operations/deployment/production.md).

## Current Security Defaults

- `AUTH_TOKEN_EXPIRATION_SECONDS` default is `14400` (4 hours)

- `RATE_LIMIT_ENABLED` default is `true` (per authenticated client)

- `RATE_LIMIT_TOKEN_ENABLED` default is `true` (per IP on `POST /v1/token`)

- `CORS_ENABLED` default is `false`

These defaults were introduced in `v0.5.0` with token-endpoint rate limiting added in `v0.7.0` (current: v0.10.0).

If upgrading from `v0.6.0`, review [v0.7.0 upgrade guide](../releases/RELEASES.md#070---2026-02-20).

## 🔒 Security Features (v0.10.0+)

The Docker image uses security-hardened configuration:

- **Distroless base image**: Google's `gcr.io/distroless/static-debian13` (pinned by SHA256)

  - No shell, package manager, or system utilities (minimal attack surface)

  - Regular security patches from Google Distroless team

  - Better CVE scanning support vs. `scratch` base  

    📖 For vulnerability scanning instructions, see [Security Scanning Guide](../operations/security/scanning.md)

- **Non-root user**: Runs as UID 65532 (`nonroot:nonroot`)

- **Static binary**: No libc dependencies, compiled with `CGO_ENABLED=0`

- **Read-only filesystem**: Can run with `--read-only` flag (no runtime writes)

- **Image pinning**: SHA256 digest pinning for immutability

- **Multi-architecture**: Native support for `linux/amd64` and `linux/arm64`  

  📖 For detailed multi-arch build instructions, see [Multi-Architecture Build Guide](../operations/deployment/multi-arch-builds.md)

- **Build metadata**: OCI labels with version, commit SHA, and build timestamp

### Health Check Endpoints

The API exposes two health endpoints for container orchestration:

- **`GET /health`**: Liveness probe (basic health check, < 10ms)

- **`GET /ready`**: Readiness probe (includes database connectivity check, < 100ms)

**Quick example**:

```bash
# Test liveness
curl http://localhost:8080/health
# Response: {"status":"healthy"}

# Test readiness
curl http://localhost:8080/ready
# Response: {"status":"ready","database":"ok"}

```

**For complete health check documentation**, including platform-specific configurations (Docker Compose, AWS ECS, Google Cloud Run), monitoring integration, and troubleshooting, see:

📖 **[Health Check Endpoints Guide](../operations/observability/health-checks.md)**

**Quick reference for common platforms**:

- **Docker Compose**: Use healthcheck sidecar (distroless has no shell)

- **AWS ECS**: Use ALB target group health checks with `/ready`

- **Google Cloud Run**: Configure startup and liveness probes with `/health` and `/ready`

- **Prometheus**: Use Blackbox Exporter to monitor endpoints

**Read-only filesystem example:**

```bash
docker run --rm --name secrets-api \
  --network secrets-net \
  --env-file .env \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=10m \
  -p 8080:8080 \
  allisson/secrets server

```

> **Note**: The `--tmpfs /tmp` volume is **optional** because the application doesn't write to the filesystem at runtime (embedded migrations, stateless binary). However, it's recommended for security hardening to support potential temporary file operations.

For comprehensive container security guidance, see [Container Security Guide](../operations/security/container-security.md).

For production security hardening, see [Security Hardening Guide](../operations/security/hardening.md).

## ⚡ Quickstart Copy Block

Use this minimal flow when you just want to get a working instance quickly:

```bash
docker pull allisson/secrets
docker network create secrets-net || true

docker run -d --name secrets-postgres --network secrets-net \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=mydb \
  postgres:16-alpine

docker run --rm allisson/secrets create-master-key --id default
# copy generated MASTER_KEYS and ACTIVE_MASTER_KEY_ID into .env

docker run --rm --network secrets-net --env-file .env allisson/secrets migrate
docker run --rm --network secrets-net --env-file .env allisson/secrets create-kek --algorithm aes-gcm
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets server

```

## 1) Pull the image

```bash
docker pull allisson/secrets

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
docker run --rm allisson/secrets create-master-key --id default

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
docker run --rm --network secrets-net --env-file .env allisson/secrets migrate
docker run --rm --network secrets-net --env-file .env allisson/secrets create-kek --algorithm aes-gcm

```

## 6) Start the API server

```bash
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets server

```

## 7) Verify

Check the liveness endpoint:

```bash
curl http://localhost:8080/health

```

Expected:

```json
{"status":"healthy"}

```

Check the readiness endpoint (includes database connectivity):

```bash
curl http://localhost:8080/ready

```

Expected (if database is connected):

```json
{"status":"ready"}

```

## 8) Create first client credentials

Use the CLI command to create your first API client and policy set:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets create-client \
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

## Common Issues (v0.10.0+)

### Volume Permission Errors

If you encounter permission errors with mounted volumes after upgrading to v0.10.0, this is due to the non-root user (UID 65532) introduced for security.

**Symptoms**:

- Container fails to start with "permission denied" errors

- Application cannot write to mounted directories

- Logs show "EACCES" or "operation not permitted"

**Quick fix** (Docker):

```bash
# Change host directory ownership to UID 65532
sudo chown -R 65532:65532 /path/to/host/directory

```

**For comprehensive solutions** (Docker Compose, named volumes), see:

- [Volume Permission Troubleshooting Guide](../operations/troubleshooting/volume-permissions.md)

### Health Check Configuration

For health check examples (Docker Compose sidecar, external monitoring), see the "Security Features" section above.

## See also

- [Local development](local-development.md)

- [Smoke test](smoke-test.md)

- [Troubleshooting](troubleshooting.md)

- [Environment variables](../configuration.md)

- [CLI commands reference](../cli-commands.md)

- [Docker Compose Examples](../operations/deployment/docker-compose.md) - Complete Docker Compose setup (coming soon)
