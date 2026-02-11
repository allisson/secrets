# Agent Guidelines for Secrets Project

This document provides essential guidelines for AI coding agents working on the Secrets project, a Go-based cryptographic key management system implementing envelope encryption with Clean Architecture principles.

## Project Overview

- **Language**: Go 1.25+
- **Web Framework**: Gin v1.11.0
- **Architecture**: Clean Architecture with Domain-Driven Design
- **Databases**: PostgreSQL 12+ and MySQL 8.0+ (dual support)
- **Pattern**: Envelope encryption (Master Key → KEK → DEK → Data)

## Build, Lint, and Test Commands

### Build Commands
```bash
make build              # Build the application binary to bin/app
make run-server         # Build and run HTTP server (port 8080)
make run-worker         # Build and run outbox event processor
make run-migrate        # Build and run database migrations
```

### Lint Commands
```bash
make lint               # Run golangci-lint with auto-fix enabled
```

The project uses golangci-lint with the following configuration (.golangci.yml):
- Default linters: standard
- Additional linters: gosec, gocritic
- Formatters: goimports, golines
- Line length: 110 characters max
- Tab width: 4 spaces
- Local import prefix: github.com/allisson/secrets

### Test Commands
```bash
# Run all tests with coverage
make test               # Runs: go test -v -race -coverprofile=coverage.out ./...

# Run tests with real databases
make test-with-db       # Starts test DBs, runs tests, stops DBs

# Individual database management
make test-db-up         # Start PostgreSQL and MySQL test containers
make test-db-down       # Stop and remove test containers

# View coverage report
make test-coverage      # Opens HTML coverage report in browser
```

### Running a Single Test
```bash
# Run a specific test function
go test -v -race -run TestFunctionName ./path/to/package

# Run a specific test with pattern matching
go test -v -race -run "TestKekUseCase_Create/Success" ./internal/crypto/usecase

# Run tests in a specific package
go test -v -race ./internal/crypto/usecase

# Run tests with coverage for a single package
go test -v -race -coverprofile=coverage.out ./internal/crypto/usecase
go tool cover -func=coverage.out
```

## Code Style Guidelines

### Package Structure and Imports

**Import Order** (enforced by goimports):
1. Standard library imports
2. External dependencies
3. Internal packages (prefixed with github.com/allisson/secrets/internal/)

**Import Aliases**:
- Use descriptive aliases for domain packages: `cryptoDomain`, `cryptoService`, `cryptoRepository`
- Use `apperrors` for `github.com/allisson/secrets/internal/errors`

Example:
```go
import (
    "context"
    "database/sql"
    
    "github.com/google/uuid"
    
    cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
    cryptoService "github.com/allisson/secrets/internal/crypto/service"
    apperrors "github.com/allisson/secrets/internal/errors"
)
```

### Architecture Layers

Follow Clean Architecture strictly:

1. **Domain Layer** (`domain/`)
   - Pure business entities and domain logic
   - No external dependencies (except UUIDs)
   - Domain-specific errors wrapping standard errors
   - Example: `Kek`, `Dek`, `MasterKey` structs

2. **Repository Layer** (`repository/`)
   - Data persistence implementations (PostgreSQL and MySQL)
   - Use `database.GetTx(ctx, db)` for transaction support
   - Wrap errors with context: `apperrors.Wrap(err, "failed to create kek")`
   - Always defer `rows.Close()` and check `rows.Err()`

3. **Use Case Layer** (`usecase/`)
   - Business logic orchestration
   - Coordinates between repositories and services
   - Defines interfaces for dependencies
   - Transaction management via `TxManager.WithTx()`

4. **Presentation Layer** (`http/`)
   - HTTP handlers using Gin web framework
   - Request/response DTOs
   - Maps domain errors to HTTP status codes
   - Input validation using jellydator/validation
   - Custom slog-based logging middleware

5. **Service Layer** (`service/`)
   - Reusable technical services (encryption, key management)
   - No business logic

### Naming Conventions

**Interfaces**: Named after behavior (e.g., `KekRepository`, `KeyManager`, `TxManager`)

**Structs**: 
- Domain entities: PascalCase (e.g., `Kek`, `MasterKey`)
- Internal implementations: lowercase with package name (e.g., `kekUseCase`, `postgresqlKekRepository`)

**Methods**: Use descriptive verbs:
- `Create`, `Update`, `List` (repositories)
- `CreateKek`, `DecryptKek`, `EncryptDek` (services)
- `Wrap`, `Unwrap`, `Rotate` (use cases)

