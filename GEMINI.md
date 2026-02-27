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
- **Test:** `make test` (unit tests) or `make test-with-db` (integration tests with Docker databases)
- **Lint:** `make lint` (uses `golangci-lint`)
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
- **Parallel Tests:** Unit tests should be able to run in parallel.
- **Integration Tests:** Located in `test/integration/`. Use `make test-with-db` to run them locally.
  - **CRITICAL:** Every new repository method (e.g., `Create`, `Get`, `List`, `Delete`) MUST have corresponding tests written natively in BOTH its `mysql_..._repository_test.go` and `postgresql_..._repository_test.go` files, and must be validated to pass against the test databases via `make test-with-db`.
- **HTTP/DTO Tests:** 
  - **CRITICAL:** Every new HTTP handler (e.g., `ListHandler`, `CreateHandler`) MUST have corresponding unit tests in its `..._handler_test.go` file.
  - **CRITICAL:** Every new mapping DTO (e.g., `MapSecretsToListResponse`) MUST have corresponding unit tests in its package (e.g., `list_secrets_response_test.go`) to ensure accurate payload mapping.
- **Usecase Tests:**
  - **CRITICAL:** Every new usecase method MUST have corresponding unit tests written natively in its `..._usecase_test.go` file to ensure core business logic is tested independently.
- **Coverage:** Aim for high coverage in `usecase` and `domain` layers.

### Contribution Guidelines
- **ADRs:** Major architectural decisions are documented as Architecture Decision Records in `docs/adr/`.
- **Documentation:** Maintain concise, reference-oriented documentation in the `docs/` directory following the Diátaxis framework principles. Avoid lengthy paragraphs in favor of bullet points, tables, and centralized code examples. **CRITICAL CI RULES:**
  1. **Freshness:** Any modified markdown file MUST have its `> Last updated: YYYY-MM-DD` header updated to match `last_docs_refresh` in `metadata.json`.
  2. **Changelog:** Every new version MUST be added to the high-level `CHANGELOG.md` in the root directory.
  3. **Main Version:** The `version` variable in `cmd/app/main.go` MUST be updated to match the new release version.
  4. **Docs Linting:** The command `make docs-lint` MUST be executed and all issues resolved.
  5. **OpenAPI Spec:** Any new API handler or configuration change MUST be reflected in `docs/openapi.yaml`.
- **Migrations:** New database changes must include both `up` and `down` SQL scripts for both MySQL and PostgreSQL.

### Tooling
- **Linting:** `golangci-lint` is mandatory.
- **Formatting:** Standard `go fmt`.
- **CI/CD:** GitHub Actions are used for CI (`.github/workflows/ci.yml`).
