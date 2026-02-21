# AGENTS.md - Coding Agent Guide

This document provides essential information for AI coding agents working in this repository. It covers build commands, code style, architecture patterns, and conventions.

## Project Overview

**Secrets** is a Go-based secrets management service with envelope encryption, transit encryption, API auth, and audit logs. The project uses Clean Architecture with a clear separation between domain, use cases, and infrastructure layers.

- **Language**: Go 1.25
- **Architecture**: Clean Architecture (cmd/, internal/ structure)
- **Database**: PostgreSQL 12+ or MySQL 8.0+ (driver-agnostic)
- **Framework**: Gin (HTTP), testify (testing), mockery (mocks)
- **Build System**: Makefile + Go toolchain

## Build, Test, and Lint Commands

### Building

```bash
# Build the application
make build

# Build produces: bin/app
go build -o bin/app ./cmd/app
```

### Testing

```bash
# Run all tests with coverage
make test
go test -v -race -p 1 -coverprofile=coverage.out ./...

# Run tests with test databases (postgres + mysql)
make test-with-db

# Run a single test
go test -v -run TestName ./internal/package/path

# Run a single test in a specific file
go test -v -run TestFunctionName ./internal/auth/usecase

# Run tests matching a pattern
go test -v -run "TestClient.*" ./internal/auth/domain

# Run tests with verbose output
go test -v ./internal/auth/service

# View coverage report in browser
make test-coverage
```

### Linting

```bash
# Run linter (includes auto-fix)
make lint
golangci-lint run -v --fix

# Linter uses: goimports, golines, gosec, gocritic
# Max line length: 110 characters
# Tab length: 4 spaces
```

### Other Commands

```bash
# Regenerate mocks (after changing interfaces)
make mocks

# Run migrations
make run-migrate

# Clean build artifacts
make clean
```

## Version Management

### Version Update Guidelines

When updating the application version, the following files MUST be updated together:

1. **`cmd/app/main.go`** - Update the `version` variable default value
   ```go
   var (
       version   = "0.10.0" // Update this for each release
       buildDate = "unknown"
       commitSHA = "unknown"
   )
   ```

2. **`docs/metadata.json`** - Update `current_release` and `last_docs_refresh`
   ```json
   {
     "current_release": "v0.10.0",
     "api_version": "v1",
     "last_docs_refresh": "2026-02-21"
   }
   ```

3. **`CHANGELOG.md`** - Add new release section at the top
   - Use semantic versioning (MAJOR.MINOR.PATCH)
   - Document all changes under Added/Changed/Removed/Fixed/Security/Documentation
   - Add comparison link at bottom: `[X.Y.Z]: https://github.com/allisson/secrets/compare/vA.B.C...vX.Y.Z`

4. **`README.md`** - Update version references in "What's New" section (if applicable)

### Version Numbering Rules

- **MAJOR** (X.0.0): Breaking changes, incompatible API changes
- **MINOR** (0.X.0): New features, backward-compatible functionality
- **PATCH** (0.0.X): Bug fixes, backward-compatible fixes

**Examples**:
- Database schema changes → MINOR or MAJOR (depending on compatibility)
- New API endpoints → MINOR
- Security fixes → PATCH
- Docker base image changes → MINOR (infrastructure change)
- Documentation-only changes → PATCH

### Build-Time Version Injection

The version is injected at build time via ldflags in the Dockerfile:

```dockerfile
-ldflags="-w -s \
-X main.version=${VERSION} \
-X main.buildDate=${BUILD_DATE} \
-X main.commitSHA=${COMMIT_SHA}"
```

**Local builds** without ldflags will use the default values from `cmd/app/main.go`.

**CI/CD builds** (GitHub Actions) automatically inject:
- `VERSION`: Git tag (e.g., `v0.10.0`) or `dev` for non-tagged builds
- `BUILD_DATE`: ISO 8601 timestamp (e.g., `2026-02-21T10:30:00Z`)
- `COMMIT_SHA`: Full git commit hash

### Version Verification

After building, verify the version:

