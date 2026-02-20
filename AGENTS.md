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

# Regenerate mock implementations
make mocks              # Regenerates all mocks using mockery v3
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

**Docstring Format**: Use the **enhanced compact format** consistently across the codebase.

**Package Documentation**: Start with concise package comment (1-2 lines):
```go
// Package domain defines core cryptographic domain models for envelope encryption.
// Implements Master Key → KEK → DEK → Data hierarchy with AESGCM and ChaCha20 support.
package domain
```

**Function Comments**: 
- Start with function name and concise description (1-2 sentences)
- Include important context inline without formal "Parameters:" or "Returns:" sections
- Document error cases and security notes inline
- Use bullet lists for patterns or special cases when needed
- Focus on "what" and "why", not implementation details

**Compact Format Examples:**

Simple function:
```go
// Create generates and persists a new KEK using the active master key.
// Returns ErrMasterKeyNotFound if the active master key is not in the chain.
func (k *kekUseCase) Create(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error
```

Function with security notes:
```go
// Authenticate validates a token hash and returns the associated client. Validates token
// is not expired/revoked and client is active. Returns ErrInvalidCredentials for
// invalid/expired/revoked tokens or missing clients to prevent enumeration attacks.
// Returns ErrClientInactive if the client is not active. All time comparisons use UTC.
func (t *tokenUseCase) Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error)
```

Function with patterns:
```go
// AuthorizationMiddleware enforces capability-based authorization for authenticated clients.
//
// MUST be used after AuthenticationMiddleware. Retrieves authenticated client from context,
// extracts request path, and checks if Client.IsAllowed(path, capability) permits access.
//
// Path Matching:
//   - Exact: "/secrets/mykey" matches policy "/secrets/mykey"
//   - Wildcard: "*" matches all paths
//   - Prefix: "secret/*" matches paths starting with "secret/"
//
// Returns:
//   - 401 Unauthorized: No authenticated client in context
//   - 403 Forbidden: Insufficient permissions
func AuthorizationMiddleware(capability authDomain.Capability, logger *slog.Logger) gin.HandlerFunc
```

**When to Include Details:**
- Security implications (timing attacks, enumeration, key zeroing)
- Error cases and return conditions
- Transaction behavior
- Special requirements or constraints
- Wildcard patterns or matching rules

**What to Avoid:**
- Step-by-step implementation details (e.g., "1. Do X, 2. Do Y, 3. Do Z")
- Redundant descriptions that simply restate the code
- Formal "Parameters:" and "Returns:" sections (integrate inline instead)
- Excessive examples unless for complex public APIs

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

**Mocks**: Generate using mockery v3 (.mockery.yaml configuration):
```bash
make mocks
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

## CLI Commands Structure

The application uses **urfave/cli v3** for command-line interface with commands organized in separate files.

### Directory Structure
```
cmd/app/
├── commands/           # Command implementations package
│   ├── helpers.go      # Unexported helper functions (closeContainer, closeMigrate)
│   ├── server.go       # RunServer() - HTTP server command
│   ├── migrations.go   # RunMigrations() - Database migration command
│   ├── master_key.go   # RunCreateMasterKey() - Master key generation command
│   ├── create_kek.go   # RunCreateKek() - KEK creation command (+ parseAlgorithm helper)
│   └── rotate_kek.go   # RunRotateKek() - KEK rotation command
└── main.go             # CLI setup and routing only (~87 lines)
```

### Command Organization

**Exported Functions**: Command entry points are exported with `Run` prefix (e.g., `RunServer`, `RunMigrations`)

**Unexported Helpers**: Shared utilities remain package-private (e.g., `closeContainer`, `parseAlgorithm`)

**Single Responsibility**: Each command lives in its own file for better maintainability

**Shared Logic**: Common algorithm parsing and cleanup functions are reused across commands

### Command Implementation Pattern

```go
// Package commands contains CLI command implementations.
package commands

import (
    "context"
    "fmt"
    "log/slog"
    
    "github.com/allisson/secrets/internal/app"
    "github.com/allisson/secrets/internal/config"
)