**Variables**:
- Use full words, not abbreviations (except common ones: `ctx`, `db`, `id`, `tx`)
- Example: `masterKey` not `mk`, `kekChain` not `kc`

### Types and Interfaces

**UUIDs**: Use `google/uuid` package, prefer UUIDv7 for database IDs:
```go
id := uuid.Must(uuid.NewV7())
```

**Context**: Always pass `context.Context` as the first parameter:
```go
func Create(ctx context.Context, kek *Kek) error
```

**Error Returns**: Return errors as the last return value:
```go
func Get(id uuid.UUID) (*Kek, error)
```

### Error Handling

**Standard Errors** (internal/errors/errors.go):
- `ErrNotFound` → 404 Not Found
- `ErrConflict` → 409 Conflict
- `ErrInvalidInput` → 422 Unprocessable Entity
- `ErrUnauthorized` → 401 Unauthorized
- `ErrForbidden` → 403 Forbidden

**Domain Errors**: Wrap standard errors with context:
```go
var ErrKekNotFound = errors.Wrap(errors.ErrNotFound, "kek not found")
```

**Error Checking**:
```go
if err != nil {
    return apperrors.Wrap(err, "failed to perform operation")
}
```

**Error Comparison**: Use `errors.Is()` and `errors.As()`:
```go
if errors.Is(err, sql.ErrNoRows) {
    return domain.ErrKekNotFound
}
```

### Validation

Use `github.com/jellydator/validation` for input validation:
```go
func (d *CreateDTO) Validate() error {
    return validation.ValidateStruct(d,
        validation.Field(&d.Name, validation.Required, validation.Length(1, 255)),
        validation.Field(&d.Email, validation.Required, customValidation.Email),
    )
}
```

Wrap validation errors: `validation.WrapValidationError(err)`

### Documentation

**Package Documentation**: Start with package comment:
```go
// Package domain defines the core cryptographic domain models and types.
package domain
```

**Function Comments**: 
- Start with function name
- Explain purpose, not implementation
- Document parameters and return values
- Include usage examples for public APIs

Example:
```go
// Create generates and persists a new Key Encryption Key.
//
// This method creates the initial KEK for the system using the active master key
// from the provided keychain.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - masterKeyChain: The keychain containing the active master key
//   - alg: The encryption algorithm to use for the KEK
//
// Returns:
//   - An error if the master key is not found or KEK generation fails
func (k *kekUseCase) Create(ctx context.Context, ...) error
```

### Testing

**Test Framework**: Use `testify` for assertions and mocks

**Test Naming**: `Test<Struct>_<Method>` or `Test<Function>`
```go
func TestKekUseCase_Create(t *testing.T)
```

**Subtests**: Use descriptive names with underscores:
```go
t.Run("Success_CreateKekWithAESGCM", func(t *testing.T) { ... })
t.Run("Error_MasterKeyNotFound", func(t *testing.T) { ... })
```

**Mocks**: Generate using mockery (.mockery.yaml configuration):
```bash
go generate ./...
```

**Test Structure**:
```go
t.Run("TestName", func(t *testing.T) {
    // Setup mocks
    mockRepo := mocks.NewMockRepository(t)
    
    // Create test data
    testData := createTestData()
    
    // Setup expectations
    mockRepo.EXPECT().Method(...).Return(...).Once()
    
    // Execute
    result, err := useCase.Method(ctx, ...)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
})
```

**Integration Tests**: Use real databases (PostgreSQL and MySQL) via testutil helpers

## Additional Guidelines

- **Line Length**: Maximum 110 characters (enforced by golines)
- **Defer Usage**: Always defer cleanup operations (`Close()`, `rows.Close()`)
- **Security**: Use `Zero()` functions to clear sensitive data from memory
- **Transactions**: Use `TxManager.WithTx()` for atomic multi-step operations
- **Thread Safety**: Use `sync.Map` for concurrent access to shared data
- **Binary Data**: Store as `[]byte`, use BYTEA (PostgreSQL) or BLOB (MySQL)
- **Timestamps**: Use `time.Time` with UTC, store with timezone in PostgreSQL

## Common Patterns

### Repository Pattern with Transactions
```go
func (r *Repository) Create(ctx context.Context, entity *Entity) error {
    querier := database.GetTx(ctx, r.db)
    _, err := querier.ExecContext(ctx, query, args...)
    return err
}
```

### Use Case with Transaction
```go
return k.txManager.WithTx(ctx, func(ctx context.Context) error {
    if err := k.repo.Update(ctx, old); err != nil {
        return err
    }
    return k.repo.Create(ctx, new)
})
```

