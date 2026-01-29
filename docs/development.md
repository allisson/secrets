# 🛠️ Development Guide

This guide covers the development workflow, coding standards, and best practices for this Go project.

## 🔨 Build Commands

### Building the Application

```bash
make build                    # Build the application binary
```

The binary will be created at `./bin/app`.

### Running Commands

```bash
make run-server              # Build and run HTTP server
make run-worker              # Build and run outbox event processor
make run-migrate             # Build and run database migrations
```

### Other Commands

```bash
make clean                   # Remove build artifacts and coverage files
make deps                    # Download and tidy dependencies
make help                    # Show all available make targets
```

## 🎨 Code Style Guidelines

### Import Organization

Follow this import grouping order (enforced by `goimports`):

1. **Standard library packages**
2. **External packages** (third-party)
3. **Local project packages** (prefixed with your module path)

**Example**:
```go
import (
    "context"
    "database/sql"
    "strings"

    "github.com/google/uuid"
    validation "github.com/jellydator/validation"

    "github.com/allisson/go-project-template/internal/database"
    "github.com/allisson/go-project-template/internal/errors"
    "github.com/allisson/go-project-template/internal/user/domain"
)
```

**Import Aliases**:
When renaming imports, use descriptive aliases:
- `apperrors` for `internal/errors`
- `appValidation` for `internal/validation`
- `outboxDomain` for `internal/outbox/domain`

### Formatting

- **Line length**: Max 110 characters (enforced by `golines`)
- **Indentation**: Use tabs (tab-len: 4)
- **Comments**: Not shortened by golines
- Run `make lint` before committing to auto-fix formatting issues

### Naming Conventions

#### Packages
- Use lowercase, single-word names when possible
- Avoid underscores or mixed caps
- ✅ Good: `domain`, `usecase`, `repository`, `http`
- ❌ Bad: `user_domain`, `userDomain`, `UserDomain`

#### Types
- Use PascalCase for exported types
- Use camelCase for unexported types
- Suffix interfaces with meaningful names
- ✅ Good: `UserRepository`, `TxManager`, `UseCase`
- ❌ Bad: `UserRepositoryInterface`, `IUserRepository`

#### Functions/Methods
- Use PascalCase for exported functions
- Use camelCase for unexported functions
- Prefix boolean functions with `is`, `has`, `can`, or `should`
- ✅ Good: `isPostgreSQLUniqueViolation`, `hasUpperCase`

#### Variables
- Use camelCase for short-lived variables
- Use descriptive names for longer-lived variables
- Avoid single-letter names except for:
  - `i`, `j`, `k` (loops)
  - `r` (receiver)
  - `w` (http.ResponseWriter)
  - `ctx` (context)

#### Constants
- Use PascalCase for exported constants
- Use camelCase for unexported constants

## 🔍 Linting

### Running the Linter

```bash
make lint                     # Run golangci-lint with auto-fix
```

Or directly:
```bash
golangci-lint run -v --fix   # Direct linter command
```

The linter will:
- ✅ Check code formatting (goimports, golines)
- ✅ Detect potential bugs
- ✅ Enforce code style guidelines
- ✅ Check for common mistakes
- ✅ Automatically fix issues when possible

### Linter Configuration

The project uses `.golangci.yml` for linter configuration. Key settings:

```yaml
linters:
  enable:
    - goimports      # Auto-organize imports
    - golines        # Enforce line length
    - errcheck       # Check error handling
    - gosimple       # Simplify code
    - govet          # Report suspicious constructs
    - staticcheck    # Static analysis
```

## 📝 Comments and Documentation

### Package Comments

Every package should have a doc comment:

```go
// Package usecase implements the user business logic and orchestrates user domain operations.
package usecase
```

### Exported Types

Document all exported types, functions, and constants:

```go
// User represents a user in the system
type User struct { ... }

// RegisterUser registers a new user and creates a user.created event
func (uc *UserUseCase) RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
    // Implementation
}
```

### Implementation Comments

Add comments for complex logic:

```go
// Hash the password using Argon2id
hashedPassword, err := uc.passwordHasher.Hash([]byte(input.Password))
```

## 🗄️ Database Patterns

### Repository Pattern

**Key Principles**:
- ✅ Implement separate repositories for MySQL and PostgreSQL
- ✅ Use `database.GetTx(ctx, r.db)` to get querier (supports transactions)
- ✅ Use numbered placeholders for PostgreSQL (`$1, $2`) and `?` for MySQL
- ✅ Transform infrastructure errors to domain errors

**Example**:
```go
func (r *PostgreSQLUserRepository) Create(ctx context.Context, user *domain.User) error {
    querier := database.GetTx(ctx, r.db)
    
    query := `INSERT INTO users (id, name, email, password, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, NOW(), NOW())`
    
    _, err := querier.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.Password)
    if err != nil {
        if isPostgreSQLUniqueViolation(err) {
            return domain.ErrUserAlreadyExists
        }
        return apperrors.Wrap(err, "failed to create user")
    }
    return nil
}
```

### Unique Constraint Violations

Check for database-specific error patterns:

**PostgreSQL**:
```go
func isPostgreSQLUniqueViolation(err error) bool {
    return strings.Contains(err.Error(), "duplicate key") ||
           strings.Contains(err.Error(), "unique constraint")
}
```

**MySQL**:
```go
func isMySQLUniqueViolation(err error) bool {
    return strings.Contains(err.Error(), "Error 1062") ||
           strings.Contains(err.Error(), "Duplicate entry")
}
```

## ✅ Validation Patterns

