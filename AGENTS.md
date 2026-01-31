# Agent Guidelines for Go Project Template

This document provides guidelines for AI coding agents working in this Go project. Follow these conventions to maintain consistency with the existing codebase.

## Build, Lint, and Test Commands

### Building
```bash
make build                    # Build the application binary
make run-server              # Build and run HTTP server
make run-worker              # Build and run background worker
make run-migrate             # Build and run database migrations
```

### Testing
```bash
make test                     # Run all tests with coverage
make test-with-db            # Start databases, run tests, stop databases (full cycle)
make test-db-up              # Start PostgreSQL (5433) and MySQL (3307) test databases
make test-db-down            # Stop test databases

# Run a single test or test file
go test -v ./internal/user/repository -run TestPostgreSQLUserRepository_Create
go test -v ./internal/user/usecase -run TestUserUseCase

# Run tests for a specific package
go test -v ./internal/user/repository
go test -v ./internal/user/usecase

# Run tests with race detection
go test -v -race ./...

# View coverage in browser
make test-coverage
```

### Linting
```bash
make lint                     # Run golangci-lint with auto-fix
golangci-lint run -v --fix   # Direct linter command
```

### Other Commands
```bash
make clean                   # Remove build artifacts and coverage files
make deps                    # Download and tidy dependencies
make help                    # Show all available make targets
```

## Project Architecture

This is a **Clean Architecture** project following **Domain-Driven Design** principles with a modular domain structure.

### Directory Structure
- `cmd/app/` - Application entry point (CLI with server/worker/migrate commands)
- `internal/`
  - `{domain}/` - Domain modules (e.g., `user/`, `outbox/`)
    - `domain/` - Entities, value objects, domain errors
    - `usecase/` - UseCase interfaces and business logic
    - `repository/` - Data persistence (MySQL and PostgreSQL implementations)
    - `http/` - HTTP handlers and DTOs
      - `dto/` - Request/response DTOs, mappers
  - `app/` - Dependency injection container
  - `errors/` - Standardized domain errors
  - `httputil/` - HTTP utilities (JSON responses, error handling)
  - `database/` - Database connection and transaction management
  - `config/` - Configuration management
  - `validation/` - Custom validation rules
  - `testutil/` - Test helper utilities

### Layer Responsibilities

**Domain Layer** (`domain/`)
- Define entities with pure business logic (no JSON tags)
- Define domain-specific errors by wrapping standard errors from `internal/errors`
- Example: `ErrUserNotFound = errors.Wrap(errors.ErrNotFound, "user not found")`

**Use Case Layer** (`usecase/`)
- Define UseCase interfaces for dependency inversion
- Implement business logic and orchestration
- Validate input using `github.com/jellydator/validation`
- Return domain errors directly without additional wrapping
- Manage transactions using `TxManager.WithTx()`

**Repository Layer** (`repository/`)
- Implement separate repositories for MySQL and PostgreSQL
- Transform infrastructure errors to domain errors (e.g., `sql.ErrNoRows` → `domain.ErrUserNotFound`)
- Use `database.GetTx(ctx, r.db)` to support transactions
- Check for unique constraint violations and return appropriate domain errors

**HTTP Layer** (`http/`)
- Define request/response DTOs with JSON tags (keep domain models pure)
- Validate DTOs using `jellydator/validation`
- Use `httputil.HandleError()` for automatic error-to-HTTP status mapping
- Use `httputil.MakeJSONResponse()` for consistent JSON responses
- Depend on UseCase interfaces, not concrete implementations

## Code Style Guidelines

### Imports
Follow this import grouping order (enforced by `goimports`):
1. Standard library packages
2. External packages (third-party)
3. Local project packages (prefixed with `github.com/allisson/secrets`)

Example:
```go
import (
    "context"
    "database/sql"
    "strings"

    "github.com/google/uuid"
    validation "github.com/jellydator/validation"

    "github.com/allisson/secrets/internal/database"
    "github.com/allisson/secrets/internal/errors"
    "github.com/allisson/secrets/internal/user/domain"
)
```

**Important:** When renaming imports, use descriptive aliases:
- `apperrors` for `internal/errors`
- `appValidation` for `internal/validation`
- `outboxDomain` for `internal/outbox/domain`

### Formatting
- **Line length**: Max 110 characters (enforced by `golines`)
- **Indentation**: Use tabs (tab-len: 4)
- **Comments**: Not shortened by golines
- Use `goimports` and `golines` for consistent formatting
- Run `make lint` before committing

### Naming Conventions