### Dependency Injection
```go
func NewUseCase(txManager TxManager, repo Repository) UseCase {
    return &useCase{txManager: txManager, repo: repo}
}
```

## HTTP Layer with Gin

### Server Setup

The project uses **Gin v1.11.0** as the web framework with custom slog-based middleware:

```go
// Create Gin engine without default middleware
router := gin.New()

// Apply custom middleware
router.Use(gin.Recovery())                    // Gin's panic recovery
router.Use(CustomLoggerMiddleware(logger))   // Custom slog logger

// Health endpoints (outside API versioning)
router.GET("/health", s.healthHandler)
router.GET("/ready", s.readinessHandler(ctx))

// API v1 routes group
v1 := router.Group("/api/v1")
{
    // Business endpoints
    v1.POST("/secrets", authMiddleware, s.createSecretHandler)
}
```

**Key Features:**
- Manual `http.Server` configuration for timeout control (ReadTimeout: 15s, WriteTimeout: 15s, IdleTimeout: 60s)
- Gin mode auto-configured from `LOG_LEVEL` environment variable (debug/release)
- Router groups for API versioning (`/api/v1`)
- Graceful shutdown support

### Handler Pattern

```go
// Handler method signature
func (s *Server) createSecretHandler(c *gin.Context) {
    var req CreateSecretRequest
    
    // 1. Parse and bind JSON
    if err := c.ShouldBindJSON(&req); err != nil {
        httputil.HandleValidationErrorGin(c, err, s.logger)
        return
    }
    
    // 2. Validate with jellydator/validation
    if err := req.Validate(); err != nil {
        httputil.HandleValidationErrorGin(c, validation.WrapValidationError(err), s.logger)
        return
    }
    
    // 3. Call use case
    result, err := s.secretUseCase.CreateOrUpdate(c.Request.Context(), req.Path, req.Value)
    if err != nil {
        httputil.HandleErrorGin(c, err, s.logger)
        return
    }
    
    // 4. Return success response
    c.JSON(http.StatusCreated, mapToResponse(result))
}
```

### Error Handling in HTTP

Use `httputil.HandleErrorGin()` to map domain errors to HTTP status codes:

```go
// Automatically maps domain errors to HTTP responses
httputil.HandleErrorGin(c, err, s.logger)

// Error mapping:
// ErrNotFound       → 404 Not Found
// ErrConflict       → 409 Conflict
// ErrInvalidInput   → 422 Unprocessable Entity
// ErrUnauthorized   → 401 Unauthorized
// ErrForbidden      → 403 Forbidden
// Unknown errors    → 500 Internal Server Error
```

### Request/Response DTOs

```go
type CreateSecretRequest struct {
    Path  string `json:"path" binding:"required"`
    Value []byte `json:"value" binding:"required"`
}

func (r *CreateSecretRequest) Validate() error {
    return validation.ValidateStruct(r,
        validation.Field(&r.Path, validation.Required, validation.Length(1, 255)),
        validation.Field(&r.Value, validation.Required),
    )
}

type SecretResponse struct {
    ID      string    `json:"id"`
    Path    string    `json:"path"`
    Version int       `json:"version"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Testing HTTP Handlers

Use Gin's test utilities for HTTP handler tests:

```go
func TestHealthHandler(t *testing.T) {
    // Set Gin to test mode
    gin.SetMode(gin.TestMode)
    
    // Create test server
    server := createTestServer()
    
    // Create test context
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
    
    // Call handler
    server.healthHandler(c)
    
    // Assert response
    assert.Equal(t, http.StatusOK, w.Code)
    var response map[string]string
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "healthy", response["status"])
}
```

**Integration Tests** (test full router):
```go
func TestRouter_HealthEndpoint(t *testing.T) {
    gin.SetMode(gin.TestMode)
    server := createTestServer()
    router := server.setupRouter(context.Background())
    
    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Middleware Pattern

Custom middleware follows Gin's signature:

```go
func CustomLoggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // Process request
        c.Next()
        
        // Log after completion
        logger.Info("http request",
            slog.String("method", c.Request.Method),
            slog.String("path", c.Request.URL.Path),
            slog.Int("status", c.Writer.Status()),
            slog.Duration("duration", time.Since(start)),
            slog.String("client_ip", c.ClientIP()),
        )
    }
}
```

Apply middleware globally or per route group:
```go
// Global middleware
router.Use(CustomLoggerMiddleware(logger))

// Per-group middleware
v1 := router.Group("/api/v1")
v1.Use(authMiddleware)
```
