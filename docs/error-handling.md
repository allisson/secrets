# ⚠️ Error Handling

This guide explains the standardized error handling system used in this Go project.

## 🎯 Philosophy

The error handling system is designed to:

- 🎯 **Express business intent** rather than expose infrastructure details
- 🔒 **Prevent information leaks** (database errors never exposed to API clients)
- ✅ **Maintain consistency** (same error always maps to same HTTP status)
- 🔍 **Enable type-safe error checking** using `errors.Is()`
- 📊 **Provide structured responses** with consistent JSON format

## 📚 Standard Domain Errors

The project defines standard domain errors in `internal/errors/errors.go`:

```go
package errors

import "errors"

var (
    ErrNotFound      = errors.New("not found")          // 404 Not Found
    ErrConflict      = errors.New("conflict")           // 409 Conflict
    ErrInvalidInput  = errors.New("invalid input")      // 422 Unprocessable Entity
    ErrUnauthorized  = errors.New("unauthorized")       // 401 Unauthorized
    ErrForbidden     = errors.New("forbidden")          // 403 Forbidden
)

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
    return fmt.Errorf("%s: %w", message, err)
}
```

## 🏷️ Domain-Specific Errors

Each domain defines its own specific errors by **wrapping** the standard errors:

```go
// internal/user/domain/user.go
package domain

import (
    apperrors "github.com/allisson/go-project-template/internal/errors"
)

var (
    ErrUserNotFound      = apperrors.Wrap(apperrors.ErrNotFound, "user not found")
    ErrUserAlreadyExists = apperrors.Wrap(apperrors.ErrConflict, "user already exists")
    ErrInvalidEmail      = apperrors.Wrap(apperrors.ErrInvalidInput, "invalid email format")
    ErrWeakPassword      = apperrors.Wrap(apperrors.ErrInvalidInput, "password does not meet strength requirements")
)
```

**Key Points**:
- ✅ Wrap standard errors with domain-specific context
- ✅ Use descriptive error messages that explain the business problem
- ✅ Group related errors in the domain package

## 🔄 Error Flow Through Layers

Errors flow through the application layers, being transformed from infrastructure concerns to domain concepts:

```
Infrastructure Error (sql.ErrNoRows)
         ↓
    [Repository Layer]
         ↓
Domain Error (ErrUserNotFound)
         ↓
    [Use Case Layer]
         ↓
Domain Error (unchanged)
         ↓
    [HTTP Handler Layer]
         ↓
HTTP Response (404 Not Found)
```

### 1️⃣ Repository Layer

**Responsibility**: Transform infrastructure errors to domain errors

```go
// internal/user/repository/postgresql_user_repository.go
func (r *PostgreSQLUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    querier := database.GetTx(ctx, r.db)
    query := `SELECT id, name, email, password, created_at, updated_at FROM users WHERE id = $1`
    
    var user domain.User
    err := querier.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrUserNotFound  // ✅ Transform infrastructure → domain
        }
        return nil, apperrors.Wrap(err, "failed to get user by id")
    }
    return &user, nil
}
```

**Unique Constraint Violations**:

```go
func (r *PostgreSQLUserRepository) Create(ctx context.Context, user *domain.User) error {
    querier := database.GetTx(ctx, r.db)
    
    query := `INSERT INTO users (id, name, email, password, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, NOW(), NOW())`
    
    _, err := querier.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.Password)
    if err != nil {
        if isPostgreSQLUniqueViolation(err) {
            return domain.ErrUserAlreadyExists  // ✅ Transform unique violation → domain error
        }
        return apperrors.Wrap(err, "failed to create user")
    }
    return nil
}

func isPostgreSQLUniqueViolation(err error) bool {
    return strings.Contains(err.Error(), "duplicate key") ||
           strings.Contains(err.Error(), "unique constraint")
}
```

**MySQL Example**:

```go
func isMySQLUniqueViolation(err error) bool {
    return strings.Contains(err.Error(), "Error 1062") ||
           strings.Contains(err.Error(), "Duplicate entry")
}
```

### 2️⃣ Use Case Layer