**Packages**
- Use lowercase, single-word names when possible
- Avoid underscores or mixed caps
- Examples: `domain`, `usecase`, `repository`, `http`

**Types**
- Use PascalCase for exported types
- Use camelCase for unexported types
- Suffix interfaces with meaningful names, not just "Interface"
  - Good: `UserRepository`, `TxManager`, `UseCase`
  - Bad: `UserRepositoryInterface`, `IUserRepository`

**Functions/Methods**
- Use PascalCase for exported functions
- Use camelCase for unexported functions
- Prefix boolean functions with `is`, `has`, `can`, or `should`
  - Examples: `isPostgreSQLUniqueViolation`, `hasUpperCase`

**Variables**
- Use camelCase for short-lived variables
- Use descriptive names for longer-lived variables
- Avoid single-letter names except for: `i`, `j`, `k` (loops), `r` (receiver), `w` (http.ResponseWriter), `ctx` (context)

**Constants**
- Use PascalCase for exported constants
- Use camelCase for unexported constants

### Types and Interfaces

**UUIDs**
- Use `uuid.UUID` type from `github.com/google/uuid`
- Generate IDs with `uuid.Must(uuid.NewV7())` (time-ordered UUIDs)
- PostgreSQL: Store as native `UUID` type
- MySQL: Store as `BINARY(16)` with marshal/unmarshal

**Domain Models**
- Keep domain models pure (no JSON tags)
- Use DTOs for API serialization
- Example:
  ```go
  // Domain model (no JSON tags)
  type User struct {
      ID        uuid.UUID
      Name      string
      Email     string
      Password  string
      CreatedAt time.Time
      UpdatedAt time.Time
  }
  ```

**DTOs**
- Add JSON tags for API contracts
- Implement `Validate()` method using `jellydator/validation`
- Create mapper functions to convert between DTOs and domain models

### Error Handling

**Standard Domain Errors** (from `internal/errors`)
- `ErrNotFound` - Resource doesn't exist (404)
- `ErrConflict` - Duplicate or conflicting data (409)
- `ErrInvalidInput` - Validation failures (422)
- `ErrUnauthorized` - Authentication required (401)
- `ErrForbidden` - Permission denied (403)

**Domain-Specific Errors**
Always wrap standard errors with domain context:
```go
var (
    ErrUserNotFound      = errors.Wrap(errors.ErrNotFound, "user not found")
    ErrUserAlreadyExists = errors.Wrap(errors.ErrConflict, "user already exists")
    ErrInvalidEmail      = errors.Wrap(errors.ErrInvalidInput, "invalid email format")
)
```

**Error Flow**
1. **Repository Layer**: Transform infrastructure errors to domain errors
   ```go
   if errors.Is(err, sql.ErrNoRows) {
       return nil, domain.ErrUserNotFound
   }
   ```

