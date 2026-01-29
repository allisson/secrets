# 🚀 Go Project Template

> A production-ready Go project template following Clean Architecture and Domain-Driven Design principles, optimized for building scalable applications with PostgreSQL or MySQL.

[![CI](https://github.com/allisson/go-project-template/workflows/CI/badge.svg)](https://github.com/allisson/go-project-template/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/allisson/go-project-template)](https://goreportcard.com/report/github.com/allisson/go-project-template)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ Features

- 🏗️ **Clean Architecture** - Clear separation of concerns with domain, repository, use case, and presentation layers
- 📦 **Modular Domain Structure** - Easy to add new domains without affecting existing code
- 🔌 **Dependency Injection** - Centralized component wiring with lazy initialization
- 🗄️ **Multi-Database Support** - PostgreSQL and MySQL with dedicated repository implementations
- 🔄 **Database Migrations** - Separate migrations for PostgreSQL and MySQL
- 🆔 **UUIDv7 Primary Keys** - Time-ordered, globally unique identifiers
- 💼 **Transaction Management** - Built-in support for database transactions
- 📬 **Transactional Outbox Pattern** - Event-driven architecture with guaranteed delivery
- ⚠️ **Standardized Error Handling** - Domain errors with proper HTTP status code mapping
- ✅ **Input Validation** - Advanced validation with custom rules (email, password strength, etc.)
- 🔒 **Password Hashing** - Secure Argon2id password hashing
- 🧪 **Integration Testing** - Real database tests instead of mocks
- 🐳 **Docker Support** - Multi-stage Dockerfile and Docker Compose setup
- 🚦 **CI/CD Ready** - GitHub Actions workflow with comprehensive testing
- 📊 **Structured Logging** - JSON logs using slog
- 🛠️ **Comprehensive Makefile** - Easy development and deployment commands

## 📚 Documentation

- 📖 [Getting Started](docs/getting-started.md) - Installation and setup guide
- 🏗️ [Architecture](docs/architecture.md) - Architectural patterns and design principles
- 🛠️ [Development](docs/development.md) - Development workflow and coding standards
- 🧪 [Testing](docs/testing.md) - Testing strategies and best practices
- ⚠️ [Error Handling](docs/error-handling.md) - Error handling system guide
- ➕ [Adding Domains](docs/adding-domains.md) - Step-by-step guide to add new domains

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 12+ or MySQL 8.0+
- Docker and Docker Compose (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/allisson/go-project-template.git
cd go-project-template

# Install dependencies
go mod download

# Start a database (using Docker)
make dev-postgres  # or make dev-mysql

# Run migrations
make run-migrate

# Start the server
make run-server
```

The server will be available at http://localhost:8080

For detailed setup instructions, see the [Getting Started Guide](docs/getting-started.md).

## 📖 Project Structure

```
go-project-template/
├── cmd/app/                    # Application entry point
├── internal/
│   ├── app/                    # Dependency injection container
│   ├── config/                 # Configuration management
│   ├── database/               # Database connection and transactions
│   ├── errors/                 # Standardized domain errors
│   ├── http/                   # HTTP server infrastructure
│   ├── httputil/               # HTTP utilities (JSON responses, error mapping)
│   ├── validation/             # Custom validation rules
│   ├── testutil/               # Test utilities
│   ├── user/                   # User domain module
│   │   ├── domain/             # User entities and domain errors
│   │   ├── usecase/            # User business logic
│   │   ├── repository/         # User data access
│   │   └── http/               # User HTTP handlers and DTOs
│   └── outbox/                 # Outbox domain module
│       ├── domain/             # Outbox entities and domain errors
│       ├── usecase/            # Outbox event processing logic
│       └── repository/         # Outbox data access
├── migrations/
│   ├── postgresql/             # PostgreSQL migrations
│   └── mysql/                  # MySQL migrations
├── docs/                       # Documentation
├── Dockerfile
├── Makefile
└── docker-compose.test.yml
```

Learn more about the architecture in the [Architecture Guide](docs/architecture.md).

## 🧪 Testing

```bash
# Start test databases
make test-db-up

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Stop test databases
make test-db-down

# Or run everything in one command
make test-with-db
```

The project uses real PostgreSQL and MySQL databases for testing instead of mocks. See the [Testing Guide](docs/testing.md) for details.

## 🛠️ Development

### Build Commands

```bash
make build                    # Build the application
make run-server              # Run HTTP server
make run-worker              # Run outbox event processor
make run-migrate             # Run database migrations
make lint                    # Run linter with auto-fix
make clean                   # Clean build artifacts
```

### API Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Register User
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "SecurePass123!"
  }'
```

For more development workflows, see the [Development Guide](docs/development.md).

## 🔑 Key Concepts

### Clean Architecture Layers

- **Domain Layer** 🎯 - Business entities and rules (pure, no external dependencies)
- **Repository Layer** 💾 - Data persistence (separate MySQL and PostgreSQL implementations)
- **Use Case Layer** 💼 - Business logic and orchestration
- **Presentation Layer** 🌐 - HTTP handlers and DTOs
- **Utility Layer** 🛠️ - Shared utilities (error handling, validation, HTTP helpers)

### Error Handling System

The project uses a standardized error handling system:

- **Standard Errors**: `ErrNotFound`, `ErrConflict`, `ErrInvalidInput`, `ErrUnauthorized`, `ErrForbidden`
- **Domain Errors**: Wrap standard errors with domain-specific context
- **Automatic HTTP Mapping**: Domain errors automatically map to appropriate HTTP status codes

Example:
```go
// Define domain error
var ErrUserNotFound = errors.Wrap(errors.ErrNotFound, "user not found")

// Use in repository
if errors.Is(err, sql.ErrNoRows) {
    return nil, domain.ErrUserNotFound  // Maps to 404 Not Found
}
```

Learn more in the [Error Handling Guide](docs/error-handling.md).

### UUIDv7 Primary Keys

All entities use UUIDv7 for primary keys:
- ⏱️ Time-ordered for better database performance
- 🌍 Globally unique across distributed systems
- 📊 Better than UUIDv4 for database indexes

### Transactional Outbox Pattern

Ensures reliable event delivery using a use case-based approach:
1. Business operation and event stored in same transaction
2. Outbox use case processes pending events with configurable retry logic
3. Guarantees at-least-once delivery
4. Extensible event processing via the `EventProcessor` interface

## 🐳 Docker

```bash
# Build Docker image
make docker-build

# Run server in Docker
make docker-run-server

# Run worker in Docker
make docker-run-worker

# Run migrations in Docker
make docker-run-migrate
```

The worker command runs the outbox event processor, which handles asynchronous event processing using the transactional outbox pattern.

## 🔧 Configuration

All configuration is done via environment variables. Create a `.env` file in your project root:

```bash
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
LOG_LEVEL=info
```

See the [Getting Started Guide](docs/getting-started.md) for all available configuration options.

## ➕ Adding New Domains

Adding a new domain is straightforward:

1. Create domain structure (`domain/`, `usecase/`, `repository/`, `http/`)
2. Define domain entity and errors
3. Create database migrations
4. Implement repositories (PostgreSQL and MySQL)
5. Implement use case with business logic
6. Create DTOs and HTTP handlers
7. Register in DI container
8. Wire HTTP routes

See the [Adding Domains Guide](docs/adding-domains.md) for a complete step-by-step tutorial.

## 📦 Dependencies

### Core Libraries

- [google/uuid](https://github.com/google/uuid) - UUID generation (UUIDv7 support)
- [jellydator/validation](https://github.com/jellydator/validation) - Advanced input validation
- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [allisson/go-env](https://github.com/allisson/go-env) - Environment configuration
- [allisson/go-pwdhash](https://github.com/allisson/go-pwdhash) - Argon2id password hashing
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Database migrations

### Database Drivers

- [lib/pq](https://github.com/lib/pq) - PostgreSQL driver
- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) - MySQL driver

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

This template leverages these excellent Go libraries:
- github.com/allisson/go-env
- github.com/allisson/go-pwdhash
- github.com/jellydator/validation
- github.com/google/uuid
- github.com/urfave/cli
- github.com/golang-migrate/migrate

---

<div align="center">
  <sub>Built with ❤️ by <a href="https://github.com/allisson">Allisson</a></sub>
</div>
