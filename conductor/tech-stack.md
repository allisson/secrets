# Tech Stack: Secrets

## Core Technologies
- **Language:** [Go](https://go.dev/) (v1.26.0+) - Chosen for its high concurrency, strong typing, and performance.
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin) - A high-performance HTTP web framework with a fast HTTP router and custom middleware support.
- **CLI Framework:** [urfave/cli/v3](https://github.com/urfave/cli) - A library for building command-line applications in Go.

## Data Persistence
- **PostgreSQL:** Primary relational database for production environments. Supported via `lib/pq`.
- **MySQL:** Alternative relational database support for broader infrastructure compatibility. Supported via `go-sql-driver/mysql`.
- **Migrations:** [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Versioned database migrations for both PostgreSQL and MySQL.

## Cryptography & Security
- **Envelope Encryption:** [gocloud.dev/secrets](https://gocloud.dev/howto/secrets/) - Abstracted access to various KMS providers for root-of-trust encryption.
- **Password Hashing:** [go-pwdhash](https://github.com/allisson/go-pwdhash) - Argon2id hashing for secure storage of client secrets and passwords.
- **Request Body Size Limiting:** Middleware to prevent DoS attacks from large payloads.
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
