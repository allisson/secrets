# 🚀 Release Notes

> Last updated: 2026-02-28

This document contains release notes for all versions of Secrets.

## 📑 Quick Navigation

**Latest Release**: [v0.19.0](#0190---2026-02-27)

**All Releases**:

- [v0.19.0 (2026-02-27)](#0190---2026-02-27) - ⚠️ **Breaking Change**: KMS mode required

- [v0.18.0 (2026-02-27)](#0180---2026-02-27) - Repository layer refactoring

- [v0.17.0 (2026-02-25)](#0170---2026-02-25) - Pagination logic standardization

- [v0.16.0 (2026-02-25)](#0160---2026-02-25) - Listing Endpoints

- [v0.15.0 (2026-02-25)](#0150---2026-02-25) - Goreleaser support

- [v0.14.1 (2026-02-25)](#0141---2026-02-25) - KEK bug fixes

- [v0.14.0 (2026-02-25)](#0140---2026-02-25) - Dedicated metrics server

- [v0.13.0 (2026-02-25)](#0130---2026-02-25) - Documentation conciseness refactor

- [v0.12.0 (2026-02-24)](#0120---2026-02-24) - Rewrap DEKs

- [v0.11.0 (2026-02-23)](#0110---2026-02-23) - Account lockout

- [v0.10.0 (2026-02-21)](#0100---2026-02-21) - Docker security improvements

- [v0.9.0 (2026-02-20)](#090---2026-02-20) - Cryptographic audit log signing

- [v0.8.0 (2026-02-20)](#080---2026-02-20) - Documentation consolidation and ADR establishment

- [v0.7.0 (2026-02-20)](#070---2026-02-20) - IP-based rate limiting for token endpoint

- [v0.6.0 (2026-02-19)](#060---2026-02-19) - KMS provider support

- [v0.5.1 (2026-02-19)](#051---2026-02-19) - Master key loading fix

- [v0.5.0 (2026-02-19)](#050---2026-02-19) - Rate limiting and CORS

- [v0.4.1 (2026-02-19)](#041---2026-02-19) - Policy matcher wildcards

- [v0.4.0 (2026-02-18)](#040---2026-02-18) - Tokenization API

- [v0.3.0 (2026-02-16)](#030---2026-02-16) - OpenTelemetry metrics

- [v0.2.0 (2026-02-14)](#020---2026-02-14) - Audit log cleanup command

- [v0.1.0 (2026-02-14)](#010---2026-02-14) - Initial release

---

## [0.19.0] - 2026-02-27

### ⚠️ BREAKING CHANGES

**KMS mode is now required**. This is a breaking change that removes support for legacy plaintext master keys.

#### What Changed

- Legacy plaintext master key mode has been completely removed
- `create-master-key` command now requires `--kms-provider` and `--kms-key-uri` flags
- All deployments must use a KMS provider: `localsecrets`, `gcpkms`, `awskms`, `azurekeyvault`, or `hashivault`

#### Why This Change

- Enforces security best practices by requiring encrypted master keys at rest
- Simplifies codebase by removing dual-mode complexity
- Aligns with compliance requirements (PCI-DSS, HIPAA) that mandate encrypted key storage

#### Migration Required

If you are currently using legacy plaintext master keys (v0.18.0 or earlier), you **must** migrate to KMS mode before upgrading to v0.19.0.

**For local development:**

```bash
# Generate a KMS key
openssl rand -base64 32

# Create master key using localsecrets provider
./bin/app create-master-key \
  --id default \
  --kms-provider localsecrets \
  --kms-key-uri "base64key://YOUR_BASE64_KEY_HERE"
```

**For production:**

Use cloud KMS providers (`gcpkms`, `awskms`, `azurekeyvault`) or HashiCorp Vault (`hashivault`). See `docs/operations/kms/setup.md` for detailed setup instructions.

#### Configuration Changes

**Old configuration (v0.18.0 - no longer supported):**

```bash
MASTER_KEYS=default:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=
ACTIVE_MASTER_KEY_ID=default
```

**New configuration (v0.19.0 - required):**

```bash
KMS_PROVIDER=localsecrets
KMS_KEY_URI=base64key://YOUR_BASE64_KEY_HERE
MASTER_KEYS=default:ARiEeAASDiXKAxzOQCw2NxQ... # KMS-encrypted ciphertext
ACTIVE_MASTER_KEY_ID=default
```

### Removed

- Removed `LoadMasterKeyChainFromEnv` function from `internal/crypto/domain/master_key.go`
- Removed `docs/operations/kms/plaintext-to-kms-migration.md` (no longer applicable)

### Changed

- Updated all documentation to reflect KMS-only mode
- `.env.example` now defaults to `localsecrets` provider for local development with commented production examples
- Error messages updated to indicate KMS configuration is required
- Updated `docs/operations/kms/setup.md` to replace "Migration from Legacy Mode" with "Migrating Between KMS Providers"

### Documentation

- Updated `docs/configuration.md` to mark `KMS_PROVIDER` and `KMS_KEY_URI` as required
- Updated `docs/getting-started/local-development.md` with KMS setup instructions
- Updated `docs/getting-started/docker.md` with KMS quickstart examples
- Updated `docs/cli-commands.md` to reflect required KMS flags

---

## [0.18.0] - 2026-02-27

### Changed

- Refactored repository layer architecture by reorganizing database-specific implementations into dedicated `mysql/` and `postgresql/` subdirectories across all modules (`auth`, `crypto`, `secrets`, `tokenization`, `transit`). This improves code maintainability and enforces clearer separation of concerns.

---

## [0.17.0] - 2026-02-25

### Changed

- Standardized pagination logic (`offset`, `limit`) across all listing endpoints using a centralized parser in `httputil`

---

## [0.16.0] - 2026-02-25

### Added

- Added `/v1/secrets` endpoint for listing secrets with pagination (does not return secret data)
- Added `/v1/transit/keys` endpoint for listing transit keys with pagination
- Added `/v1/tokenization/keys` endpoint for listing tokenization keys with pagination

---

## [0.15.0] - 2026-02-25

### Added

- Goreleaser support for automated cross-platform builds and releases.

---

## [0.14.1] - 2026-02-25

### Fixed

- Fixed panic in `NewKekChain` when no KEKs are found
- Added proper `ErrKekNotFound` error handling in `kekUseCase.Unwrap` when attempting to unwrap with an empty KEK chain

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

---

## See also

- [Documentation index](../README.md)

- [API compatibility policy](../api/fundamentals.md#compatibility-and-versioning-policy)

- [Production operations](../operations/deployment/docker-hardened.md)