// RunCommandName performs the command operation.
// Brief description of what the command does and any requirements.
func RunCommandName(ctx context.Context, args string) error {
    // Load configuration
    cfg := config.Load()
    
    // Create DI container
    container := app.NewContainer(cfg)
    logger := container.Logger()
    
    // Ensure cleanup on exit
    defer closeContainer(container, logger)
    
    // Command implementation
    // ...
    
    return nil
}

// unexported helper functions shared across commands
func closeContainer(container *app.Container, logger *slog.Logger) {
    if err := container.Shutdown(context.Background()); err != nil {
        logger.Error("failed to shutdown container", slog.Any("error", err))
    }
}
```

### CLI Setup in main.go

The `main.go` file contains only CLI definitions and routes to command functions:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/urfave/cli/v3"
    
    "github.com/allisson/secrets/cmd/app/commands"
)

func main() {
    cmd := &cli.Command{
        Name:    "app",
        Usage:   "Application description",
        Version: "1.0.0",
        Commands: []*cli.Command{
            {
                Name:  "server",
                Usage: "Start the HTTP server",
                Action: func(ctx context.Context, cmd *cli.Command) error {
                    return commands.RunServer(ctx)
                },
            },
            // Additional commands...
        },
    }
    
    if err := cmd.Run(context.Background(), os.Args); err != nil {
        slog.Error("application error", slog.Any("error", err))
        os.Exit(1)
    }
}
```

### Available Commands

**Server Commands:**
- `app server` - Start HTTP server with graceful shutdown
- `app migrate` - Run database migrations (PostgreSQL or MySQL)

**Cryptographic Key Management:**
- `app create-master-key [--id <key-id>]` - Generate new 32-byte master key
- `app create-kek [--algorithm aes-gcm|chacha20-poly1305]` - Create initial KEK
- `app rotate-kek [--algorithm aes-gcm|chacha20-poly1305]` - Rotate existing KEK

**Audit Log Operations:**
- `app clean-audit-logs --days <days> [--dry-run] [--format text|json]` - Delete old audit logs or preview count

### Command Testing

When adding new commands:
1. Create new file in `cmd/app/commands/` with `Run<CommandName>` function
2. Add command definition to `main.go` CLI setup
3. Verify with `make build && ./bin/app --help`
4. Test command execution: `./bin/app <command-name>`

## HTTP Layer with Gin

### Server Setup

The project uses **Gin v1.11.0** as the web framework with custom slog-based middleware:

```go
// Create Gin engine without default middleware
router := gin.New()

// Apply custom middleware
router.Use(gin.Recovery())                   // Gin's panic recovery
router.Use(requestid.New(requestid.WithGenerator(func() string {
    return uuid.Must(uuid.NewV7()).String()
})))                                         // Request ID with UUIDv7
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
- Request ID tracking with UUIDv7 (`X-Request-Id` header)

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
            slog.String("request_id", requestid.Get(c)),
        )
    }
}
```

**Request ID Tracking:**
- Every HTTP request automatically generates a unique UUIDv7 request ID
- Request ID is included in `X-Request-Id` response header
- Request ID is logged with every HTTP request for tracing
- Handlers can access request ID using `requestid.Get(c)` for distributed tracing

Example log output with request ID:
```json
{
  "time": "2026-02-12T10:30:45Z",
  "level": "INFO",
  "msg": "http request",
  "method": "GET",
  "path": "/api/v1/secrets",
  "status": 200,
  "duration": "15ms",
  "client_ip": "192.168.1.100",
  "request_id": "01933e4a-7890-7abc-def0-123456789abc"
}
```

Apply middleware globally or per route group:
```go
// Global middleware (in order)
router.Use(gin.Recovery())
router.Use(requestid.New(requestid.WithGenerator(func() string {
    return uuid.Must(uuid.NewV7()).String()
})))
router.Use(CustomLoggerMiddleware(logger))

// Per-group middleware
v1 := router.Group("/api/v1")
v1.Use(authMiddleware)
```

### Rate Limiting Middleware

