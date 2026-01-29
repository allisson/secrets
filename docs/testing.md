# 🧪 Testing Guide

This guide covers testing strategies, best practices, and how to write effective tests for this Go project.

## 🎯 Testing Philosophy

The project uses **integration testing with real databases** instead of mocks for repository layer tests.

**Why Real Databases?**
- ✅ **Accuracy** - Tests verify actual SQL queries and database behavior
- ✅ **Real Integration** - Catches database-specific issues (constraints, types, unique violations)
- ✅ **Production Parity** - Tests reflect real production scenarios
- ✅ **Less Maintenance** - No mock expectations to maintain or update
- ✅ **Confidence** - Full database integration coverage

## 🏗️ Test Infrastructure

### Test Databases

Tests use Docker Compose to spin up isolated test databases:

- **PostgreSQL**: `localhost:5433` (testuser/testpassword/testdb)
- **MySQL**: `localhost:3307` (testuser/testpassword/testdb)

**Note**: Different ports from development (5432/3306) to avoid conflicts.

### Test Utilities

The `testutil` package (`internal/testutil/database.go`) provides helper functions:

1. **`SetupPostgresDB(t)`** - Connect to PostgreSQL and run migrations
2. **`SetupMySQLDB(t)`** - Connect to MySQL and run migrations
3. **`CleanupPostgresDB(t, db)`** - Clean up PostgreSQL test data
4. **`CleanupMySQLDB(t, db)`** - Clean up MySQL test data
5. **`TeardownDB(t, db)`** - Close database connection

## 🚀 Running Tests

### Start Test Databases

Before running tests, start the test databases:

```bash
make test-db-up
```

This command:
- Starts PostgreSQL on port 5433
- Starts MySQL on port 3307
- Waits for databases to be healthy

### Run All Tests

```bash
make test
```

This runs all tests with coverage reporting.

### Run Tests with Automatic Database Management

```bash
make test-with-db
```

This command:
1. Starts test databases
2. Runs all tests
3. Stops test databases

Perfect for CI/CD environments!

### Run Tests with Coverage

```bash
make test-coverage
```

This generates an HTML coverage report and opens it in your browser.

### Stop Test Databases

```bash
make test-db-down
```

### Run Tests for Specific Package

```bash
# Run all tests in a package
go test -v ./internal/user/repository

# Run a specific test
go test -v ./internal/user/repository -run TestPostgreSQLUserRepository_Create

# Run with race detection
go test -v -race ./internal/user/repository
```

## 📝 Writing Tests

### Repository Tests

Repository tests use real databases to verify SQL queries and database interactions.

**Test Structure**:

```go
package repository

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"

    "github.com/allisson/go-project-template/internal/testutil"
    "github.com/allisson/go-project-template/internal/user/domain"
)

func TestPostgreSQLUserRepository_Create(t *testing.T) {
    // Setup: Connect to database and run migrations
    db := testutil.SetupPostgresDB(t)
    defer testutil.TeardownDB(t, db)            // Close connection
    defer testutil.CleanupPostgresDB(t, db)     // Clean up test data
    
    // Create repository
    repo := NewPostgreSQLUserRepository(db)
    ctx := context.Background()
    
    // Prepare test data
    user := &domain.User{
        ID:       uuid.Must(uuid.NewV7()),
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "hashed_password",
    }
    
    // Execute test
    err := repo.Create(ctx, user)
    
    // Assert results
    assert.NoError(t, err)
    
    // Verify by querying the real database
    createdUser, err := repo.GetByID(ctx, user.ID)
    assert.NoError(t, err)
    assert.Equal(t, user.Name, createdUser.Name)
    assert.Equal(t, user.Email, createdUser.Email)
}
```

**Key Points**:
- 🔄 Use `defer` for cleanup (connection and data)
- 🧹 Clean up test data to prevent test pollution
- ✅ Verify operations by querying the database
- 🎯 Test one thing per test function

### Testing Error Cases

```go
func TestPostgreSQLUserRepository_GetByID_NotFound(t *testing.T) {
    db := testutil.SetupPostgresDB(t)
    defer testutil.TeardownDB(t, db)
    defer testutil.CleanupPostgresDB(t, db)
    
    repo := NewPostgreSQLUserRepository(db)
    ctx := context.Background()
    
    // Try to get non-existent user
    nonExistentID := uuid.Must(uuid.NewV7())
    user, err := repo.GetByID(ctx, nonExistentID)
    
    // Verify error handling
    assert.Error(t, err)
    assert.Nil(t, user)
    assert.ErrorIs(t, err, domain.ErrUserNotFound)
}
```