**Responsibility**: Return domain errors directly (don't wrap again)

```go
// internal/user/usecase/user_usecase.go
func (uc *UserUseCase) RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
    // Validate input
    if err := input.Validate(); err != nil {
        return nil, err  // ✅ Return validation error directly
    }
    
    // Check if user exists
    existingUser, err := uc.userRepo.GetByEmail(ctx, input.Email)
    if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
        return nil, err  // ✅ Return unexpected errors directly
    }
    if existingUser != nil {
        return nil, domain.ErrUserAlreadyExists  // ✅ Return domain error directly
    }
    
    // Hash password
    hashedPassword, err := uc.passwordHasher.Hash([]byte(input.Password))
    if err != nil {
        return nil, apperrors.Wrap(err, "failed to hash password")
    }
    
    // Create user
    user := &domain.User{
        ID:       uuid.Must(uuid.NewV7()),
        Name:     input.Name,
        Email:    input.Email,
        Password: string(hashedPassword),
    }
    
    err = uc.txManager.WithTx(ctx, func(ctx context.Context) error {
        if err := uc.userRepo.Create(ctx, user); err != nil {
            return err  // ✅ Pass through domain errors
        }
        return uc.outboxRepo.Create(ctx, event)
    })
    
    return user, err  // ✅ Return error unchanged
}
```

**Key Points**:
- ✅ **Never wrap domain errors** in the use case layer
- ✅ **Return domain errors directly** to maintain error chain
- ✅ **Use `errors.Is()`** to check for specific error types

### 3️⃣ HTTP Handler Layer

**Responsibility**: Map domain errors to HTTP responses

```go
// internal/user/http/user_handler.go
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
    var req dto.RegisterUserRequest
    
    // Decode request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.HandleValidationError(w, err, h.logger)  // 400 Bad Request
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        httputil.HandleError(w, err, h.logger)  // 422 Unprocessable Entity
        return
    }
    
    // Execute use case
    input := dto.ToRegisterUserInput(req)
    user, err := h.userUseCase.RegisterUser(r.Context(), input)
    if err != nil {
        httputil.HandleError(w, err, h.logger)  // ✅ Auto-maps to HTTP status
        return
    }
    
    // Success response
    response := dto.ToUserResponse(user)
    httputil.MakeJSONResponse(w, http.StatusCreated, response)
}
```

## 🌐 HTTP Error Mapping

The `httputil.HandleError()` function automatically maps domain errors to HTTP status codes:

```go
// internal/httputil/response.go
func HandleError(w http.ResponseWriter, err error, logger *slog.Logger) {
    var statusCode int
    var errorCode string
    var message string

    switch {
    case errors.Is(err, apperrors.ErrNotFound):
        statusCode = http.StatusNotFound
        errorCode = "not_found"
        message = "The requested resource was not found"
    case errors.Is(err, apperrors.ErrConflict):
        statusCode = http.StatusConflict
        errorCode = "conflict"
        message = "A conflict occurred with existing data"
    case errors.Is(err, apperrors.ErrInvalidInput):
        statusCode = http.StatusUnprocessableEntity
        errorCode = "invalid_input"
        message = err.Error()  // Include detailed validation message
    case errors.Is(err, apperrors.ErrUnauthorized):
        statusCode = http.StatusUnauthorized
        errorCode = "unauthorized"
        message = "Authentication is required"
    case errors.Is(err, apperrors.ErrForbidden):
        statusCode = http.StatusForbidden
        errorCode = "forbidden"
        message = "You don't have permission to access this resource"
    default:
        statusCode = http.StatusInternalServerError
        errorCode = "internal_error"
        message = "An internal error occurred"
    }

    // Log error with context
    logger.Error("request error",
        "error", err.Error(),
        "status_code", statusCode,
        "error_code", errorCode,
    )

    // Send JSON response
    MakeJSONResponse(w, statusCode, map[string]string{
        "error":   errorCode,
        "message": message,
    })
}
```

### Error to HTTP Status Mapping Table

| Domain Error | HTTP Status | Error Code | Use Case |
|--------------|-------------|------------|----------|
| `ErrNotFound` | 404 | `not_found` | Resource doesn't exist |
| `ErrConflict` | 409 | `conflict` | Duplicate or conflicting data |
| `ErrInvalidInput` | 422 | `invalid_input` | Validation failures |
| `ErrUnauthorized` | 401 | `unauthorized` | Authentication required |
| `ErrForbidden` | 403 | `forbidden` | Permission denied |
| Unknown | 500 | `internal_error` | Unexpected errors |

## 📋 Error Response Format

All error responses follow a consistent JSON structure:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

### Examples

**404 Not Found**:
```json
{
  "error": "not_found",
  "message": "The requested resource was not found"
}
```

**409 Conflict**:
```json
{
  "error": "conflict",
  "message": "A conflict occurred with existing data"
}
```

**422 Validation Error**:
```json
{
  "error": "invalid_input",
  "message": "email: must be a valid email address; password: password must contain at least one uppercase letter."
}
```

**500 Internal Server Error**:
```json
{
  "error": "internal_error",
  "message": "An internal error occurred"
}
```

## ✅ Validation Errors

Validation errors are treated as `ErrInvalidInput` and automatically return 422 status.

### DTO Validation

```go
// internal/user/http/dto/request.go
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
    return appValidation.WrapValidationError(err)  // ✅ Wraps as ErrInvalidInput
}
```

### WrapValidationError

The `WrapValidationError` function converts validation errors to `ErrInvalidInput`:

```go
// internal/validation/rules.go
func WrapValidationError(err error) error {
    if err == nil {
        return nil
    }
    return apperrors.Wrap(apperrors.ErrInvalidInput, err.Error())
}
```

## 🔍 Checking Errors

Use `errors.Is()` to check for specific error types:

```go
user, err := uc.userRepo.GetByEmail(ctx, email)
if err != nil {
    if errors.Is(err, domain.ErrUserNotFound) {
        // Handle not found case
        return nil, domain.ErrInvalidCredentials
    }
    // Handle other errors
    return nil, err
}
```

**DO ✅**:
```go
if errors.Is(err, domain.ErrUserNotFound) {
    // Handle error
}
```

**DON'T ❌**:
```go
if err == domain.ErrUserNotFound {  // Won't work with wrapped errors
    // Handle error
}
```

## ➕ Adding Errors to New Domains

When creating a new domain (e.g., `product`), define domain-specific errors:

```go
// internal/product/domain/product.go
package domain

import (
    apperrors "github.com/allisson/go-project-template/internal/errors"
)

var (
    ErrProductNotFound    = apperrors.Wrap(apperrors.ErrNotFound, "product not found")
    ErrInsufficientStock  = apperrors.Wrap(apperrors.ErrConflict, "insufficient stock")
    ErrInvalidPrice       = apperrors.Wrap(apperrors.ErrInvalidInput, "invalid price")
    ErrInvalidQuantity    = apperrors.Wrap(apperrors.ErrInvalidInput, "quantity must be positive")
)
```

Then use `httputil.HandleError()` in your HTTP handlers for automatic mapping - no additional code needed!

## 🎯 Best Practices

### DO ✅

- ✅ **Define domain-specific errors** by wrapping standard errors
- ✅ **Transform infrastructure errors** in repository layer
- ✅ **Return domain errors directly** from use case layer
- ✅ **Use `errors.Is()`** for type-safe error checking
- ✅ **Use `httputil.HandleError()`** for consistent HTTP error responses
- ✅ **Log errors with context** before responding to client
- ✅ **Include descriptive messages** in domain errors
- ✅ **Keep error messages user-friendly** (no stack traces or internal details)

### DON'T ❌

- ❌ **Don't expose infrastructure errors** (like `sql.ErrNoRows`) to API clients
- ❌ **Don't wrap domain errors** multiple times in use case layer
- ❌ **Don't compare errors with `==`** (use `errors.Is()` instead)
- ❌ **Don't return different HTTP status codes** for the same domain error
- ❌ **Don't include sensitive information** in error messages
- ❌ **Don't return stack traces** to API clients
- ❌ **Don't create new error types** when standard errors suffice

## 🔒 Security Considerations

### Information Disclosure

**DO ✅**:
```go
// Return generic error message
return domain.ErrUserNotFound
```

**DON'T ❌**:
```go
// Reveals internal database structure
return fmt.Errorf("no row found in users table for id=%s", id)
```

### Error Logging

Log detailed errors on the server, but return generic messages to clients:

```go
func HandleError(w http.ResponseWriter, err error, logger *slog.Logger) {
    // Log detailed error server-side
    logger.Error("request error",
        "error", err.Error(),
        "stack", getStackTrace(),
    )
    
    // Return generic message to client
    MakeJSONResponse(w, statusCode, map[string]string{
        "error":   errorCode,
        "message": genericMessage,  // No sensitive details
    })
}
```

## 📊 Benefits Summary

1. 🎯 **No Infrastructure Leaks** - Database errors never exposed to API clients
2. 💼 **Business Intent** - Errors express domain concepts (e.g., `ErrUserNotFound` vs `sql.ErrNoRows`)
3. ✅ **Consistent HTTP Mapping** - Same domain error always maps to same HTTP status
4. 🔒 **Type-Safe** - Use `errors.Is()` to check for specific error types
5. 📋 **Structured Responses** - All errors return consistent JSON format
6. 📊 **Centralized Logging** - All errors logged with full context before responding
7. 🛡️ **Security** - Sensitive information never leaked in error messages
