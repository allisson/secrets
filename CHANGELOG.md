# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.10.0] - 2026-02-21

### Added
- Docker image security improvements with Google Distroless base (Debian 13 Trixie)
- SHA256 digest pinning for immutable container builds
- Build-time version injection via ldflags (version, buildDate, commitSHA)
- Comprehensive OCI labels for better security scanning and SBOM generation
- Multi-architecture build support (linux/amd64, linux/arm64) in Dockerfile
- `.dockerignore` file to reduce build context size by ~90%
- Explicit non-root user execution (UID 65532: nonroot:nonroot)
- Read-only filesystem support for enhanced runtime security
- Container security documentation: `docs/operations/security/container-security.md`
- Health check endpoint documentation for Kubernetes and Docker Compose
- GitHub Actions workflow enhancements for build metadata injection
- Version management guidelines in AGENTS.md for coding agents

### Changed
- Base builder image: `golang:1.25.5-alpine` → `golang:1.25.5-trixie` (Debian 13)
- Final runtime image: `scratch` → `gcr.io/distroless/static-debian13@sha256:d90359c7a3ad67b3c11ca44fd5f3f5208cbef546f2e692b0dc3410a869de46bf`
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
- Added comprehensive container security guide (`docs/operations/security/container-security.md`) with 10 sections covering base image security, runtime security, network security, secrets management, image scanning, health checks, build security, and deployment best practices
- Added complete health check guide (`docs/operations/observability/health-checks.md`) with platform integrations for Kubernetes, Docker Compose, AWS ECS, Google Cloud Run, and monitoring tools
- Added security scanning guide (`docs/operations/security/scanning.md`) covering Trivy, Docker Scout, Grype, SBOM generation, and CI/CD integration
- Added OCI labels reference (`docs/operations/deployment/oci-labels.md`) documenting image metadata schema for security scanning and compliance
- Added Kubernetes deployment guide (`docs/operations/deployment/kubernetes.md`) with production-ready manifests and security hardening
- Added Docker Compose deployment guide (`docs/operations/deployment/docker-compose.md`) with development and production configurations
- Added multi-architecture builds guide (`docs/operations/deployment/multi-arch-builds.md`) for linux/amd64 and linux/arm64
- Added base image migration guide (`docs/operations/deployment/base-image-migration.md`) for Alpine/scratch to distroless transitions
- Added volume permissions troubleshooting guide (`docs/operations/troubleshooting/volume-permissions.md`) for non-root container issues
- Added error reference guide (`docs/operations/troubleshooting/error-reference.md`) with HTTP, database, KMS, and configuration errors
- Added comprehensive migration guide in `docs/releases/RELEASES.md` with rollback procedures and validation gates
- Added known issues section to `docs/releases/RELEASES.md` documenting ARM64 builds, health checks, and volume permissions
- Added rollback testing guidance to `docs/operations/deployment/production-rollout.md`
- Enhanced KMS security warnings in `docs/configuration.md` and `docs/operations/kms/setup.md`
- Updated Docker quick start guide with security features overview and health check examples
- Updated Dockerfile with comprehensive inline documentation (~180 comment lines)
- Added version management guidelines in AGENTS.md for AI coding agents

## [0.9.0] - 2026-02-20

### Added
- Added cryptographic audit log signing with HMAC-SHA256 for tamper detection
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

[0.10.0]: https://github.com/allisson/secrets/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/allisson/secrets/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/allisson/secrets/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/allisson/secrets/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/allisson/secrets/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/allisson/secrets/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/allisson/secrets/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/allisson/secrets/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/allisson/secrets/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/allisson/secrets/releases/tag/v0.1.0
