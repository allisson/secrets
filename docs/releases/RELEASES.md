# 🚀 Release Notes

> Last updated: 2026-02-20

This document contains release notes and upgrade guides for all versions of Secrets.

For the compatibility matrix across versions, see [compatibility-matrix.md](compatibility-matrix.md).

## 📑 Quick Navigation

**Latest Release**: [v0.8.0](#080---2026-02-20)

**All Releases**:

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

### Upgrade from v0.7.0

#### What Changed

- Documentation structure improvements (no code or runtime changes)
- All v0.7.0 functionality remains identical
- No environment variables, schema, or API changes

#### Upgrade Steps

No upgrade required. v0.8.0 is documentation-only and fully backward compatible with v0.7.0.

If referencing documentation, update any bookmarks or links to reflect new documentation structure:

- API fundamentals consolidated into `docs/api/fundamentals.md`
- API endpoints organized by theme: `auth/`, `data/`, `observability/`
- Operations runbooks centralized in `docs/operations/runbooks/README.md`
- Development guide now at `docs/contributing.md`

#### Documentation Updates

- 8 new ADRs documenting architectural decisions (capability-based auth, dual database support, transaction management, rate limiting, API versioning, Gin framework, UUIDv7, Argon2id)
- API documentation restructured with auth/, data/, observability/ subdirectories
- Operations documentation consolidated with runbook hub and themed organization
- All development documentation merged into single contributing.md guide
- Comprehensive cross-reference updates (182+ link updates)
- All validation passing (627 OK links, 0 errors)

#### See Also

- [Compatibility matrix](compatibility-matrix.md)
- [Architecture Decision Records](../adr/)
- [Documentation index](../README.md)

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

### Upgrade from v0.6.0

#### What Changed

- Added IP-based token endpoint rate limiting for `POST /v1/token`
- Added new token endpoint throttling configuration (`RATE_LIMIT_TOKEN_*`)
- Token issuance can now return `429 Too Many Requests` with `Retry-After`

#### Env Diff

```diff
+ RATE_LIMIT_TOKEN_ENABLED=true
+ RATE_LIMIT_TOKEN_REQUESTS_PER_SEC=5.0
+ RATE_LIMIT_TOKEN_BURST=10
```

#### Recommended Upgrade Steps

1. Update image/binary to `v0.7.0`
2. Add `RATE_LIMIT_TOKEN_*` variables to runtime configuration
3. Restart API instances with standard rolling rollout process
4. Run baseline checks: `GET /health`, `GET /ready`
5. Run token and key-dependent checks

#### Quick Verification Commands

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready

TOKEN_RESPONSE="$(curl -sS -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}')"

CLIENT_TOKEN="$(printf '%s' "${TOKEN_RESPONSE}" | jq -r '.token')"

curl -sS -X POST http://localhost:8080/v1/secrets/upgrade/v070 \
  -H "Authorization: Bearer ${CLIENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"value":"djA3MC1zbW9rZQ=="}'
```

#### Operator Verification Checklist

1. Confirm `GET /health` and `GET /ready` succeed
2. Confirm `POST /v1/token` issues tokens normally for expected request rates
3. Confirm token endpoint returns controlled `429` with `Retry-After` when intentionally exceeded
4. Confirm authenticated route limits and retry behavior still match policy

#### Documentation Updates

- Added [API rate limiting](../api/fundamentals.md#rate-limiting) with token endpoint scope
- Updated [Environment variables](../configuration.md) with `RATE_LIMIT_TOKEN_*`
- Updated [Troubleshooting](../getting-started/troubleshooting.md) with token endpoint `429` diagnostics

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

### Upgrade from v0.5.1

#### What Changed

- Added KMS-backed master key loading mode (`KMS_PROVIDER`, `KMS_KEY_URI`)
- Added KMS flags to `create-master-key`
- Added `rotate-master-key` CLI command for staged master key rotation
- Added fail-fast validation for partial KMS configuration

#### Recommended Upgrade Steps

1. Update image/binary to `v0.6.0`
2. Decide runtime key mode:
   - Keep legacy mode (no KMS vars set), or
   - Enable KMS mode (`KMS_PROVIDER` and `KMS_KEY_URI` both set)
3. Restart API instances with standard rolling rollout process
4. Run baseline checks: `GET /health`, `GET /ready`
5. Run key-dependent smoke checks

#### Decision Path

- **Stay on legacy mode now:**
  - Keep `KMS_PROVIDER` and `KMS_KEY_URI` unset
  - Upgrade binaries/images and validate normal crypto flows
- **Adopt KMS mode now:**
  - Set both `KMS_PROVIDER` and `KMS_KEY_URI`
  - Ensure all `MASTER_KEYS` entries are KMS ciphertext
  - Follow migration workflow in [KMS setup guide](../operations/kms/setup.md)
  - Track rollout gates in [KMS migration checklist](../operations/kms/setup.md#migration-checklist)

#### Quick Verification Commands

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready

TOKEN_RESPONSE="$(curl -sS -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}')"

CLIENT_TOKEN="$(printf '%s' "${TOKEN_RESPONSE}" | jq -r '.token')"

curl -sS -X POST http://localhost:8080/v1/secrets/upgrade/v060 \
  -H "Authorization: Bearer ${CLIENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"value":"djA2MC1zbW9rZQ=="}'
```

#### Operator Verification Checklist

1. Confirm `GET /health` and `GET /ready` succeed
2. Confirm startup logs reflect intended key mode and active master key
3. Confirm token issuance and secrets/transit round-trip flows
4. Confirm no KMS auth/decrypt errors in startup logs

#### Documentation Updates

- Added [KMS setup guide](../operations/kms/setup.md)
- Updated [CLI commands](../cli-commands.md) with KMS flags and `rotate-master-key`
- Updated [Environment variables](../configuration.md) with KMS mode configuration

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

### Upgrade from v0.5.0

#### What Changed

- Fixed master key loading from `MASTER_KEYS` to preserve active key material after decode
- Added secure zeroing of all keychain-held master keys during `Close`
- Added regression test coverage for these memory lifecycle paths

#### Recommended Upgrade Steps

1. Update image/binary to `v0.5.1`
2. Restart API instances with standard rolling rollout process
3. Run baseline checks: `GET /health`, `GET /ready`
4. Run key-dependent smoke checks: token issuance, secrets write/read, transit encrypt/decrypt

#### Quick Verification Commands

```bash
curl -sS http://localhost:8080/health

TOKEN_RESPONSE="$(curl -sS -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}')"

CLIENT_TOKEN="$(printf '%s' "${TOKEN_RESPONSE}" | jq -r '.token')"

curl -sS -X POST http://localhost:8080/v1/secrets/upgrade/smoke \
  -H "Authorization: Bearer ${CLIENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"value":"c21va2UtdjA1MQ=="}'

curl -sS -X GET http://localhost:8080/v1/secrets/upgrade/smoke \
  -H "Authorization: Bearer ${CLIENT_TOKEN}"
```

#### Operator Verification Checklist

1. Confirm service health and readiness after rollout
2. Confirm startup succeeds with configured `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID`
3. Confirm secrets and transit workflows succeed under normal traffic
4. Confirm no unexpected key configuration or decryption errors in logs

#### Patch Release Safety

- Most environments require no configuration changes for this release
- Rolling upgrade is recommended; keep standard health and smoke checks in place
- Rollback to the previous stable image is safe when incident criteria are met

#### Documentation Updates

- Updated [release compatibility matrix](compatibility-matrix.md) with `v0.5.0 -> v0.5.1`
- Updated current-release references across docs and pinned image examples to `v0.5.1`

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

### Upgrade from v0.4.x

#### What changed

- Default token expiration is now shorter (`24h` -> `4h`)
- Per-client rate limiting is enabled by default
- CORS is configurable and remains disabled by default
- Security hardening guidance expanded for production deployments

#### Env diff

```diff
- AUTH_TOKEN_EXPIRATION_SECONDS=86400
+ AUTH_TOKEN_EXPIRATION_SECONDS=14400

+ RATE_LIMIT_ENABLED=true
+ RATE_LIMIT_REQUESTS_PER_SEC=10.0
+ RATE_LIMIT_BURST=20

+ CORS_ENABLED=false
+ CORS_ALLOW_ORIGINS=
```

If your clients rely on 24-hour tokens, keep explicit configuration:

```dotenv
AUTH_TOKEN_EXPIRATION_SECONDS=86400
```

#### Upgrade steps

1. Update image/binary to `v0.5.0`
2. Review and set explicit `AUTH_TOKEN_EXPIRATION_SECONDS`
3. Add `RATE_LIMIT_*` variables with values matching your traffic profile
4. Keep `CORS_ENABLED=false` unless browser-based access is required
5. Restart API servers with updated environment

#### Post-upgrade verification

1. Health checks pass: `GET /health`, `GET /ready`
2. Token issuance works and expiration matches expected TTL
3. Authenticated endpoint rate limit returns `429` with `Retry-After` when exceeded
4. Normal traffic does not hit `429` unexpectedly
5. CORS behavior is correct for your deployment mode

#### Operator Verification Checklist

1. Confirm health endpoints: `GET /health`, `GET /ready`
2. Validate token issuance and expiration expectations after upgrade
3. Confirm authenticated API traffic is not unintentionally rate limited
4. Validate `429` behavior and `Retry-After` header with controlled load test
5. Confirm CORS behavior matches policy (disabled by default, explicit origins only when enabled)

#### Security Guidance

- Use TLS termination at reverse proxy/load balancer
- Use database TLS in production (`sslmode=require` or stronger / `tls=true` or stronger)
- Store master keys in a dedicated secrets manager
- Review least-privilege client policies and rotate credentials regularly

#### Documentation Updates

- Added [Security hardening guide](../operations/security/hardening.md)
- Updated [Environment variables](../configuration.md) with rate limiting, CORS, and token expiration migration notes
- Updated [Production deployment guide](../operations/deployment/production.md) with security hardening links

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

### Verification Checklist

1. Deploy binaries/images with `v0.4.1`
2. Verify baseline health (`GET /health`, `GET /ready`)
3. Re-run policy smoke checks for expected allow/deny behavior
4. Confirm wildcard policies used in production match intended path semantics

### Operator Quick Checklist (v0.4.1)

1. Search client policies for rotate patterns and replace broad forms with `/v1/transit/keys/*/rotate` when needed.
2. Run route-shape smoke checks (`/v1/transit/keys/payment/extra/rotate` and `/v1/secrets`) and expect `404`.
3. Run allow/deny policy smoke checks and expect capability-denied calls to return `403`.
4. Review recent denied audit events and confirm mismatches are expected after policy rollout.

### Documentation Migration Map (v0.4.1)

- Policy matching semantics: [Policies cookbook / Path matching behavior](../api/auth/policies.md#path-matching-behavior)
- Route-vs-policy triage: [Policies cookbook / Route shape vs policy shape](../api/auth/policies.md#route-shape-vs-policy-shape)
- Pre-deploy policy checks: [Policies cookbook / Policy review checklist before deploy](../api/auth/policies.md#policy-review-checklist-before-deploy)
- Capability verification: [Capability matrix](../api/fundamentals.md#capability-matrix)
- Operational validation steps: [Policy smoke tests](../operations/runbooks/policy-smoke-tests.md)
- Incident triage and matcher FAQ: [Troubleshooting](../getting-started/troubleshooting.md)

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

### Documentation Updates

- Added [Tokenization API](../api/data/tokenization.md) reference
- Updated [CLI commands reference](../cli-commands.md) with tokenization commands
- Updated [Production operations](../operations/deployment/production.md) with tokenization workflows

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

### Documentation Updates

- Added [Monitoring operations guide](../operations/observability/monitoring.md)
- Updated [Environment variables](../configuration.md)
- Updated [Production operations](../operations/deployment/production.md)

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

### Documentation Updates

- Updated [CLI commands reference](../cli-commands.md)
- Updated [Audit Logs API](../api/observability/audit-logs.md)
- Updated [Production operations](../operations/deployment/production.md)

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
- [Production operations](../operations/deployment/production.md)
