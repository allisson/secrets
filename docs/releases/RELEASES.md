# 🚀 Release Notes

> Last updated: 2026-02-25

This document contains release notes and upgrade guides for all versions of Secrets.

For the compatibility matrix across versions, see [compatibility-matrix.md](compatibility-matrix.md).

## 📑 Quick Navigation

**Latest Release**: [v0.14.0](#0140---2026-02-25)

**All Releases**:

- [v0.14.0 (2026-02-25)](#0140---2026-02-25) - Dedicated metrics server

- [v0.13.0 (2026-02-25)](#0130---2026-02-25) - Documentation conciseness refactor

- [v0.12.0 (2026-02-24)](#0120---2026-02-24) - Rewrap DEKs

- [v0.11.0 (2026-02-23)](#0110---2026-02-23) - Account lockout

- [v0.10.0 (2026-02-21)](#0100---2026-02-21) - Docker security improvements

- [v0.9.0 (2026-02-20)](#090---2026-02-20) - Cryptographic audit log signing

- [v0.8.0 (2026-02-20)](#080---2026-02-20) - Documentation consolidation and ADR establishment

- [v0.7.0 (2026-02-20)](#070---2026-02-20) - IP-based rate limiting for token endpoint

- [v0.6.0 (2026-02-19)](#060---2026-02-19) - KMS provider support

- [v0.5.1 (2026-02-19)](#051---2026-02-19) - Audit log cleanup command

- [v0.5.0 (2026-02-19)](#050---2026-02-19) - Tokenization and CORS

- [v0.4.1 (2026-02-19)](#041---2026-02-19) - Pagination bug fix

- [v0.4.0 (2026-02-18)](#040---2026-02-18) - Audit logging

- [v0.3.0 (2026-02-16)](#030---2026-02-16) - Client management

- [v0.2.0 (2026-02-14)](#020---2026-02-14) - Transit encryption

- [v0.1.0 (2026-02-14)](#010---2026-02-14) - Initial release

---

## [0.14.0] - 2026-02-25

### Dedicated Metrics Server

This release separates the Prometheus `/metrics` endpoint from the main API server. It now runs on a dedicated port (default `8081`), improving security by allowing operators to easily block public access to metrics at the network level.

### Added

- `METRICS_PORT` environment variable (default `8081`) to configure the dedicated metrics server port
- Built-in metrics server executing in a separate goroutine

### Changed

- The Prometheus `/metrics` endpoint is no longer exposed on the main API port (`8080`)
- Updated deployment examples (Docker Compose, etc.) and documentation to reflect the new `8081` metrics port

### Security

- Reduces the risk of exposing internal application metrics to the public internet by decoupling it from the main API port

---

## [0.13.0] - 2026-02-25

### Documentation Conciseness Refactor

This release significantly reduces documentation bloat by centralizing code examples, merging redundant container operations/security guides, and moving release-specific migrations out of the main changelog.

### Added

- A centralized `upgrades.md` runbook for universal upgrades and rollbacks
- An `examples/` directory for full repository examples (like `docker-compose`)
- A single, punchy `docker-hardened.md` guide for production container deployments

### Changed

- Stripped `RELEASES.md` into a pure changelog without embedded tutorials
- Restructured `troubleshooting` into a centralized `index.md` file
- Consolidated operations, security, and deployment directories to reduce text repetition

### Removed

- Deleted the repetitive `container-security.md`, `hardening.md`, and `production.md` references (merged)
- Archived the point-in-time `base-image-migration.md` guide

---

## [0.12.0] - 2026-02-24

### Rewrap DEKs

This release introduces the `rewrap-deks` CLI command, enabling operators to bulk re-encrypt existing Data Encryption Keys (DEKs) that aren't currently secured with a specific Key Encryption Key (KEK). This allows complete key transitions after a KEK rotation.

### Notes

- Version bump and documentation updates preparing for v0.12.0 release

---

## [0.11.0] - 2026-02-23

### Account Lockout

This release adds persistent account lockout to prevent brute-force attacks against the token endpoint.

### Added

- Account lockout: clients are locked for 30 minutes after 10 consecutive failed authentication attempts

- `LOCKOUT_MAX_ATTEMPTS` environment variable (default `10`) — configures the failure threshold

- `LOCKOUT_DURATION_MINUTES` environment variable (default `30`) — configures the lockout duration in minutes

- `423 Locked` HTTP response on `POST /v1/token` when a client is locked

- Database migration `000004_add_account_lockout` — adds `failed_attempts` and `locked_until` columns to the `clients` table

### Runtime Changes

- New environment variables:

  - `LOCKOUT_MAX_ATTEMPTS` (default `10`)

  - `LOCKOUT_DURATION_MINUTES` (default `30`)

- `POST /v1/token` may now return `423 Locked` when a client account is locked due to too many failed attempts

- Failed attempt counter and lock expiry are reset automatically on successful authentication

- Lock check is based on `locked_until` timestamp — expired locks are treated as unlocked

### Security and Operations Impact

- Complements the existing IP-based rate limiting on `POST /v1/token` — lockout is per-client identity, not per-IP

- Operators can manually unlock a client by setting `locked_until = NULL` and `failed_attempts = 0` in the database

---

## [0.10.0] - 2026-02-21

### 🐳 Docker Security Improvements

This release focuses on comprehensive Docker security enhancements, migrating to Google Distroless base images with SHA256 digest pinning for immutable builds.

### Added

- Docker image security improvements with Google Distroless base (Debian 13 Trixie)

- SHA256 digest pinning for immutable container builds  

- Build-time version injection via ldflags (version, buildDate, commitSHA)

- Comprehensive OCI labels for better security scanning and SBOM generation

- Multi-architecture build support (linux/amd64, linux/arm64) in Dockerfile

- `.dockerignore` file to reduce build context size by ~90%

- Explicit non-root user execution (UID 65532: nonroot:nonroot)

- Read-only filesystem support for enhanced runtime security

- Container security documentation: `docs/operations/deployment/docker-hardened.md`

- Health check endpoint documentation for Docker Compose

- GitHub Actions workflow enhancements for build metadata injection

- Version management guidelines in AGENTS.md for coding agents

### Changed

- Base builder image: `golang:1.25.5-alpine` → `golang:1.25.5-trixie` (Debian 13)

- Final runtime image: `scratch` → `gcr.io/distroless/static-debian13@sha256:d90359c7...`

- Application version management: hardcoded → build-time injection

- Docker image now includes default `CMD ["server"]` for better UX

- Updated `docs/getting-started/docker.md` with security features and health check examples

### Removed

- Manual migration directory copy (now embedded in binary via Go embed.FS)

- Manual CA certificates and timezone data copy (included in distroless)

### Security

- **BREAKING**: Container now runs as non-root user (UID 65532) by default

- Minimal attack surface: no shell, package manager, or system utilities in final image

- Regular security patches from Google Distroless project

- Immutable builds with SHA256 digest pinning prevent supply chain attacks

- Enhanced CVE scanning support with comprehensive OCI metadata

- Image size reduced by 10-20% while improving security posture

### Documentation

- Added comprehensive container security guide with 10 sections

- Updated Docker quick start guide with security features overview

- Added health check endpoint documentation for orchestration platforms

- Added version management guidelines for AI coding agents

### Migration Guide

⚠️ **BREAKING CHANGE**: v0.10.0 introduces non-root user (UID 65532) which may cause volume permission issues.

**For teams migrating from custom Docker images** (Alpine, scratch, Debian), see the comprehensive [Base Image Migration Guide](../operations/deployment/docker-hardened.md).

#### Pre-Migration Checklist

Complete these steps before upgrading:

- [ ] **Backup database** (test restore in staging environment)

- [ ] **Review breaking changes** (see "Security" section above)

- [ ] **Test in staging** (verify volume permissions and health checks work)

- [ ] **Plan rollback window** (see "Rollback Procedures" below)

- [ ] **Update monitoring** (adjust alerts for potential startup delays)

- [ ] **Review volume mounts** (identify host directories that need permission fixes)

#### Docker Migration

*### Step 1: Update image reference**

```bash
# Pull new version
docker pull allisson/secrets:v0.12.0

# Verify version and metadata
docker run --rm allisson/secrets:v0.12.0 --version
# Version:    v0.10.0
# Build Date: 2026-02-21T...
# Commit SHA: ...

```

**Step 2: Fix volume permissions** (if using host bind mounts)

```bash
# Option A: Change host directory ownership
sudo chown -R 65532:65532 /path/to/data

# Option B: Use named volumes (recommended for production)
docker volume create secrets-data
# Then use -v secrets-data:/data in docker run

```

*### Step 3: Test health checks**

```bash
# Run test container
docker run -d --name secrets-test \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v0.12.0 server

# Wait for startup
sleep 5

# Verify health endpoints
curl http://localhost:8080/health  # Should return 200 OK
curl http://localhost:8080/ready   # Should return 200 OK

# Cleanup
docker rm -f secrets-test

```

*### Step 4: Update production**

```bash
# Stop old container
docker stop secrets-api
docker rm secrets-api

# Start new container with volume fix
docker run -d --name secrets-api \
  --env-file .env \
  -p 8080:8080 \
  -v secrets-data:/data \
  allisson/secrets:v0.12.0 server

# Verify startup
docker logs -f secrets-api

```

#### Docker Compose Migration

**Full production-ready example** with healthcheck sidecar and named volumes:

```yaml
version: '3.8'

services:
  secrets-api:
    image: allisson/secrets:v0.12.0
    env_file: .env
    ports:
      - "8080:8080"

    volumes:
      # Use named volume (Docker handles permissions automatically)
      - secrets-data:/data

    restart: unless-stopped
    networks:
      - secrets-net

  # Healthcheck sidecar (distroless has no curl/wget)
  healthcheck:
    image: curlimages/curl:latest
    command: >
      sh -c 'while true; do
        curl -f http://secrets-api:8080/health || exit 1;
        sleep 30;
      done'
    depends_on:
      - secrets-api

    restart: unless-stopped
    networks:
      - secrets-net

  # PostgreSQL database
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: secrets
      POSTGRES_PASSWORD: secrets
      POSTGRES_DB: secrets
    volumes:
      - postgres-data:/var/lib/postgresql/data

    restart: unless-stopped
    networks:
      - secrets-net

volumes:
  secrets-data:
    driver: local
  postgres-data:
    driver: local

networks:
  secrets-net:
    driver: bridge

```

**Migration steps**:

```bash
# 1. Update docker-compose.yml with example above

# 2. Pull new images
docker-compose pull

# 3. Stop old containers
docker-compose down

# 4. Start with new version
docker-compose up -d

# 5. Verify health
curl http://localhost:8080/health
docker-compose logs -f secrets-api

```

#### Rollback Procedures

If issues occur during or after migration, rollback to v0.9.0:

**Docker**:

```bash
# 1. Stop v0.10.0 container
docker stop secrets-api
docker rm secrets-api

# 2. Revert volume permissions (if you changed them)
sudo chown -R root:root /path/to/host/data
# OR use the user/group that owned them before

# 3. Start v0.9.0 container
docker run -d --name secrets-api \
  --env-file .env \
  -p 8080:8080 \
  -v /path/to/host/data:/data \
  allisson/secrets:v0.9.0 server

# 4. Verify health
curl http://localhost:8080/health
docker logs -f secrets-api

```

**Docker Compose**:

```bash
# 1. Update image in docker-compose.yml
#    Change: image: allisson/secrets:v0.12.0
#    To:     image: allisson/secrets:v0.9.0

# 2. Restart services
docker-compose down
docker-compose up -d

# 3. Verify
curl http://localhost:8080/health
docker-compose logs -f secrets-api

```

**Database compatibility**: v0.10.0 has **no database schema changes** from v0.9.0. You can rollback without reverting migrations.

**Volume permissions note**: If you changed host directory ownership to UID 65532, revert it after rollback (v0.9.0 runs as root and expects root-owned files).

#### Post-Migration Validation

After migration, verify everything works:

**Application health**:

- [ ] `GET /health` returns 200 OK

- [ ] `GET /ready` returns 200 OK

- [ ] No permission errors in logs

- [ ] Container stays running (not crash-looping)

**Functional tests**:

- [ ] Can authenticate and get token (`POST /v1/token`)

- [ ] Can create secrets (`POST /v1/secrets/...`)

- [ ] Can retrieve secrets (`GET /v1/secrets/...`)

- [ ] Can create transit keys (`POST /v1/transit/keys`)

- [ ] Can encrypt/decrypt with transit (`POST /v1/transit/encrypt/...`)

- [ ] Audit logs are created successfully

**Operational checks**:

- [ ] Metrics are being exported (if enabled)

- [ ] Logs are being forwarded to aggregator

- [ ] Health checks passing in load balancer/orchestrator

- [ ] No increase in error rates (monitor for 15-30 minutes)

**Security validation**:

- [ ] Container runs as UID 65532 (not root): `docker exec secrets-api id`

- [ ] Read-only filesystem works: `docker run --rm --read-only --tmpfs /tmp allisson/secrets:v0.12.0 --version`

- [ ] No privilege escalation: Verify container security settings

#### Rollback Testing (Pre-Production Required)

**⚠️ CRITICAL**: Test rollback procedures in staging BEFORE production deployment.

**Test procedure** (15-30 minutes):

```bash
# 1. Deploy v0.10.0 to staging (Docker Compose example)
docker-compose pull
docker-compose up -d

# 2. Create test data
TOKEN=$(curl -X POST http://staging:8080/v1/token \
  -d '{"client_id":"test","client_secret":"test"}' | jq -r '.token')

curl -X POST http://staging:8080/v1/secrets/test/rollback \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"dGVzdA=="}'

# 3. Note secret version and timestamp
curl http://staging:8080/v1/secrets/test/rollback \
  -H "Authorization: Bearer $TOKEN"

# 4. Simulate failure and rollback
# Update docker-compose.yml to use v0.9.0
docker-compose down
docker-compose up -d

# 5. Verify data integrity after rollback
curl http://staging:8080/v1/secrets/test/rollback \
  -H "Authorization: Bearer $TOKEN"
# Should return same data

# 6. Deploy v0.10.0 again (forward migration)
# Update docker-compose.yml back to v0.10.0
docker-compose down
docker-compose up -d

# 7. Verify data still accessible
curl http://staging:8080/v1/secrets/test/rollback \
  -H "Authorization: Bearer $TOKEN"
# Should still return same data

# 8. Document rollback time
# Measure time from "docker-compose down" to "curl succeeds"

```

**Expected rollback time**: 1-3 minutes (depends on container restart time and health check settings)

**Document results**:

- Rollback duration: _____ seconds

- Data integrity: PASS / FAIL

- Issues encountered: _____

- Mitigation required: _____

#### Troubleshooting Migration Issues

*### Issue: Health checks failing after upgrade**

```bash
# Check logs for errors
docker logs secrets-api

# Common causes:
# - Database connection failed (check DB_CONNECTION_STRING)

# - Port 8080 not accessible (check firewall/network policy)

# - Volume permission errors (see above)

```

---

*### Issue: Container won't start**

```bash
# Check container logs
docker logs secrets-api

# Check if running as correct user
docker run --rm allisson/secrets:v0.12.0 id
# Should show: uid=65532(nonroot)

# Test without volumes to isolate issue
docker run --rm --env-file .env allisson/secrets:v0.12.0 server

```

#### Additional Resources

- [Container Security Guide](../operations/deployment/docker-hardened.md) (security best practices)

- [Production Rollout Guide](../operations/deployment/production-rollout.md) (deployment checklist)

- [Docker Quick Start](../getting-started/docker.md) (getting started)

---

## [0.9.0] - 2026-02-20

### Highlights

- Added cryptographic audit log signing with HMAC-SHA256 for tamper detection

- Added `verify-audit-logs` CLI command for integrity verification with text/JSON output

- Added HKDF-SHA256 key derivation to separate encryption and signing key usage

- Added database migration 000003 with signature columns and FK constraints

- Enhanced audit log integrity with automatic signing on creation

### Runtime Changes

- **Database migration required** (000003) - adds `signature`, `kek_id`, `is_signed` columns

- **Foreign key constraints added:**

  - `fk_audit_logs_client_id` - prevents client deletion with audit logs

  - `fk_audit_logs_kek_id` - prevents KEK deletion with audit logs

- Audit log API responses now include signature metadata

- New CLI command: `verify-audit-logs --start-date <YYYY-MM-DD> --end-date <YYYY-MM-DD> [--format text|json]`

- Existing audit logs marked as legacy (`is_signed=false`) after migration

### Security and Operations Impact

- **Breaking Change:** Foreign key constraints prevent deletion of clients/KEKs with associated audit logs

- Enables cryptographic verification of audit log integrity and tamper detection

- Legacy unsigned logs remain queryable but cannot be cryptographically verified

---

## [0.8.0] - 2026-02-20

### Highlights

- Documentation consolidation: reduced from 77 to 47 markdown files (39% reduction)

- Established 8 new Architecture Decision Records (ADR 0003-0010) covering key architectural decisions

- Restructured API documentation with themed subdirectories (auth/, data/, observability/)

- Consolidated operations documentation with centralized runbook hub

- Merged all development documentation into contributing.md

- Comprehensive cross-reference updates throughout documentation (182+ updates)

### Runtime Changes

None - this is a documentation-only release.

---

## [0.7.0] - 2026-02-20

### Highlights

- Added IP-based rate limiting for `POST /v1/token`

- Added token endpoint rate-limit configuration via `RATE_LIMIT_TOKEN_*` variables

- Added token endpoint `429 Too Many Requests` behavior with `Retry-After`

- Expanded docs and runbooks for token endpoint abuse protection and rollout validation

### Runtime Changes

- New environment variables:

  - `RATE_LIMIT_TOKEN_ENABLED` (default `true`)

  - `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC` (default `5.0`)

  - `RATE_LIMIT_TOKEN_BURST` (default `10`)

- `POST /v1/token` may now return `429 Too Many Requests` when per-IP token limits are exceeded

- Authenticated per-client rate limiting (`RATE_LIMIT_*`) remains unchanged

### Security and Operations Impact

- Improves protection against token endpoint credential stuffing and brute-force traffic

- Applies stricter defaults on unauthenticated token issuance than authenticated API routes

- Requires review of proxy/trusted-IP setup when using forwarded headers in production

---

## [0.6.0] - 2026-02-19

### Highlights

- Added KMS support for master key loading and decryption at startup

- Added CLI KMS flags to `create-master-key` (`--kms-provider`, `--kms-key-uri`)

- Added new `rotate-master-key` CLI command for staged master key rotation

- Added provider setup and migration runbook: [KMS setup guide](../operations/kms/setup.md)

### Runtime Changes

- New environment variables:

  - `KMS_PROVIDER`

  - `KMS_KEY_URI`

- Master key loading now supports two modes:

  - KMS mode: both variables set

  - Legacy mode: both variables unset

- Startup fails fast if only one KMS variable is set

### Security and Operations Impact

- KMS mode encrypts master keys at rest and centralizes key access control in your KMS provider

- Existing legacy environments remain supported without immediate migration

- Master key rotation now has an explicit CLI workflow for appending a new active key before cleanup

---

## [0.5.1] - 2026-02-19

### Highlights

- Fixed master key loading from environment variables to avoid zeroing the in-use key slice

- Hardened keychain shutdown by zeroing all master keys before clearing chain state

- Added regression tests for key usability after load and secure zeroing on close

### Fixes

- `LoadMasterKeyChainFromEnv` now stores a copy of decoded key bytes before zeroing temporary buffers

- `MasterKeyChain.Close` now zeros every loaded master key before clearing the key map

### Security Impact

- Reduces risk of leaked key material remaining in temporary decode buffers

- Ensures explicit in-memory zeroing of master keys during keychain teardown

### Runtime and Compatibility

- API baseline remains `v1` (`/v1/*`)

- No endpoint, payload, or status code contract changes

- No schema migrations required specifically for this patch release

---

## [0.5.0] - 2026-02-19

### Highlights

- Added per-client rate limiting for authenticated API routes

- Added configurable CORS middleware with secure defaults

- Reduced default token expiration from 24 hours to 4 hours

- Added comprehensive production security hardening guide

### Runtime Changes

- New rate limiting settings:

  - `RATE_LIMIT_ENABLED` (default `true`)

  - `RATE_LIMIT_REQUESTS_PER_SEC` (default `10.0`)

  - `RATE_LIMIT_BURST` (default `20`)

- New CORS settings:

  - `CORS_ENABLED` (default `false`)

  - `CORS_ALLOW_ORIGINS` (default empty)

- Authenticated endpoints now return `429 Too Many Requests` when limits are exceeded and include `Retry-After` response header

### Breaking / Behavior Changes

- **Default token expiration changed**:

  - Previous default: `AUTH_TOKEN_EXPIRATION_SECONDS=86400` (24h)

  - New default: `AUTH_TOKEN_EXPIRATION_SECONDS=14400` (4h)

If your clients expected 24-hour tokens, explicitly set `AUTH_TOKEN_EXPIRATION_SECONDS=86400` and verify refresh behavior.

---

## [0.4.1] - 2026-02-19

### Highlights

- Fixed authorization path matching for policies using mid-path wildcards

- Clarified wildcard matching semantics for exact, trailing wildcard, and segment wildcard paths

- Expanded automated coverage for policy templates, wildcard edge cases, and common policy mistakes

### Bug Fixes

- Policy matcher now supports mid-path wildcard patterns such as `/v1/transit/keys/*/rotate`

- Mid-path `*` wildcard now matches exactly one path segment

- Trailing wildcard `/*` behavior remains greedy for nested subpaths

### Runtime and Compatibility

- API baseline remains v1 (`/v1/*`)

- No breaking API path or payload contract changes

- Local development targets: Linux and macOS

- CI baseline: Go `1.25.5`, PostgreSQL `16-alpine`, MySQL `8.0`

- Compatibility targets: PostgreSQL `12+`, MySQL `8.0+`

### Upgrade Notes

- Recommended for all users relying on wildcard policy path matching

- No schema migrations required specifically for this bugfix release

- Existing tokenization, secrets, transit, auth, and audit flows remain API-compatible

### Policy Migration Note

If existing policies assumed prefix-only behavior, review wildcard paths used for rotate and similar endpoint-specific actions.

Before (too broad for intent):

```json
[
  {
    "path": "/v1/transit/keys/*",
    "capabilities": ["rotate"]
  }
]

```

After (scoped to rotate endpoint pattern):

```json
[
  {
    "path": "/v1/transit/keys/*/rotate",
    "capabilities": ["rotate"]
  }
]

```

---

## [0.4.0] - 2026-02-18

### Highlights

- Added tokenization API under `/v1/tokenization/*`

- Added tokenization key lifecycle: create, rotate, delete

- Added token lifecycle: tokenize, detokenize, validate, revoke

- Added deterministic mode support for repeatable token generation

- Added token format support: `uuid`, `numeric`, `luhn-preserving`, `alphanumeric`

- Added expired-token maintenance command: `clean-expired-tokens`

### API Additions

New endpoints:

- `POST /v1/tokenization/keys`

- `POST /v1/tokenization/keys/{name}/rotate`

- `DELETE /v1/tokenization/keys/{id}`

- `POST /v1/tokenization/keys/{name}/tokenize`

- `POST /v1/tokenization/detokenize`

- `POST /v1/tokenization/validate`

- `POST /v1/tokenization/revoke`

### CLI Additions

- `create-tokenization-key --name <name> --format <fmt> [--deterministic] [--algorithm <alg>]`

- `rotate-tokenization-key --name <name> --format <fmt> [--deterministic] [--algorithm <alg>]`

- `clean-expired-tokens --days <n> [--dry-run] [--format text|json]`

### Data Model and Migrations

Added migration `000002_add_tokenization` for PostgreSQL and MySQL:

- `tokenization_keys` table for versioned key metadata

- `tokenization_tokens` table for token-to-ciphertext mapping and lifecycle fields

### Observability

Added tokenization business operations metrics in the `tokenization` domain, including key and token lifecycle operations.

### Runtime and Compatibility

- API baseline remains v1 (`/v1/*`)

- Local development targets: Linux and macOS

- CI baseline: Go `1.25.5`, PostgreSQL `16-alpine`, MySQL `8.0`

- Compatibility targets: PostgreSQL `12+`, MySQL `8.0+`

### Upgrade Notes

- Non-breaking addition: tokenization capability under API v1

- Existing auth, secrets, transit, and audit behavior remain compatible

- Run database migrations before using tokenization endpoints or CLI commands

### Upgrade Checklist

1. Deploy binaries/images with `v0.4.0`
2. Run DB migrations (`app migrate`) before serving traffic
3. Verify baseline health (`GET /health`, `GET /ready`)
4. Create a tokenization key (`create-tokenization-key` or `POST /v1/tokenization/keys`)
5. Run round-trip check: tokenize -> detokenize -> validate -> revoke
6. Schedule retention cleanup for expired tokens (`clean-expired-tokens`)

### Rollback Notes

- `000002_add_tokenization` is additive schema migration and is expected to remain applied during app rollback.

- Rolling back binaries/images to pre-`v0.4.0` can leave tokenization tables unused but present.

- Avoid destructive schema rollback in production unless you have a validated backup/restore plan.

- If rollback is required, keep existing data and disable tokenization traffic paths operationally until re-upgrade.

---

## [0.3.0] - 2026-02-16

### Highlights

- Added OpenTelemetry metrics provider with Prometheus exporter

- Added optional `/metrics` endpoint for Prometheus scraping

- Added HTTP metrics middleware for request counts and latency histograms

- Added business operation metrics across auth, secrets, and transit use cases

- Added metrics configuration via `METRICS_ENABLED` and `METRICS_NAMESPACE`

### Metrics and Monitoring

New metric families:

- `{namespace}_http_requests_total`

- `{namespace}_http_request_duration_seconds`

- `{namespace}_operations_total`

- `{namespace}_operation_duration_seconds`

Runtime behavior:

- When `METRICS_ENABLED=true` (default), the server exposes `GET /metrics`

- When `METRICS_ENABLED=false`, metrics middleware and `/metrics` are not registered

- `METRICS_NAMESPACE` (default `secrets`) prefixes metric names

### Runtime and Compatibility

- API baseline remains v1 (`/v1/*`)

- Metrics endpoint is outside API versioning (`/metrics`)

- Local development targets: Linux and macOS

- CI baseline: Go `1.25.5`, PostgreSQL `16-alpine`, MySQL `8.0`

- Compatibility targets: PostgreSQL `12+`, MySQL `8.0+`

### Upgrade Notes

- Non-breaking addition: observability and metrics instrumentation

- Existing API paths and behavior remain compatible under API v1 documentation

- Update your environment configuration if you want custom metric namespace values

Example:

```bash
export METRICS_ENABLED=true
export METRICS_NAMESPACE=secrets
curl http://localhost:8080/metrics

```

---

## [0.2.0] - 2026-02-14

### Highlights

- New CLI command: `clean-audit-logs`

- Supports retention by age in days (`--days`)

- Supports safe preview mode (`--dry-run`) before deletion

- Supports machine-friendly output (`--format json`) and human-readable output (`--format text`)

### Included CLI Addition

- `clean-audit-logs --days <n> [--dry-run] [--format text|json]`

Operational behavior:

- Dry-run mode counts matching rows without deleting

- Execution mode deletes rows older than the computed UTC cutoff date

- Works with both PostgreSQL and MySQL repositories

### Runtime and Compatibility

- API baseline remains v1 (`/v1/*`)

- Local development targets: Linux and macOS

- CI baseline: Go `1.25.5`, PostgreSQL `16-alpine`, MySQL `8.0`

- Compatibility targets: PostgreSQL `12+`, MySQL `8.0+`

### Operational Notes

- Use `--dry-run` first for production safety

- Ensure database is reachable and migrated before cleanup runs

- Keep retention execution on a defined cadence (for example monthly)

Example:

```bash
./bin/app clean-audit-logs --days 90 --dry-run --format json

```

### Upgrade Notes

- Non-breaking addition: new CLI command for operations

- Existing API paths and behavior remain compatible under API v1 documentation

---

## [0.1.0] - 2026-02-14

### Highlights

- Envelope encryption model with `Master Key -> KEK -> DEK -> Secret Data`

- Transit encryption API for encrypt/decrypt without storing application payload

- Token authentication and policy-based authorization

- Versioned secret storage by path and soft-delete behavior

- Audit logging with request correlation via `request_id`

- PostgreSQL and MySQL runtime support

### Included API Surface

- Auth: `POST /v1/token`

- Clients: `GET/POST /v1/clients`, `GET/PUT/DELETE /v1/clients/:id`

- Secrets: `POST/GET/DELETE /v1/secrets/*path`

- Transit: create/rotate/encrypt/decrypt/delete under `/v1/transit/keys*`

- Audit logs: `GET /v1/audit-logs`

- Health/readiness: `GET /health`, `GET /ready`

### Runtime and Compatibility

- Local development targets: Linux and macOS

- CI baseline: Go `1.25.5`, PostgreSQL `16-alpine`, MySQL `8.0`

- Compatibility targets: PostgreSQL `12+`, MySQL `8.0+`

### Operational Notes

- Restart API servers after master key or KEK rotation so processes load new key material

- Base64 request fields are encoding only, not encryption; always use HTTPS/TLS

- For transit decrypt, pass ciphertext exactly as returned by encrypt (`<version>:<base64-ciphertext>`)

### Known Limitations (v0.1.0)

- `docs/openapi.yaml` is a baseline subset focused on common flows, not full endpoint parity

- API v1 compatibility policy applies to documented endpoint behavior in API reference docs

### Upgrade Notes

- Initial release: no prior upgrade path required

---

## See also

- [Release compatibility matrix](compatibility-matrix.md)

- [Documentation index](../README.md)

- [API compatibility policy](../api/fundamentals.md#compatibility-and-versioning-policy)

- [Production operations](../operations/deployment/docker-hardened.md)
