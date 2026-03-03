# GEMINI.md

## Interactive Planning Protocol
Whenever I request a complex task or code change:
1. Do not write the code immediately.
2. Enter Plan Mode and present a technical summary of what you intend to do.
3. Required refinement: If the task can be done in different ways, stop and ask 2 or 3 strategic questions to validate the approach (e.g. "Should we use NATS or PostgreSQL for this event?"), use the ask_user tool if available.
4. Wait for my confirmation or adjustment before proceeding with the implementation.

## Project Overview
**Secrets** is a lightweight secrets manager designed for simplicity and security. It provides envelope encryption, transit encryption, API authentication, and cryptographic audit logs. While inspired by HashiCorp Vault, it is intentionally simpler and focuses on ease of use and deployment.

### Main Technologies
- **Language:** Go 1.25
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin)
- **Databases:** PostgreSQL 12+ and MySQL 8.0+ (driver-agnostic)
- **Cryptography:** 
  - Envelope Encryption: `Master Key → KEK → DEK → Secret Data`
  - Algorithms: AES-GCM and ChaCha20-Poly1305
  - KMS Support: Google Cloud KMS, AWS KMS, Azure Key Vault, HashiCorp Vault (via `gocloud.dev`)
  - Password Hashing: Argon2id
  - Audit Log Signing: HMAC-SHA256
- **Observability:** OpenTelemetry metrics with Prometheus export
- **CLI Framework:** `urfave/cli/v3`

### Architecture
The project follows a **Modular Clean Architecture** (inspired by DDD) located in the `internal/` directory. Each module (e.g., `auth`, `secrets`, `transit`, `tokenization`) is organized into:
- `domain/`: Core entities and repository interfaces.
- `usecase/`: Application logic and business rules.
- `service/`: Domain-specific services.
- `repository/`: Persistence implementation (MySQL/PostgreSQL).
- `http/`: Web handlers and middleware.

## Building and Running

### Key Commands
- **Build:** `make build` (creates binary in `bin/app`)
- **Run Server:** `make run-server`
- **Run Migrations:** `make migrate-up`
- **Test:** `make test` (unit tests, fast), `make test-integration` (DB tests), `make test-with-db` (DB tests with lifecycle), `make test-all` (complete suite)
- **Lint:** `make lint` (runs `golangci-lint` with `gosec` + `govulncheck` for security scanning)
- **Docker:** `make docker-build`

### Configuration
Configuration is managed via environment variables (see `internal/config/config.go` and `.env.example`). Key variables include:
- `DB_DRIVER`: `postgres` or `mysql`
- `DB_CONNECTION_STRING`: Database URL
- `KMS_PROVIDER`: KMS provider name
- `KMS_KEY_URI`: URI for the master key in KMS
- `AUTH_TOKEN_EXPIRATION_SECONDS`: Token TTL

## Development Conventions

### Coding Style
- **Standard Go Layout:** CLI commands in `cmd/app`, core logic in `internal/`.
- **Error Handling:** Custom error package in `internal/errors`.
- **Validation:** Uses `github.com/jellydator/validation` for request validation.
- **Mocks:** Interface mocks are generated using `mockery` (run `make mocks`).

### Testing Practices

**Two-Tier Testing Strategy:**
- **Unit Tests:** Fast, in-memory tests (run in parallel, no external dependencies)
- **Integration Tests:** Database-dependent tests tagged with `//go:build integration`

**Quick Reference:**
- `make test` - Unit tests only (fast feedback)
- `make test-integration` - Integration tests only (requires running databases)
- `make test-with-db` - Integration tests with automatic DB lifecycle
- `make test-all` - Complete suite (unit + integration)

**Build Tags:**
- All database-dependent repository tests require `//go:build integration` as first line
- Applies to: `internal/*/repository/{postgresql,mysql}/*_test.go` and `test/integration/*_test.go`

**Test Organization:**
- Integration tests in `test/integration/` are split by feature area (auth_flow, kms_flow, secrets_flow, tokenization_flow, transit_flow, etc.)
- Each uses shared `helpers_test.go` for common setup utilities

**Critical Requirements:**
- **Repository:** Every new method MUST have tests in BOTH MySQL and PostgreSQL test files with `//go:build integration` tag
- **HTTP Handlers:** Every new handler MUST have unit tests in its `..._handler_test.go`
- **DTOs:** Every new mapping function MUST have unit tests for payload accuracy
- **Usecases:** Every new method MUST have unit tests in its `..._usecase_test.go`

**CI:** Unit tests run first (fast feedback), then integration tests (comprehensive validation). See `docs/contributing.md` for complete testing guide.

### Contribution Guidelines
- **ADRs:** Major architectural decisions are documented as Architecture Decision Records in `docs/adr/`.
- **Documentation:** Maintain concise, reference-oriented documentation in the `docs/` directory following the Diátaxis framework principles. Avoid lengthy paragraphs in favor of bullet points, tables, and centralized code examples. **CRITICAL CI RULES:**
  1. **Changelog:** Every new version MUST be added to the high-level `CHANGELOG.md` in the root directory.
  2. **Main Version:** The `version` variable in `cmd/app/main.go` MUST be updated to match the new release version.
  3. **Docs Linting:** The command `make docs-lint` MUST be executed and all issues resolved.
  4. **OpenAPI Spec:** Any new API handler or configuration change MUST be reflected in `docs/openapi.yaml`.
- **Migrations:** New database changes must include both `up` and `down` SQL scripts for both MySQL and PostgreSQL.

### Tooling
- **Linting:** `golangci-lint` is mandatory (includes `gosec` for security checks).
- **Security Scanning:** 
  - `gosec` runs via `golangci-lint` for static security analysis
  - `govulncheck` scans for known vulnerabilities in dependencies
  - Both run automatically in `make lint` and CI (required to pass before merge)
  - Auto-installs `govulncheck` if not present
- **Formatting:** Standard `go fmt`.
- **CI/CD:** GitHub Actions are used for CI (`.github/workflows/ci.yml`).