The project implements two types of rate limiting middleware to protect against abuse:

#### 1. Client-Based Rate Limiting (Authenticated Endpoints)

**File:** `/internal/auth/http/rate_limit_middleware.go`

**Purpose:** Protects authenticated endpoints from abuse by limiting requests per authenticated client.

**Usage:**
```go
// Create middleware with configuration
rateLimitMiddleware := authHTTP.RateLimitMiddleware(
    cfg.RateLimitRequestsPerSec,  // e.g., 10.0 requests/second
    cfg.RateLimitBurst,            // e.g., 20 burst capacity
    logger,
)

// Apply to authenticated route groups
clients := v1.Group("/clients")
clients.Use(authMiddleware)        // Must come first
clients.Use(rateLimitMiddleware)   // Rate limit per client
```

**Key Features:**
- **Requires authentication:** Must be used after `AuthenticationMiddleware`
- **Per-client limits:** Each authenticated client (by client ID) gets independent rate limiter
- **Token bucket algorithm:** Uses `golang.org/x/time/rate` for smooth rate limiting
- **Automatic cleanup:** Removes stale limiters after 1 hour of inactivity
- **Configurable:** Controlled by `RATE_LIMIT_ENABLED`, `RATE_LIMIT_REQUESTS_PER_SEC`, `RATE_LIMIT_BURST`

**Response:**
- Returns `429 Too Many Requests` with `Retry-After` header when limit exceeded
- Error response: `{"error": "rate_limit_exceeded", "message": "Too many requests. Please retry after the specified delay."}`

#### 2. IP-Based Rate Limiting (Unauthenticated Endpoints)

**File:** `/internal/auth/http/token_rate_limit_middleware.go`

**Purpose:** Protects unauthenticated endpoints (e.g., token issuance) from credential stuffing and brute force attacks.

**Usage:**
```go
// Create middleware with configuration
tokenRateLimitMiddleware := authHTTP.TokenRateLimitMiddleware(
    cfg.RateLimitTokenRequestsPerSec,  // e.g., 5.0 requests/second
    cfg.RateLimitTokenBurst,            // e.g., 10 burst capacity
    logger,
)

// Apply to unauthenticated endpoints
if tokenRateLimitMiddleware != nil {
    v1.POST("/token", tokenRateLimitMiddleware, tokenHandler.IssueTokenHandler)
}
```

**Key Features:**
- **No authentication required:** Works on unauthenticated endpoints
- **Per-IP limits:** Each IP address gets independent rate limiter
- **Automatic IP detection:** Uses `c.ClientIP()` which handles:
  - `X-Forwarded-For` header (takes first IP)
  - `X-Real-IP` header
  - Direct connection remote address
- **Token bucket algorithm:** Uses `golang.org/x/time/rate` for smooth rate limiting
- **Automatic cleanup:** Removes stale limiters after 1 hour of inactivity
- **Configurable:** Controlled by `RATE_LIMIT_TOKEN_ENABLED`, `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC`, `RATE_LIMIT_TOKEN_BURST`

**Response:**
- Returns `429 Too Many Requests` with `Retry-After` header when limit exceeded
- Error response: `{"error": "rate_limit_exceeded", "message": "Too many token requests from this IP. Please retry after the specified delay."}`

**Security Considerations:**

*Strengths:*
- Protects against credential stuffing and brute force attacks
- Stricter default limits (5 req/sec, burst 10) than authenticated endpoints
- No overhead on authenticated endpoints

*Limitations & Mitigations:*
- **Shared IPs (NAT, corporate proxies):** May affect legitimate users behind same IP
  - Mitigation: Reasonable burst capacity (10 requests) handles legitimate retries
  - Mitigation: Can be disabled via `RATE_LIMIT_TOKEN_ENABLED=false` if needed
- **IP Spoofing via X-Forwarded-For:** Attacker could rotate IPs in header
  - Mitigation: Configure Gin's trusted proxy settings in production
  - Mitigation: Deploy behind proper reverse proxy/load balancer