```bash
# Local binary
./bin/app --version

# Example output:
#   Version:    0.10.0
#   Build Date: unknown
#   Commit SHA: unknown

# Docker image
docker run --rm allisson/secrets:latest --version

# Example output (with injected build metadata):
#   Version:    v0.10.0
#   Build Date: 2026-02-21T10:30:00Z
#   Commit SHA: 23d48a137821f9428304e9929cf470adf8c3dee6
```

**Note**: Local builds without ldflags will show default values (`Build Date: unknown`, `Commit SHA: unknown`). Docker and CI/CD builds inject actual metadata via build args.

## Docker Commands

### Building Images

Build production-ready Docker images with security features and version injection.

```bash
# Build with auto-detected version (from git tags)
make docker-build
# Produces: allisson/secrets:latest, allisson/secrets:<VERSION>

# Custom registry
make docker-build DOCKER_REGISTRY=myregistry.io/myorg

# Override version
make docker-build VERSION=v1.0.0-rc1
```

**Version injection** (automatic via build args):
- `VERSION`: Git tag (e.g., `v0.10.0`), commit hash, or `"dev"` fallback
- `BUILD_DATE`: ISO 8601 UTC timestamp
- `COMMIT_SHA`: Full git commit hash

### Multi-Architecture Builds

Build and push multi-platform images for amd64 and arm64 architectures.

**Requirements**: Docker Buildx (included in Docker Desktop 19.03+), authenticated registry access

```bash
# Authenticate to registry
docker login

# Build and push multi-arch images (linux/amd64, linux/arm64)
make docker-build-multiarch VERSION=v0.10.0

# Verify images
docker manifest inspect allisson/secrets:v0.10.0
```

**Note**: Images are automatically pushed to the registry. Use `docker-build` for local testing.

### Inspecting and Scanning

**Inspect image metadata** (requires `jq`):
```bash
make docker-inspect

# Displays:
#   - Version information (version, build date, commit SHA)
#   - Security settings (user, base image)
#   - Full OCI labels (JSON format)
```

**Scan for vulnerabilities**:
```bash
make docker-scan

# Uses Trivy to scan for HIGH and CRITICAL CVEs
# If Trivy not installed, provides installation instructions

# Manual scan alternative:
trivy image --severity HIGH,CRITICAL allisson/secrets:latest
```

### Running Containers

**Run HTTP server**:
```bash
make docker-run-server

# Runs on http://localhost:8080
# Health endpoints: /health (liveness), /ready (readiness)
```

**Run database migrations**:
```bash
make docker-run-migrate

# Runs embedded migrations against configured database
```

**Custom configuration**:
```bash
# Run with custom environment variables
docker run --rm -p 8080:8080 \
  -e DB_DRIVER=postgres \
  -e DB_CONNECTION_STRING="postgres://user:pass@localhost:5432/db?sslmode=disable" \
  -e MASTER_KEY_PROVIDER=plaintext \
  -e MASTER_KEY_PLAINTEXT=your-base64-encoded-32-byte-key \
  allisson/secrets:latest server
```

**Common patterns**:
```bash
# Run with environment file
docker run --rm -p 8080:8080 --env-file .env allisson/secrets:latest server

# Run with read-only filesystem (security hardening)
docker run --rm -p 8080:8080 --read-only \
  -v /tmp \
  --env-file .env \
  allisson/secrets:latest server

# Verify version
docker run --rm allisson/secrets:latest --version
```

### Docker Variables

| Variable | Default | Description | Override Example |
|----------|---------|-------------|------------------|
| `DOCKER_REGISTRY` | `allisson` | Docker registry namespace | `make docker-build DOCKER_REGISTRY=myregistry.io/myorg` |
| `DOCKER_IMAGE` | `$(DOCKER_REGISTRY)/secrets` | Full image name | Auto-computed from `DOCKER_REGISTRY` |
| `DOCKER_TAG` | `latest` | Default image tag | `make docker-build DOCKER_TAG=stable` |
| `VERSION` | Auto-detected | Application version | `make docker-build VERSION=v1.0.0` |
| `BUILD_DATE` | Auto-computed | ISO 8601 build timestamp | Auto-computed (not overridable) |
| `COMMIT_SHA` | Auto-detected | Git commit hash | Auto-detected (not overridable) |