### Testing Unique Constraints

```go
func TestPostgreSQLUserRepository_Create_DuplicateEmail(t *testing.T) {
    db := testutil.SetupPostgresDB(t)
    defer testutil.TeardownDB(t, db)
    defer testutil.CleanupPostgresDB(t, db)
    
    repo := NewPostgreSQLUserRepository(db)
    ctx := context.Background()
    
    // Create first user
    user1 := &domain.User{
        ID:       uuid.Must(uuid.NewV7()),
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "password1",
    }
    err := repo.Create(ctx, user1)
    assert.NoError(t, err)
    
    // Try to create second user with same email
    user2 := &domain.User{
        ID:       uuid.Must(uuid.NewV7()),
        Name:     "Jane Doe",
        Email:    "john@example.com",  // Duplicate email
        Password: "password2",
    }
    err = repo.Create(ctx, user2)
    
    // Verify unique constraint error
    assert.Error(t, err)
    assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}
```

### Use Case Tests

Use case tests can use real repositories or mocks depending on the scenario.

**Testing with Real Repository**:

```go
func TestUserUseCase_RegisterUser(t *testing.T) {
    db := testutil.SetupPostgresDB(t)
    defer testutil.TeardownDB(t, db)
    defer testutil.CleanupPostgresDB(t, db)
    
    // Setup dependencies
    userRepo := repository.NewPostgreSQLUserRepository(db)
    outboxRepo := outboxRepository.NewPostgreSQLOutboxRepository(db)
    txManager := database.NewTxManager(db)
    passwordHasher := pwdhash.NewArgon2Hasher(pwdhash.Argon2Config{})
    
    // Create use case
    uc := usecase.NewUserUseCase(txManager, userRepo, outboxRepo, passwordHasher)
    ctx := context.Background()
    
    // Prepare input
    input := usecase.RegisterUserInput{
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "SecurePass123!",
    }
    
    // Execute
    user, err := uc.RegisterUser(ctx, input)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, input.Name, user.Name)
    assert.Equal(t, input.Email, user.Email)
    
    // Verify user was created in database
    createdUser, err := userRepo.GetByEmail(ctx, input.Email)
    assert.NoError(t, err)
    assert.Equal(t, user.ID, createdUser.ID)
    
    // Verify outbox event was created
    events, err := outboxRepo.GetPending(ctx, 10)
    assert.NoError(t, err)
    assert.Len(t, events, 1)
    assert.Equal(t, "user.created", events[0].EventType)
}
```

### HTTP Handler Tests

HTTP handler tests verify the presentation layer.

```go
func TestUserHandler_RegisterUser(t *testing.T) {
    db := testutil.SetupPostgresDB(t)
    defer testutil.TeardownDB(t, db)
    defer testutil.CleanupPostgresDB(t, db)
    
    // Setup dependencies
    userRepo := repository.NewPostgreSQLUserRepository(db)
    outboxRepo := outboxRepository.NewPostgreSQLOutboxRepository(db)
    txManager := database.NewTxManager(db)
    passwordHasher := pwdhash.NewArgon2Hasher(pwdhash.Argon2Config{})
    uc := usecase.NewUserUseCase(txManager, userRepo, outboxRepo, passwordHasher)
    logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
    handler := http.NewUserHandler(uc, logger)
    
    // Prepare request
    reqBody := `{
        "name": "John Doe",
        "email": "john@example.com",
        "password": "SecurePass123!"
    }`
    req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    // Execute
    handler.RegisterUser(w, req)
    
    // Assert response
    assert.Equal(t, http.StatusCreated, w.Code)
    
    var response dto.UserResponse
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, "John Doe", response.Name)
    assert.Equal(t, "john@example.com", response.Email)
}
```

### Testing Validation Errors

```go
func TestUserHandler_RegisterUser_ValidationError(t *testing.T) {
    // Setup (minimal dependencies for validation test)
    logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
    handler := http.NewUserHandler(nil, logger)
    
    // Invalid request (missing required fields)
    reqBody := `{
        "name": "",
        "email": "invalid-email",
        "password": "weak"
    }`
    req := httptest.NewRequest("POST", "/api/users", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    // Execute
    handler.RegisterUser(w, req)
    
    // Assert validation error response
    assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
    
    var errorResponse map[string]string
    err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
    assert.NoError(t, err)
    assert.Equal(t, "invalid_input", errorResponse["error"])
    assert.Contains(t, errorResponse["message"], "name")
    assert.Contains(t, errorResponse["message"], "email")
    assert.Contains(t, errorResponse["message"], "password")
}
```