**Configuration Example (.env):**
```bash
# Authenticated endpoint rate limiting (per client)
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20

# Token endpoint rate limiting (per IP, unauthenticated)
RATE_LIMIT_TOKEN_ENABLED=true
RATE_LIMIT_TOKEN_REQUESTS_PER_SEC=5.0
RATE_LIMIT_TOKEN_BURST=10
```

**Testing:**
Both middleware implementations include comprehensive test coverage:
- Requests within limit allowed
- Requests exceeding limit blocked with 429
- Retry-After header present
- Independent limits per client/IP
- Burst capacity handling
- Automatic cleanup of stale entries

**Reference:**
- Client-based: `/internal/auth/http/rate_limit_middleware.go` and `rate_limit_middleware_test.go`
- IP-based: `/internal/auth/http/token_rate_limit_middleware.go` and `token_rate_limit_middleware_test.go`

## Authentication & Authorization HTTP Layer

### HTTP Handler Organization Pattern

The HTTP layer follows a structured organization pattern that separates concerns by domain responsibility:

**Directory Structure:**
```
internal/auth/http/
├── client_handler.go          # ClientHandler - manages API clients (CRUD)
├── client_handler_test.go     # ClientHandler integration tests
├── token_handler.go           # TokenHandler - token issuance
├── token_handler_test.go      # TokenHandler integration tests
├── middleware.go              # Authentication & authorization middleware
├── middleware_test.go         # Middleware tests
├── context.go                 # Context helper functions (WithClient, GetClient)
├── test_helpers.go            # Shared test utilities (createTestContext)
├── dto/                       # Data Transfer Objects package
│   ├── request.go             # Request DTOs with validation
│   ├── request_test.go        # Request validation tests
│   ├── response.go            # Response DTOs with mapping functions
│   └── response_test.go       # Response mapping tests
└── mocks/                     # Manual mocks (separate from generated mocks)
    └── token_usecase.go       # MockTokenUseCase
```

**Handler Organization Guidelines:**

