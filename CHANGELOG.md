# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0] - 2026-02-20

### Added
- Added cryptographic audit log signing with HMAC-SHA256 for tamper detection (PCI DSS Requirement 10.2.2)
- Added HKDF-SHA256 key derivation to separate encryption and signing key usage
- Added `verify-audit-logs` CLI command for batch integrity verification with text/JSON output
- Added database columns: `signature` (BYTEA), `kek_id` (UUID FK), `is_signed` (BOOLEAN)
- Added foreign key constraints: `fk_audit_logs_client_id` and `fk_audit_logs_kek_id` to prevent orphaned records
- Added `AuditSigner` service for canonical log serialization and HMAC generation
- Added test infrastructure: `CreateTestClient()` and `CreateTestKek()` helpers for FK-compliant testing

### Changed
- Audit logs now automatically signed on creation when KEK chain is available
- Audit log API responses now include signature metadata (`signature`, `kek_id`, `is_signed`)
- Database migration 000003 required (adds signature columns and FK constraints)

### Fixed
- Fixed 46 audit log repository tests to comply with FK constraints

### Security
- Enhanced audit log tamper detection with cryptographic integrity verification
- Enforced data integrity with FK constraints preventing orphaned client/KEK references

### Documentation
- Added `docs/releases/v0.9.0-upgrade.md` upgrade guide with pre/post-migration checks
- Updated `docs/cli-commands.md` with `verify-audit-logs` command
- Updated `docs/api/observability/audit-logs.md` with signature field documentation
- Added AGENTS.md guidelines for audit signer architecture and FK testing patterns

## [0.8.0] - 2026-02-20

### Documentation
- Documentation consolidation: reduced from 77 to 47 markdown files (39% reduction)
- Established 8 new Architecture Decision Records (ADR 0003-0010) covering key architectural decisions
- Restructured API documentation with themed subdirectories (auth/, data/, observability/)
- Consolidated operations documentation with centralized runbook hub
- Merged all development documentation into contributing.md
- Comprehensive cross-reference updates throughout documentation (182+ updates)

## [0.7.0] - 2026-02-20

### Added
- Added IP-based rate limiting middleware for unauthenticated `POST /v1/token`
- Added token endpoint rate-limit configuration via `RATE_LIMIT_TOKEN_ENABLED`, `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC`, and `RATE_LIMIT_TOKEN_BURST`

### Changed
- Token issuance endpoint can now return `429 Too Many Requests` with `Retry-After` when per-IP limits are exceeded

### Security
- Hardened token issuance path against credential stuffing and brute-force request bursts

### Documentation
- Added `docs/releases/v0.7.0.md` release notes and `docs/releases/v0.7.0-upgrade.md` upgrade guide
- Updated docs for token endpoint throttling behavior, configuration, and troubleshooting guidance

## [0.6.0] - 2026-02-19

### Added
- Added KMS-backed master key support with `KMS_PROVIDER` and `KMS_KEY_URI`
- Added `rotate-master-key` CLI command for staged master key rotation
- Added `create-master-key` KMS flags: `--kms-provider` and `--kms-key-uri`
- Added gocloud-based KMS service support for `localsecrets`, Google Cloud KMS, AWS KMS, Azure Key Vault, and HashiCorp Vault

### Changed
- Master key loading now auto-detects KMS mode vs legacy mode and validates KMS configuration consistency at startup

### Security
- Added encrypted-at-rest master key workflow through external KMS providers
- Added startup validation and error paths for incomplete KMS configuration and decryption failures

### Documentation
- Added `docs/releases/v0.6.0.md` release notes and `docs/releases/v0.6.0-upgrade.md` upgrade guide
- Added KMS operations guide: `docs/operations/kms/setup.md`
- Updated CLI and environment variable docs for KMS configuration and master key rotation workflows

## [0.5.1] - 2026-02-19

### Fixed
- Fixed master key loading from `MASTER_KEYS` so decoded key material remains usable after secure buffer zeroing
- Fixed `MasterKeyChain.Close()` to zero all in-memory master keys before clearing chain state

### Security
- Hardened master key memory lifecycle by zeroing temporary decode buffers and keychain-resident keys on teardown
- Added regression tests for key usability-after-load and key zeroing-on-close behavior

### Documentation
- Added `docs/releases/v0.5.1.md` release notes and `docs/releases/v0.5.1-upgrade.md` upgrade guide
- Updated current release references and pinned examples to `v0.5.1`

## [0.5.0] - 2026-02-19