2. **Use Case Layer**: Return domain errors directly (don't wrap again)
   ```go
   user, err := uc.userRepo.GetByID(ctx, id)
   if err != nil {
       return nil, err  // Pass through domain error
   }
   ```

3. **HTTP Handler Layer**: Use `httputil.HandleError()` for automatic mapping
   ```go
   if err != nil {
       httputil.HandleError(w, err, h.logger)
       return
   }
   ```

**Error Checking**
- Use `errors.Is()` to check for specific errors
- Use `errors.As()` to extract error types
- Never compare errors with `==` (except for `sql.ErrNoRows` from stdlib)

### Validation

Use `github.com/jellydator/validation` for input validation:

```go
import (
    validation "github.com/jellydator/validation"
    appValidation "github.com/allisson/secrets/internal/validation"
)

func (r *RegisterUserRequest) Validate() error {
    err := validation.ValidateStruct(r,
        validation.Field(&r.Name,
            validation.Required.Error("name is required"),
            appValidation.NotBlank,
            validation.Length(1, 255),
        ),
        validation.Field(&r.Email,
            validation.Required.Error("email is required"),
            appValidation.Email,
        ),
        validation.Field(&r.Password,
            validation.Required.Error("password is required"),
            appValidation.PasswordStrength{
                MinLength:      8,
                RequireUpper:   true,
                RequireLower:   true,
                RequireNumber:  true,
                RequireSpecial: true,
            },
        ),
    )
    return appValidation.WrapValidationError(err)
}
```

**Custom Validation Rules** (in `internal/validation`):
- `Email` - Email format validation
- `NotBlank` - Not empty after trimming
- `NoWhitespace` - No leading/trailing whitespace
- `PasswordStrength` - Configurable password requirements

### Database Patterns

**Repository Pattern**
- Implement separate repositories for MySQL and PostgreSQL
- Use `database.GetTx(ctx, r.db)` to get querier (works with and without transactions)
- Use numbered placeholders for PostgreSQL (`$1, $2`) and `?` for MySQL

**Transaction Management**
```go
err := uc.txManager.WithTx(ctx, func(ctx context.Context) error {
    if err := uc.userRepo.Create(ctx, user); err != nil {
        return err
    }
    if err := uc.outboxRepo.Create(ctx, event); err != nil {
        return err
    }
    return nil
})
```

**Unique Constraint Violations**
Check for database-specific error patterns:
- PostgreSQL: `strings.Contains(err.Error(), "duplicate key")`
- MySQL: `strings.Contains(err.Error(), "duplicate entry")`

### Testing

**Integration Testing Philosophy**
- Use real databases (PostgreSQL port 5433, MySQL port 3307) instead of mocks
- Tests verify actual SQL queries and database behavior
- Start test databases with `make test-db-up` before running tests

**Test Structure**
```go
func TestPostgreSQLUserRepository_Create(t *testing.T) {
    db := testutil.SetupPostgresDB(t)           // Connect and run migrations
    defer testutil.TeardownDB(t, db)            // Clean up connection
    defer testutil.CleanupPostgresDB(t, db)     // Clean up test data
    
    repo := NewPostgreSQLUserRepository(db)
    ctx := context.Background()
    
    user := &domain.User{
        ID:       uuid.Must(uuid.NewV7()),
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "hashed_password",
    }
    
    err := repo.Create(ctx, user)
    assert.NoError(t, err)
}
```

**Test Naming**
- Format: `Test{Type}_{Method}` or `Test{Type}_{Method}_{Scenario}`
- Examples: `TestPostgreSQLUserRepository_Create`, `TestUserUseCase_RegisterUser_DuplicateEmail`

### Comments and Documentation

**Package Comments**
Every package should have a doc comment:
```go
// Package usecase implements the user business logic and orchestrates user domain operations.
package usecase
```

**Exported Types**
Document all exported types, functions, and constants:
```go
// User represents a user in the system
type User struct { ... }

// RegisterUser registers a new user and creates a user.created event
func (uc *UserUseCase) RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
```

**Implementation Comments**
Add comments for complex logic:
```go
// Hash the password using Argon2id
hashedPassword, err := uc.passwordHasher.Hash([]byte(input.Password))
```

## Adding New Domains

When adding a new domain (e.g., `product`):

1. **Create the domain structure**:
   ```
   internal/product/
   ├── domain/
   │   └── product.go              # Entity + domain errors
   ├── usecase/
   │   └── product_usecase.go      # UseCase interface + implementation
   ├── repository/
   │   ├── mysql_product_repository.go
   │   └── postgresql_product_repository.go
   └── http/
       ├── dto/
       │   ├── request.go           # With Validate() method
       │   ├── response.go          # With JSON tags
       │   └── mapper.go            # DTO ↔ domain conversions
       └── product_handler.go       # Uses httputil functions
   ```

2. **Define domain errors** in `domain/product.go`:
   ```go
   var (
       ErrProductNotFound = errors.Wrap(errors.ErrNotFound, "product not found")
       ErrInvalidPrice    = errors.Wrap(errors.ErrInvalidInput, "invalid price")
   )
   ```

3. **Register in DI container** (`internal/app/di.go`):
   - Add repository and use case fields
   - Add getter methods with `sync.Once` initialization
   - Select repository based on `c.config.DBDriver`

4. **Wire HTTP handlers** in `internal/http/server.go`

5. **Write migrations** for both PostgreSQL and MySQL

## Common Pitfalls to Avoid

❌ **Don't** add JSON tags to domain models  
✅ **Do** use DTOs for API serialization

❌ **Don't** wrap domain errors multiple times  
✅ **Do** return domain errors directly from use cases

❌ **Don't** use `sql.ErrNoRows` in use cases  
✅ **Do** transform to domain errors in repositories

❌ **Don't** use single repository for both databases  
✅ **Do** implement separate MySQL and PostgreSQL repositories

❌ **Don't** use auto-incrementing integer IDs  
✅ **Do** use UUIDv7 with `uuid.Must(uuid.NewV7())`

❌ **Don't** create mocks for repositories  
✅ **Do** use real test databases (ports 5433 and 3307)

❌ **Don't** depend on concrete use case implementations  
✅ **Do** depend on UseCase interfaces in handlers and DI container