**When to Split Handlers:**
- Split by **domain responsibility**, not by CRUD operation
- Example: `ClientHandler` (client management) vs `TokenHandler` (token issuance)
- Each handler struct manages one domain concept with multiple HTTP methods
- Avoid creating separate handlers for each HTTP method (e.g., don't create `CreateClientHandler`, `UpdateClientHandler`)

**DTO Package Conventions:**

1. **Separation by Direction:**
   - `request.go` - Request DTOs and validation logic
   - `response.go` - Response DTOs and mapping functions

2. **Validation Placement:**
   - Request DTOs include `Validate() error` methods
   - Use `github.com/jellydator/validation` for validation rules
   - Unexported helper functions (e.g., `validatePolicyDocument()`) stay in `request.go`

3. **Mapping Functions:**
   - Response mapping functions live in `response.go`
   - Export mapping functions that handlers need (e.g., `MapClientToResponse()`)
   - Keep unexported helpers for internal transformations

4. **Testing:**
   - Create corresponding test files: `request_test.go`, `response_test.go`
   - Test validation logic in isolation from HTTP handlers
   - Test mapping functions with domain model fixtures

**Test Helper Guidelines:**

1. **Shared Utilities:**
   - Extract common test setup to `test_helpers.go` (not `*_test.go` suffix)
   - Example: `createTestContext(method, path, body) (*gin.Context, *httptest.ResponseRecorder)`
   - Reuse across all handler test files

2. **Mock Organization:**
   - Manual mocks go in `mocks/` subdirectory (e.g., `mocks/token_usecase.go`)
   - Generated mocks (via mockery v3) are consolidated in `mocks/mocks.go` per package
   - Keep manual and generated mocks separate to avoid conflicts

**Example Handler Structure:**

```go
// client_handler.go
package http

import (
    authUseCase "github.com/allisson/secrets/internal/auth/usecase"
    authDTO "github.com/allisson/secrets/internal/auth/http/dto"
)

type ClientHandler struct {
    clientUseCase   authUseCase.ClientUseCase
    auditLogUseCase authUseCase.AuditLogUseCase
}

func (h *ClientHandler) CreateHandler(c *gin.Context) {
    var req authDTO.CreateClientRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        httputil.HandleValidationErrorGin(c, err, h.logger)
        return
    }
    
    if err := req.Validate(); err != nil {
        httputil.HandleValidationErrorGin(c, validation.WrapValidationError(err), h.logger)
        return
    }
    
    client, secret, err := h.clientUseCase.Create(c.Request.Context(), ...)
    if err != nil {
        httputil.HandleErrorGin(c, err, h.logger)
        return
    }
    
    response := authDTO.CreateClientResponse{
        ID:     client.ID.String(),
        Secret: secret,
    }
    c.JSON(http.StatusCreated, response)
}
```

**Key Patterns:**
- Import DTOs with alias: `authDTO "github.com/allisson/secrets/internal/auth/http/dto"`
- Use `authDTO.CreateClientRequest` for request binding
- Call `req.Validate()` after binding
- Use `authDTO.MapClientToResponse(client)` for response mapping
- Keep handlers thin - delegate business logic to use cases

### Authentication Middleware

The project implements Bearer token authentication via `AuthenticationMiddleware`:

```go
// AuthenticationMiddleware validates Bearer tokens and sets authenticated client in context
func AuthenticationMiddleware(tokenUseCase authUseCase.TokenUseCase, logger *slog.Logger) gin.HandlerFunc
```

**Behavior:**
- Extracts token from `Authorization` header (case-insensitive "Bearer" prefix: `bearer`, `Bearer`, `BEARER`)
- Validates token hash via `TokenUseCase.Authenticate()` which checks:
  - Token exists and is not expired/revoked
  - Associated client exists and is active
  - All time comparisons use UTC
- Sets authenticated client in context via `authHTTP.WithClient(c, client)`
- Returns 401 Unauthorized for:
  - Missing Authorization header
  - Malformed header (not "Bearer <token>")
  - Invalid/expired/revoked token
  - Inactive client
  - Database errors (prevents enumeration attacks)

**Usage:**
```go
// Apply to routes requiring authentication
router.POST("/v1/clients", authenticationMiddleware, handler)
```

**Reference:** `/internal/auth/http/middleware.go` (lines 15-74)

**Context Helpers:**
- `authHTTP.WithClient(c, client)` - Store client in context
- `authHTTP.GetClient(c)` - Retrieve client from context
- See `/internal/auth/http/context.go` for all context helpers

### Authorization Middleware

Enforces capability-based authorization via `AuthorizationMiddleware`:

```go
// AuthorizationMiddleware checks if authenticated client has required capability for the request path
func AuthorizationMiddleware(capability authDomain.Capability, logger *slog.Logger) gin.HandlerFunc
```

**Requirements:**
- **MUST** be used after `AuthenticationMiddleware`
- Authenticated client must be present in context

**Behavior:**
- Retrieves authenticated client from context via `authHTTP.GetClient(c)`
- Extracts request path from `c.Request.URL.Path`
- Stores path and capability in context for audit logging
- Checks `client.IsAllowed(path, capability)` which implements path matching:
  - **Exact match:** `/secrets/mykey` matches policy path `/secrets/mykey`
  - **Wildcard:** `*` matches all paths
  - **Prefix:** `secrets/*` matches paths starting with `secrets/`
- Returns 401 Unauthorized if no authenticated client in context
- Returns 403 Forbidden if client lacks required capability for path

**Usage:**
```go
// Apply with specific capability per route
router.POST("/v1/clients", authMiddleware, authzMiddleware(authDomain.WriteCapability), handler)
router.GET("/v1/clients/:id", authMiddleware, authzMiddleware(authDomain.ReadCapability), handler)
router.DELETE("/v1/clients/:id", authMiddleware, authzMiddleware(authDomain.DeleteCapability), handler)
```

**Available Capabilities:**
- `ReadCapability` - View resources
- `WriteCapability` - Create/update resources
- `DeleteCapability` - Delete resources
- `EncryptCapability` - Encrypt data
- `DecryptCapability` - Decrypt data
- `RotateCapability` - Rotate keys

**Reference:** `/internal/auth/http/middleware.go` (lines 76-130)

### Client Management Handler Pattern

Client management handlers follow this pattern:

```go
// ClientHandler handles HTTP requests for client management
type ClientHandler struct {
    clientUseCase   authUseCase.ClientUseCase
    auditLogUseCase authUseCase.AuditLogUseCase
}

func NewClientHandler(clientUseCase authUseCase.ClientUseCase, auditLogUseCase authUseCase.AuditLogUseCase) *ClientHandler
```

**Request DTOs:**
```go
type CreateClientRequest struct {
    Name           string                    `json:"name" binding:"required"`
    IsActive       bool                      `json:"is_active"`
    PolicyDocument *authDomain.PolicyDocument `json:"policy_document" binding:"required"`
}

type UpdateClientRequest struct {
    Name           string                    `json:"name" binding:"required"`
    IsActive       bool                      `json:"is_active"`
    PolicyDocument *authDomain.PolicyDocument `json:"policy_document" binding:"required"`
}
```

**Response DTOs:**
```go
// CreateClientResponse includes the client secret (only returned on creation)
type CreateClientResponse struct {
    ID     string `json:"id"`
    Secret string `json:"secret"`
}

// ClientResponse excludes the secret for Get/Update operations
type ClientResponse struct {
    ID             string                    `json:"id"`
    Name           string                    `json:"name"`
    IsActive       bool                      `json:"is_active"`
    PolicyDocument *authDomain.PolicyDocument `json:"policy_document"`
    CreatedAt      time.Time                 `json:"created_at"`
    UpdatedAt      time.Time                 `json:"updated_at"`
}
```

**Handler Methods:**
- `CreateHandler(c *gin.Context)` - POST, returns 201 with ID and secret
- `GetHandler(c *gin.Context)` - GET by UUID param, returns 200 with client (no secret)
- `UpdateHandler(c *gin.Context)` - PUT by UUID param, returns 200 with updated client
- `DeleteHandler(c *gin.Context)` - DELETE by UUID param, returns 204 No Content

**Key Patterns:**

**UUID Extraction from URL:**
```go
id, err := uuid.Parse(c.Param("id"))
if err != nil {
    httputil.HandleValidationErrorGin(c, validation.WrapValidationError(err), h.logger)
    return
}
```

**Policy Document Validation:**
```go
// validatePolicyDocument ensures policy document has valid structure
func validatePolicyDocument(doc *authDomain.PolicyDocument) error {
    if doc == nil {
        return errors.New("policy_document is required")
    }
    for _, policy := range doc.Policies {
        if policy.Path == "" {
            return errors.New("policy path cannot be empty")
        }
        if len(policy.Capabilities) == 0 {
            return errors.New("policy capabilities cannot be empty")
        }
    }
    return nil
}
```

**DELETE Handler Pattern:**
```go
// DELETE must use c.Data() to properly set 204 No Content with empty body
if err := h.clientUseCase.Delete(c.Request.Context(), id); err != nil {
    httputil.HandleErrorGin(c, err, h.logger)
    return
}
c.Data(http.StatusNoContent, "application/json", nil)  // NOT c.Status()
```

**Reference:** 
- Implementation: `/internal/auth/http/client_handler.go` and `/internal/auth/http/token_handler.go`
- Tests: `/internal/auth/http/client_handler_test.go` and `/internal/auth/http/token_handler_test.go`
- DTOs: `/internal/auth/http/dto/` package (request.go, response.go)
- Test Helpers: `/internal/auth/http/test_helpers.go`
- Mocks: `/internal/auth/http/mocks/token_usecase.go`

### Route Registration with Authentication & Authorization

Client management routes are registered in `SetupRouter()` with middleware chaining:

```go
func (s *Server) SetupRouter(
    clientHandler *authHTTP.ClientHandler,
    tokenUseCase authUseCase.TokenUseCase,
    tokenService authService.TokenService,
    auditLogUseCase authUseCase.AuditLogUseCase,
) {
    // Create middleware instances
    authMiddleware := authHTTP.AuthenticationMiddleware(tokenUseCase, s.logger)
    auditMiddleware := authHTTP.AuditLogMiddleware(auditLogUseCase, s.logger)
    
    // Register client management routes under /v1/clients
    v1 := s.router.Group("/v1")
    v1.Use(auditMiddleware)  // Apply audit logging to all v1 routes
    {
        clients := v1.Group("/clients")
        {
            // POST /v1/clients - Create client (requires WriteCapability)
            clients.POST("", 
                authMiddleware,
                authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, s.logger),
                clientHandler.CreateHandler,
            )
            
            // GET /v1/clients/:id - Get client (requires ReadCapability)
            clients.GET("/:id",
                authMiddleware,
                authHTTP.AuthorizationMiddleware(authDomain.ReadCapability, s.logger),
                clientHandler.GetHandler,
            )
            
            // PUT /v1/clients/:id - Update client (requires WriteCapability)
            clients.PUT("/:id",
                authMiddleware,
                authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, s.logger),
                clientHandler.UpdateHandler,
            )
            
            // DELETE /v1/clients/:id - Delete client (requires DeleteCapability)
            clients.DELETE("/:id",
                authMiddleware,
                authHTTP.AuthorizationMiddleware(authDomain.DeleteCapability, s.logger),
                clientHandler.DeleteHandler,
            )
        }
    }
}
```

**Middleware Execution Order:**
1. Global middleware (Recovery, RequestID, CustomLogger)
2. Route group middleware (AuditLog)
3. Route-specific middleware (Authentication → Authorization)
4. Handler

**Capability Mapping:**
- `POST /v1/clients` → `WriteCapability` (create new client)
- `GET /v1/clients/:id` → `ReadCapability` (view client details)
- `PUT /v1/clients/:id` → `WriteCapability` (modify client)
- `DELETE /v1/clients/:id` → `DeleteCapability` (remove client)

**Reference:** `/internal/http/server.go` (SetupRouter method)

## KMS Service Implementation

The project supports KMS (Key Management Service) integration for encrypting master keys at rest using external providers. KMS functionality follows interface segregation principles with the domain layer defining minimal interfaces and the service layer providing concrete implementations.

### Interface Segregation Pattern

**Domain Layer** (`internal/crypto/domain/master_key.go`):
```go
// Minimal interfaces defined by domain - no external dependencies
type KMSService interface {
    OpenKeeper(ctx context.Context, keyURI string) (KMSKeeper, error)
}

type KMSKeeper interface {
    Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
    Close() error
}
```

**Service Layer** (`internal/crypto/service/kms_service.go`):
- Implements `KMSService` using `gocloud.dev/secrets`
- Imports all KMS provider drivers (gcpkms, awskms, azurekeyvault, hashivault, localsecrets)
- Returns `*secrets.Keeper` which naturally implements `KMSKeeper` (duck typing)

**Type Compatibility:**
- `*secrets.Keeper` from gocloud.dev implements both `Decrypt()` and `Close()` methods
- No wrapper types needed - direct type assertion works in implementation code

**Reference:** `/internal/crypto/service/kms_service.go` and `/internal/crypto/domain/master_key.go:114-128`

### Testing with localsecrets Provider

**Always use `localsecrets` provider for tests** - no external dependencies or credentials required.

**Generate test KMS key:**
```go
func generateLocalSecretsKMSKey(t *testing.T) string {
    t.Helper()
    key := make([]byte, 32)
    _, err := rand.Read(key)
    require.NoError(t, err)
    return "base64key://" + base64.URLEncoding.EncodeToString(key)
}
```

**Type assertion for Encrypt method** (not part of domain interface):
```go
keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
require.NoError(t, err)

// Type assert to access Encrypt method for tests
keeper, ok := keeperInterface.(*secrets.Keeper)
require.True(t, ok, "keeper should be *secrets.Keeper")

ciphertext, err := keeper.Encrypt(ctx, plaintext)
```

**Mock implementations must return copies** to avoid issues when ciphertext is zeroed:
```go
// BAD - returns slice of input (will be zeroed)
func (m *MockKMSKeeper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
    return ciphertext, nil
}

// GOOD - returns a copy
func (m *MockKMSKeeper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
    result := make([]byte, len(ciphertext))
    copy(result, ciphertext)
    return result, nil
}
```

**Reference:** `/internal/crypto/service/kms_service_test.go` and `/test/integration/api_test.go` (KMS helpers)

### Error Handling for Close() Calls

**All `Close()` calls MUST check errors** (enforced by golangci-lint errcheck).

**Production code pattern** (with logging):
```go
defer func() {
    if closeErr := keeper.Close(); closeErr != nil {
        logger.Error("failed to close KMS keeper", slog.Any("error", closeErr))
    }
}()
```

**Test code pattern** (with assertions):
```go
defer func() {
    assert.NoError(t, keeper.Close())
}()
```

**CLI code pattern** (with user-facing message):
```go
defer func() {
    if closeErr := keeperInterface.Close(); closeErr != nil {
        fmt.Printf("Warning: failed to close KMS keeper: %v\n", closeErr)
    }
}()
```

**Reference:** `/internal/crypto/domain/master_key.go:213-217` and `/cmd/app/commands/master_key.go:58-62`

### Memory Safety and Performance

**Startup-only decryption:**
- KMS operations happen only at application startup
- Master keys decrypted into memory once via `LoadMasterKeyChain()`
- No per-operation KMS calls (performance optimization)

**Memory cleanup:**
- Master key zeroing handled by existing `MasterKeyChain.Close()`
- KEK chain similarly zeroed via `KekChain.Close()`
- No additional cleanup needed for KMS-decrypted keys

**Ownership transfer:**
- Decrypted key data ownership transfers to `MasterKeyChain`
- Original slices can be safely reused by KMS keeper
- Domain layer makes defensive copies when needed

**Reference:** `/internal/crypto/domain/master_key.go:183-285` (loadMasterKeyChainFromKMS)

### URI Masking for Security

**Use `maskKeyURI()` to redact sensitive URI components in logs:**

```go
maskedURI := maskKeyURI(cfg.KMSKeyURI)
logger.Info("opening KMS keeper",
    slog.String("kms_provider", cfg.KMSProvider),
    slog.String("kms_key_uri", maskedURI),
)
```

**Masking examples:**
- `gcpkms://projects/my-project/...` → `gcpkms://projects/***/...`
- `awskms://key-id-123?region=us-east-1` → `awskms://***?region=us-east-1`
- `azurekeyvault://vault.azure.net/keys/mykey` → `azurekeyvault://***`
- `base64key://c2VjcmV0a2V5` → `base64key://***`

**Purpose:**
- Prevents sensitive key identifiers from appearing in logs
- Preserves provider type and structure for debugging
- Retains query parameters (e.g., region) that are not sensitive

**Reference:** `/internal/crypto/domain/master_key.go:130-181` (maskKeyURI function)

### Auto-Detection Mode

**KMS vs Legacy mode determined by environment variables:**

```go
// KMS mode: both KMS_PROVIDER and KMS_KEY_URI must be set
if cfg.KMSProvider != "" && cfg.KMSKeyURI != "" {
    return loadMasterKeyChainFromKMS(ctx, cfg, kmsService, logger)
}

// Legacy mode: neither should be set
if cfg.KMSProvider == "" && cfg.KMSKeyURI == "" {
    return LoadMasterKeyChainFromEnv()
}

// Error: inconsistent configuration
return ErrKMSProviderNotSet or ErrKMSKeyURINotSet
```

**Validation:**
- Fail fast on inconsistent configuration (one set, one empty)
- Clear error messages indicating which variable is missing
- No silent fallbacks - explicit mode selection

**Reference:** `/internal/crypto/domain/master_key.go:287-315` (LoadMasterKeyChain)

## See also

- [Repository README](README.md)
- [Documentation index](docs/README.md)
- [Testing guide](docs/development/testing.md)
- [Contributing guide](docs/contributing.md)