### Added
- Per-client rate limiting for authenticated endpoints (default: 10 req/sec, burst 20)
- Configurable CORS support (disabled by default)
- Comprehensive security hardening documentation (`docs/operations/security/hardening.md`)
- Rate limiting configuration via `RATE_LIMIT_ENABLED`, `RATE_LIMIT_REQUESTS_PER_SEC`, `RATE_LIMIT_BURST`
- CORS configuration via `CORS_ENABLED`, `CORS_ALLOW_ORIGINS`

### Changed
- **BREAKING**: Default token expiration reduced from 24 hours to 4 hours (86400 → 14400 seconds)
- Updated environment variables documentation with security warnings
- Updated production deployment guide with security hardening reference

### Migration Notes

**Token Expiration Change:**
If you rely on the previous default token expiration of 24 hours, explicitly set `AUTH_TOKEN_EXPIRATION_SECONDS=86400` in your environment configuration. Otherwise, tokens will now expire after 4 hours by default.

**Review Client Token Refresh Logic:**
Ensure your client applications handle token refresh before expiration. The shorter default expiration improves security but may require updating client-side token refresh logic if you were relying on the previous 24-hour default.

**Database SSL/TLS:**
If you are using `sslmode=disable` (PostgreSQL) or `tls=false` (MySQL) in production, this is insecure. Update your `DB_CONNECTION_STRING` to use `sslmode=require` or `sslmode=verify-full` (PostgreSQL) or `tls=true` or `tls=custom` (MySQL). See `docs/operations/security/hardening.md` for guidance.

### Security
- Added database SSL/TLS configuration warnings in documentation
- Added reverse proxy TLS requirements in documentation
- Added master key storage security guidance
- Added metrics endpoint protection recommendations

### Documentation
- Added `docs/operations/security/hardening.md` with comprehensive security guidance
- Updated `docs/configuration/environment-variables.md` with new variables and security warnings
- Updated `.env.example` with security warnings for development-only configurations
- Updated `docs/getting-started/docker.md` and `docs/getting-started/local-development.md` with security warnings
- Updated `docs/concepts/security-model.md` with production recommendations
- Updated `README.md` with security hardening link

## [0.4.1] - 2026-02-19

### Fixed
- Policy matcher now supports mid-path wildcard patterns (e.g., `/v1/transit/keys/*/rotate`)
- Mid-path `*` wildcard now matches exactly one path segment
- Trailing wildcard `/*` behavior remains greedy for nested subpaths

### Documentation
- Added policy path-matching behavior documentation
- Added policy migration examples for wildcard patterns
- Added policy review checklist for operators

## [0.4.0] - 2026-02-18

### Added
- Tokenization API for token generation, detokenization, validation, and revocation
- Tokenization key management (create, rotate, delete)
- Deterministic and non-deterministic tokenization support
- Token TTL and revocation capabilities
- Token metadata support (non-encrypted)
- CLI commands for tokenization key management
- Expired token cleanup command (`clean-expired-tokens`)

### Documentation
- Added `docs/api/tokenization.md` with API reference
- Added tokenization examples in curl, Python, JavaScript, and Go
- Added tokenization monitoring and operations guidance
- Added tokenization migration verification guide

## [0.3.0] - 2026-02-16

### Added
- OpenTelemetry metrics collection with Prometheus-compatible `/metrics` endpoint
- Configurable metrics namespace via `METRICS_NAMESPACE`
- Metrics enable/disable toggle via `METRICS_ENABLED`
- HTTP request metrics (total requests, duration, status codes)
- Cryptographic operation metrics (secret operations, transit operations, audit log operations)

### Documentation
- Added `docs/operations/observability/monitoring.md` with Prometheus and Grafana quickstart
- Added metrics naming contract and endpoint documentation
- Added production hardening guidance for securing `/metrics` endpoint

## [0.2.0] - 2026-02-14

### Added
- Audit log retention cleanup command (`clean-audit-logs`)
- Dry-run mode for audit log cleanup
- JSON and text output formats for cleanup commands

### Documentation
- Added audit log retention cleanup runbook
- Added CLI reference documentation
- Updated production operations guide with retention workflows

## [0.1.0] - 2026-02-14

### Added
- Envelope encryption with Master Key → KEK → DEK → Data hierarchy
- Transit encryption API (encrypt/decrypt as a service)
- Token-based authentication and capability-based authorization
- Versioned secrets storage by path
- Audit logging with request correlation
- Support for PostgreSQL and MySQL databases
- Support for AES-GCM and ChaCha20-Poly1305 encryption algorithms
- Health and readiness endpoints
- Client management API (create, get, update, delete)
- Master key and KEK management CLI commands
- Docker image distribution

### Documentation
- Initial documentation structure
- API reference documentation
- Getting started guides (Docker and local development)
- Operations guides (production deployment, key management)
- Example code (curl, Python, JavaScript, Go)
- Security model documentation
- Architecture documentation