## 📊 Test Coverage

### Viewing Coverage Reports

```bash
# Generate and view HTML coverage report
make test-coverage
```

This opens an HTML report in your browser showing:
- 📈 Overall coverage percentage
- 📁 Coverage by package
- 📄 Line-by-line coverage highlighting

### Coverage Goals

Aim for these coverage targets:
- **Domain Layer**: 90%+ (core business logic)
- **Use Case Layer**: 85%+ (business orchestration)
- **Repository Layer**: 90%+ (data access)
- **HTTP Layer**: 80%+ (handlers and DTOs)

### Checking Coverage from Command Line

```bash
# Run tests with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

## 🔍 Test Naming Conventions

Use descriptive test names that follow this pattern:

```
Test{Type}_{Method}_{Scenario}
```

**Examples**:
- `TestPostgreSQLUserRepository_Create`
- `TestPostgreSQLUserRepository_GetByID_NotFound`
- `TestUserUseCase_RegisterUser_DuplicateEmail`
- `TestUserHandler_RegisterUser_ValidationError`

**Benefits**:
- ✅ Easy to identify what's being tested
- ✅ Clear understanding of test scenarios
- ✅ Better test failure messages

## 🏃 CI/CD Testing

### GitHub Actions

The project includes a GitHub Actions workflow (`.github/workflows/ci.yml`) that:

1. ✅ Starts PostgreSQL (port 5433) and MySQL (port 3307) containers
2. ✅ Waits for both databases to be healthy
3. ✅ Runs all tests with race detection
4. ✅ Generates coverage reports
5. ✅ Uploads coverage to Codecov

**CI Configuration**:
- Same database credentials as local tests (testuser/testpassword/testdb)
- Same port mappings as Docker Compose (5433 for Postgres, 3307 for MySQL)
- Runs on every push to `main` and all pull requests
- All tests must pass before merging

### Running Tests Like CI Locally

```bash
# Exact same command as CI
make test-with-db
```

This ensures consistency between local development and CI environments.

## 🛠️ Debugging Tests

### Run Tests with Verbose Output

```bash
go test -v ./internal/user/repository
```

### Run Single Test

```bash
go test -v ./internal/user/repository -run TestPostgreSQLUserRepository_Create
```

### Enable Race Detection

```bash
go test -race ./...
```

### Print Test Output

```go
func TestSomething(t *testing.T) {
    t.Logf("Debug info: %v", someValue)
    
    // Or use fmt for immediate output
    fmt.Printf("Debug info: %v\n", someValue)
}
```

### Check Database State During Tests

```go
func TestUserRepository_Create(t *testing.T) {
    db := testutil.SetupPostgresDB(t)
    defer testutil.TeardownDB(t, db)
    defer testutil.CleanupPostgresDB(t, db)
    
    repo := NewPostgreSQLUserRepository(db)
    
    // Create user
    err := repo.Create(ctx, user)
    assert.NoError(t, err)
    
    // Manually query database to verify
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", user.Email).Scan(&count)
    assert.NoError(t, err)
    assert.Equal(t, 1, count)
}
```

## 🎯 Best Practices

### DO ✅

- ✅ Use real databases for repository tests
- ✅ Clean up test data after each test
- ✅ Test both success and error cases
- ✅ Use descriptive test names
- ✅ Test one thing per test function
- ✅ Use table-driven tests for multiple scenarios
- ✅ Run tests before committing
- ✅ Maintain high test coverage
- ✅ Test edge cases and boundary conditions

### DON'T ❌

- ❌ Share test data between tests
- ❌ Rely on test execution order
- ❌ Skip cleanup in defer statements
- ❌ Use production databases for testing
- ❌ Hardcode database credentials
- ❌ Leave test databases running
- ❌ Commit commented-out tests
- ❌ Test implementation details instead of behavior

## 📚 Testing Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [testify/assert](https://github.com/stretchr/testify) - Assertion library
- [Testing Best Practices](https://golang.org/doc/effective_go#testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