### Using jellydator/validation

```go
import (
    validation "github.com/jellydator/validation"
    appValidation "github.com/allisson/go-project-template/internal/validation"
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

### Custom Validation Rules

The project provides reusable validation rules in `internal/validation`:

- `Email` - Email format validation
- `NotBlank` - Not empty after trimming
- `NoWhitespace` - No leading/trailing whitespace
- `PasswordStrength` - Configurable password requirements

## 🆔 Working with UUIDs

### Generating UUIDs

Always use UUIDv7 for primary keys:

```go
import "github.com/google/uuid"

user := &domain.User{
    ID:       uuid.Must(uuid.NewV7()),
    Name:     input.Name,
    Email:    input.Email,
    Password: hashedPassword,
}
```

### Database Storage

**PostgreSQL** - Native UUID type:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL
);
```

**MySQL** - BINARY(16) with marshal/unmarshal:
```sql
CREATE TABLE users (
    id BINARY(16) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```go
// MySQL requires UUID marshaling
uuidBytes, err := user.ID.MarshalBinary()
if err != nil {
    return apperrors.Wrap(err, "failed to marshal UUID")
}
_, err = querier.ExecContext(ctx, query, uuidBytes, user.Name, user.Email)
```

## 🔄 Transaction Management

### Using TxManager

```go
err := uc.txManager.WithTx(ctx, func(ctx context.Context) error {
    // All operations in this function share the same transaction
    
    if err := uc.userRepo.Create(ctx, user); err != nil {
        return err  // Transaction will rollback
    }
    
    if err := uc.outboxRepo.Create(ctx, event); err != nil {
        return err  // Transaction will rollback
    }
    
    return nil  // Transaction will commit
})
```

### Repository Support for Transactions

Repositories automatically use transactions when available:

```go
func (r *PostgreSQLUserRepository) Create(ctx context.Context, user *domain.User) error {
    // GetTx returns transaction if present in context, otherwise returns DB
    querier := database.GetTx(ctx, r.db)
    
    _, err := querier.ExecContext(ctx, query, args...)
    return err
}
```

## 📦 Dependency Injection

### Adding Components to DI Container

When adding new components to `internal/app/di.go`:

1. **Add fields** to Container struct:
```go
type Container struct {
    // ... existing fields
    productRepo        productUsecase.ProductRepository
    productUseCase     productUsecase.UseCase  // Interface!
    productRepoInit    sync.Once
    productUseCaseInit sync.Once
}
```

2. **Add getter methods** with lazy initialization:
```go
func (c *Container) ProductRepository() (productUsecase.ProductRepository, error) {
    var err error
    c.productRepoInit.Do(func() {
        c.productRepo, err = c.initProductRepository()
        if err != nil {
            c.initErrors["productRepo"] = err
        }
    })
    if err != nil {
        return nil, err
    }
    if existingErr, ok := c.initErrors["productRepo"]; ok {
        return nil, existingErr
    }
    return c.productRepo, nil
}
```

3. **Add initialization methods**:
```go
func (c *Container) initProductRepository() (productUsecase.ProductRepository, error) {
    db, err := c.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get database: %w", err)
    }
    
    switch c.config.DBDriver {
    case "mysql":
        return productRepository.NewMySQLProductRepository(db), nil
    case "postgres":
        return productRepository.NewPostgreSQLProductRepository(db), nil
    default:
        return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
    }
}
```

## 🚀 Development Workflow

### 1. Create a New Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

- Write code following the style guidelines
- Add tests for new functionality
- Update documentation as needed

### 3. Run Tests

```bash
make test-db-up    # Start test databases
make test          # Run tests
make test-db-down  # Stop test databases
```

Or use the combined command:
```bash
make test-with-db  # Start databases, run tests, stop databases
```

### 4. Run Linter

```bash
make lint
```

### 5. Build the Application

```bash
make build
```

### 6. Commit Your Changes

```bash
git add .
git commit -m "feat: add new feature"
```

**Commit Message Format**:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Test updates
- `chore:` - Maintenance tasks

### 7. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## 🔧 Common Development Tasks

### Add a New Endpoint

1. Define request/response DTOs in `internal/{domain}/http/dto/`
2. Add validation to request DTO
3. Create handler in `internal/{domain}/http/{domain}_handler.go`
4. Register route in `internal/http/server.go`
5. Add tests

### Add a New Use Case

1. Define use case method in interface (`internal/{domain}/usecase/`)
2. Implement business logic
3. Add validation
4. Handle transactions if needed
5. Add tests

### Create a Database Migration

1. Create `000xxx_description.up.sql` in `migrations/postgresql/` or `migrations/mysql/`
2. Create corresponding `000xxx_description.down.sql`
3. Run migrations: `make run-migrate`

### Add a Custom Validation Rule

1. Add rule to `internal/validation/rules.go`
2. Implement `Validate(value interface{}) error` method
3. Add tests in `internal/validation/rules_test.go`
4. Use in DTOs

## 📚 Useful Resources

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)

## 🐛 Debugging Tips

### Enable Debug Logging

Set `LOG_LEVEL=debug` in your `.env` file:

```bash
LOG_LEVEL=debug
```

### Database Query Debugging

Add logging in repositories to see SQL queries:

```go
log.Printf("Executing query: %s with args: %v", query, args)
```

### Use Delve Debugger

Install Delve:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Run with debugger:
```bash
dlv debug ./cmd/app -- server
```

### Check Health Endpoints

```bash
curl http://localhost:8080/health    # Application health
curl http://localhost:8080/ready     # Database connectivity
```