**Version detection logic**:
1. **Git tag** (if available): `git describe --tags --always --dirty` → e.g., `v0.10.0`
2. **Commit hash** (if no tag): e.g., `abc123d`
3. **Fallback**: `"dev"` (if git not available)

**Examples**:
```bash
# Default: uses auto-detected version
make docker-build
# → allisson/secrets:latest, allisson/secrets:v0.10.0

# Custom registry
make docker-build DOCKER_REGISTRY=ghcr.io/myorg
# → ghcr.io/myorg/secrets:latest, ghcr.io/myorg/secrets:v0.10.0

# Force version for testing
make docker-build VERSION=v0.9.0-beta1
# → allisson/secrets:latest, allisson/secrets:v0.9.0-beta1
```

## Docker Compose

### Test Databases

The project uses docker-compose to manage PostgreSQL and MySQL test databases for integration testing.

**Start test databases and run tests**:
```bash
make test-with-db
# Starts databases → runs tests → stops databases

# Manual control:
make test-db-up      # Start databases only
make test            # Run tests
make test-db-down    # Stop and remove databases
```

**Database services** (`docker-compose.test.yml`):
- **postgres-test**: PostgreSQL 16 on port 5433
- **mysql-test**: MySQL 8.0 on port 3307

Both services include health checks and auto-restart on failure.

**Common operations**:
```bash
# View logs
docker compose -f docker-compose.test.yml logs -f postgres-test

# Check service status
docker compose -f docker-compose.test.yml ps

# Restart specific service
docker compose -f docker-compose.test.yml restart mysql-test

# Clean up volumes
docker compose -f docker-compose.test.yml down -v
```

**When to use**: Integration tests that require actual database connections (e.g., repository tests, migration tests).

## Development Databases

For local development, use standalone Docker containers for databases (alternative to docker-compose).

**PostgreSQL**:
```bash
make dev-postgres
# Runs: postgres:16-alpine on port 5432
# Connection string: See .env.example
```

**MySQL**:
```bash
make dev-mysql
# Runs: mysql:8.0 on port 3306
# Connection string: See .env.example
```

**Stop all dev databases**:
```bash
make dev-stop
```

**When to use**:
- **Development databases** (`dev-postgres`, `dev-mysql`): Local development, manual testing, running the app locally
- **Test databases** (`test-with-db`): Automated integration tests via `make test-with-db`

**Key differences**:
- Dev databases run on **standard ports** (5432, 3306)
- Test databases run on **alternate ports** (5433, 3307) to avoid conflicts
- Test databases are **ephemeral** (cleaned up after tests)
- Dev databases **persist** until manually stopped

## Documentation Validation

All documentation changes MUST be validated before committing.

**Lint documentation**:
```bash
make docs-lint
```

This command checks:
- Markdown syntax and formatting (markdownlint-cli2)
- Code examples validation (`docs-check-examples`)
- Metadata consistency (`docs-check-metadata`)
- Release tag verification (`docs-check-release-tags`)

**Note**: Always run `make docs-lint` after updating any `.md` files in the `docs/` directory or root documentation files.

## Code Style Guidelines

### Line Length and Formatting

- **Max line length**: 110 characters
- **Tab length**: 4 spaces
- **Auto-format**: Use `golangci-lint run -v --fix` before committing
- **Line breaking**: Chain split on dots for method chaining

### Import Organization

Imports MUST be organized in 3 sections separated by blank lines:

```go
import (
    // 1. Standard library
    "context"
    "errors"
    "fmt"
    "time"

    // 2. External dependencies
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"

    // 3. Local packages (use domain aliasing for clarity)
    "github.com/allisson/secrets/internal/database"
    authDomain "github.com/allisson/secrets/internal/auth/domain"
    authDTO "github.com/allisson/secrets/internal/auth/http/dto"
    cryptoService "github.com/allisson/secrets/internal/crypto/service"
)
```

