# Tech Stack: Secrets

## Core Technologies
- **Language:** [Go](https://go.dev/) (v1.26.1+) - Chosen for its high concurrency, strong typing, and performance.
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin) - A high-performance HTTP web framework with a fast HTTP router and custom middleware support.
- **CLI Framework:** [urfave/cli/v3](https://github.com/urfave/cli) - A library for building command-line applications in Go.

## Data Persistence
- **PostgreSQL:** Primary relational database for production environments. Supported via `lib/pq`.
- **MySQL:** Alternative relational database support for broader infrastructure compatibility. Supported via `go-sql-driver/mysql`.
- **Connection Management:** Configurable connection pool settings including max open/idle connections, lifetime, and idle time for optimized resource usage.
- **Migrations:** [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Versioned database migrations for both PostgreSQL and MySQL.

## Cryptography & Security
- **Envelope Encryption:** [gocloud.dev/secrets](https://gocloud.dev/howto/secrets/) - Abstracted access to various KMS providers for root-of-trust encryption.
- **Password Hashing:** [go-pwdhash](https://github.com/allisson/go-pwdhash) - Argon2id hashing for secure storage of client secrets and passwords.
- **Configurable Metrics Timeouts:** Environment-controlled Read, Write, and Idle timeouts for the Prometheus metrics server to prevent resource exhaustion.
- **Request Body Size Limiting:** Middleware to prevent DoS attacks from large payloads.
- **Rate Limiting:** Per-client and per-IP rate limiting middleware for DoS protection and API abuse prevention.
- **Tokenization Batch Limit:** Configurable limit for batch tokenization operations to ensure predictable performance and resource usage.
- **Secret Value Size Limiting:** Global limit on individual secret values to ensure predictable storage and memory usage.
- **Strict Capability Validation:** Centralized domain helpers for validating policy capabilities (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`) in CLI and API layers.
- **Secret Path Validation:** Strict naming rules for secret paths (alphanumeric, -, _, /) to ensure consistency and security.
- **Audit Signing:** HMAC-SHA256 for tamper-evident cryptographic audit logs.

## KMS Providers (Native Support)
- **Google Cloud KMS**
- **AWS KMS**
- **Azure Key Vault**
- **HashiCorp Vault** (via transit engine)

## Observability & Monitoring
- **OpenTelemetry:** Native instrumentation for metrics using `go.opentelemetry.io/otel`.
- **Prometheus Export:** Standardized metrics endpoints compatible with Prometheus scrapers.

## Testing & Quality Assurance
- **Unit Testing:** [Testify](https://github.com/stretchr/testify) for assertions and mocks.
- **Database Mocking:** [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) for repository-level unit tests.
- **Static Analysis:** [golangci-lint](https://golangci-lint.run/) with `gosec` and `govulncheck` for security scanning.