**Local import prefix**: `github.com/allisson/secrets`

**Domain aliasing pattern**: When importing multiple packages from different domains, use aliases like `authDomain`, `transitDomain`, `cryptoService`, `authDTO`.

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Variables | camelCase | `userID`, `clientName`, `isValid` |
| Constants | PascalCase or SCREAMING_SNAKE_CASE | `DefaultTimeout`, `MAX_RETRIES` |
| Functions | PascalCase (exported), camelCase (private) | `CreateClient()`, `validateInput()` |
| Types (structs) | PascalCase | `Client`, `AuditLog`, `TransitKey` |
| Interfaces | PascalCase + descriptive | `ClientRepository`, `TokenUseCase`, `SecretService` |
| Interface methods | PascalCase | `GetByID()`, `Create()`, `Delete()` |
| Test functions | `Test` + PascalCase | `TestCreateClient`, `TestValidatePolicy` |
| Table test variables | `tt` or `tc` | `for _, tt := range tests` |
| Mock types | `Mock` + InterfaceName | `MockClientRepository` |

### Function Comments

All exported identifiers (functions, types, constants, variables) MUST have comments. Comments should describe what the code does, not how it does it.

**General Rules**:
- Start comment with the name of what you're documenting
- Use complete sentences with proper punctuation
- Use present tense ("creates", "validates", not "will create")
- End with a period
- Place comment directly above the declaration

**Exported Functions**:

```go
// Create generates and persists a new Client with a random secret.
// Returns the client ID and plain text secret. The plain secret is only returned once
// and must be securely stored by the caller.
func (uc *ClientUseCase) Create(
    ctx context.Context,
    input *domain.CreateClientInput,
) (*domain.CreateClientOutput, error)
```

**Unexported Functions** (comment when logic is non-trivial):

```go
// matchPath checks if the request path matches the policy path pattern.
// Supports three types of wildcards:
//  1. Full wildcard: "*" matches any path
//  2. Trailing wildcard: "prefix/*" matches any path starting with "prefix/"
//  3. Mid-path wildcard: "/v1/keys/*/rotate" matches paths with * as single segment
func matchPath(policyPath, requestPath string) bool
```

**Package Comments** (required, placed before package declaration):

```go
// Package usecase implements transit encryption business logic.
//
// Coordinates between cryptographic services and repositories to manage transit keys
// with versioning and envelope encryption. Uses TxManager for transactional consistency.
package usecase
```

**Type/Struct Comments**:

```go
// Client represents an authentication client with associated authorization policies.
// Clients are used to authenticate API requests and enforce access control.
type Client struct {
    ID        uuid.UUID
    Name      string
    Secret    string //nolint:gosec // hashed client secret (not plaintext)
    IsActive  bool
    Policies  []PolicyDocument
}
```

**Interface Method Comments**:

```go
type ClientRepository interface {
    // Create stores a new client in the repository.
    Create(ctx context.Context, client *domain.Client) error
    
    // Get retrieves a client by ID. Returns ErrClientNotFound if not found.
    Get(ctx context.Context, clientID uuid.UUID) (*domain.Client, error)
}
```

**Special Annotations**:
- `SECURITY:` - Security warnings or sensitive operations
- `Returns ErrXxx` - Document error conditions
- `Examples:` - Provide usage examples with bullet points
- `NOTE:` - Important implementation details

**Quick Reference**:

| Context | Pattern | Required? |
|---------|---------|-----------|
| Exported function | `// FunctionName describes what it does.` | Yes |
| Unexported function | Same format, when non-trivial | Conditional |
| Package | Multi-line above `package` statement | Yes |
| Type/Struct | `// TypeName represents...` | Yes (if exported) |
| Interface methods | Comment each method | Yes |
| Constructor | `// NewTypeName creates a new...` | Yes |
| HTTP handlers | Include route and capability requirements | Yes |

### Type Usage Patterns

**Interfaces**: Define behavior contracts, typically in the package that uses them

```go
// Repository pattern (in usecase package)
type ClientRepository interface {
    Create(ctx context.Context, client *domain.Client) error
    GetByID(ctx context.Context, id string) (*domain.Client, error)
    Update(ctx context.Context, client *domain.Client) error
    Delete(ctx context.Context, id string) error
}
```

**Structs**: Domain entities, DTOs, use cases, services

```go
// Domain entity
type Client struct {
    ID        string
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Use case with dependency injection
type ClientUseCase struct {
    repo     ClientRepository
    txMgr    database.TxManager
}
```

### Error Handling

**Error types**: Define domain-specific errors as package-level variables

```go
var (
    ErrClientNotFound     = errors.New("client not found")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrUnauthorized       = errors.New("unauthorized")
)
```

**Error wrapping**: Use `fmt.Errorf` with `%w` to wrap errors

```go
func (uc *ClientUseCase) GetByID(ctx context.Context, id string) (*domain.Client, error) {
    client, err := uc.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get client: %w", err)
    }
    return client, nil
}
```

**Error checking**: Always check errors immediately, use early returns

```go
// Good
if err != nil {
    return nil, fmt.Errorf("operation failed: %w", err)
}

// Bad - don't ignore errors
_ = someOperation()
```

## Testing Guidelines

### Test File Naming

- Test files: `*_test.go` in the same package
- Integration tests: `test/integration/`
- Table-driven tests: Preferred pattern

### Table-Driven Test Pattern

```go
func TestCreateClient(t *testing.T) {
    tests := []struct {
        name    string
        input   *domain.Client
        wantErr bool
        errType error
    }{
        {
            name:    "valid client",
            input:   &domain.Client{Name: "test"},
            wantErr: false,
        },
        {
            name:    "empty name",
            input:   &domain.Client{Name: ""},
            wantErr: true,
            errType: ErrInvalidInput,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CreateClient(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errType != nil {
                    assert.ErrorIs(t, err, tt.errType)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Test Assertions

Use `testify/assert` and `testify/require`:
- `assert.*`: Continues test on failure
- `require.*`: Stops test on failure

```go
require.NoError(t, err) // Stop if error
assert.Equal(t, expected, actual)
assert.NotNil(t, result)
assert.True(t, condition)
```

### Mocks

- Generate mocks using mockery: `make mocks`
- Configuration: `.mockery.yaml`
- Mock location: `internal/package/mocks/mocks.go`
- Mock naming: `Mock{InterfaceName}`

## Common Patterns

### Dependency Injection

Use constructor functions with interface dependencies:

```go
func NewClientUseCase(repo ClientRepository, txMgr database.TxManager) *ClientUseCase {
    return &ClientUseCase{
        repo:  repo,
        txMgr: txMgr,
    }
}
```

### Repository Pattern

Repositories handle data persistence, typically implemented with SQL:

```go
type clientRepository struct {
    db *sql.DB
}

func (r *clientRepository) Create(ctx context.Context, client *domain.Client) error {
    query := `INSERT INTO clients (id, name, created_at) VALUES ($1, $2, $3)`
    _, err := r.db.ExecContext(ctx, query, client.ID, client.Name, client.CreatedAt)
    return err
}
```

### HTTP Handlers (Gin)

```go
func (h *ClientHandler) Create(c *gin.Context) {
    var req dto.CreateClientRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httputil.RespondError(c, http.StatusBadRequest, err)
        return
    }

    client, err := h.useCase.Create(c.Request.Context(), &req)
    if err != nil {
        httputil.RespondError(c, http.StatusInternalServerError, err)
        return
    }

    c.JSON(http.StatusCreated, client)
}
```

## Security Notes

- Never commit secrets to `.env` files (use `.env.example`)
- Use KMS providers for production (not plaintext master keys)
- Always validate user input using `validation` package
- Use parameterized queries (never string concatenation for SQL)
- Follow principle of least privilege for client policies

## Additional Resources

- **Makefile**: Run `make help` for all available commands
- **Configuration**: See `.env.example` for all environment variables
- **Architecture docs**: `docs/concepts/architecture.md`
- **API docs**: `docs/api/` directory
- **Contributing**: `docs/contributing.md`
