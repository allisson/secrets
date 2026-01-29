# 🏗️ Architecture

This document describes the architectural patterns and design principles used in this Go project template.

## 📐 Clean Architecture Layers

The project follows Clean Architecture principles with clear separation of concerns across five layers:

### 1. Domain Layer 🎯
**Location**: `internal/{domain}/domain/`

Contains business entities, domain errors, and business rules.

**Responsibilities**:
- Define entities with pure business logic (no JSON tags)
- Define domain-specific errors by wrapping standard errors
- Implement domain validation rules

**Example**:
```go
// internal/user/domain/user.go
package domain

import (
    "time"
    "github.com/google/uuid"
    "github.com/allisson/go-project-template/internal/errors"
)

type User struct {
    ID        uuid.UUID
    Name      string
    Email     string
    Password  string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Domain-specific errors
var (
    ErrUserNotFound      = errors.Wrap(errors.ErrNotFound, "user not found")
    ErrUserAlreadyExists = errors.Wrap(errors.ErrConflict, "user already exists")
    ErrInvalidEmail      = errors.Wrap(errors.ErrInvalidInput, "invalid email format")
)
```

### 2. Repository Layer 💾
**Location**: `internal/{domain}/repository/`

Handles data persistence and retrieval. Implements separate repositories for MySQL and PostgreSQL.

**Responsibilities**:
- Implement data access for each database type
- Transform infrastructure errors to domain errors (e.g., `sql.ErrNoRows` → `domain.ErrUserNotFound`)
- Use `database.GetTx(ctx, r.db)` to support transactions
- Handle database-specific concerns (UUID marshaling, placeholder syntax, etc.)

**Key Differences**:
| Feature | MySQL | PostgreSQL |
|---------|-------|------------|
| UUID Storage | `BINARY(16)` - requires marshaling/unmarshaling | Native `UUID` type |
| Placeholders | `?` for all parameters | `$1, $2, $3...` numbered parameters |
| Unique Errors | Check for "1062" or "duplicate entry" | Check for "duplicate key" or "unique constraint" |

**Example**:
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
            return nil, domain.ErrUserNotFound  // Transform to domain error
        }
        return nil, apperrors.Wrap(err, "failed to get user by id")
    }
    return &user, nil
}
```

### 3. Use Case Layer 💼
**Location**: `internal/{domain}/usecase/`

Implements business logic and orchestrates domain operations.

**Responsibilities**:
- Define UseCase interfaces for dependency inversion
- Implement business logic and orchestration
- Validate input using `github.com/jellydator/validation`
- Return domain errors directly without additional wrapping
- Manage transactions using `TxManager.WithTx()`

**Example**:
```go
// internal/user/usecase/user_usecase.go
type UseCase interface {
    RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error)
}

func (uc *UserUseCase) RegisterUser(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
    // Validate input
    if err := input.Validate(); err != nil {
        return nil, err
    }
    
    // Business logic
    hashedPassword, err := uc.passwordHasher.Hash([]byte(input.Password))
    if err != nil {
        return nil, apperrors.Wrap(err, "failed to hash password")
    }
    
    user := &domain.User{
        ID:       uuid.Must(uuid.NewV7()),
        Name:     input.Name,
        Email:    input.Email,
        Password: string(hashedPassword),
    }
    
    // Transaction management
    err = uc.txManager.WithTx(ctx, func(ctx context.Context) error {
        if err := uc.userRepo.Create(ctx, user); err != nil {
            return err  // Pass through domain errors
        }
        // Create outbox event in same transaction
        event := &outboxDomain.OutboxEvent{
            ID:          uuid.Must(uuid.NewV7()),
            EventType:   "user.created",
            Payload:     string(payload),
            Status:      outboxDomain.StatusPending,
        }
        return uc.outboxRepo.Create(ctx, event)
    })
    
    return user, err
}
```

### 4. Presentation Layer 🌐
**Location**: `internal/{domain}/http/`

Contains HTTP handlers and DTOs (Data Transfer Objects).

**Responsibilities**:
- Define request/response DTOs with JSON tags
- Validate DTOs using `jellydator/validation`
- Use `httputil.HandleError()` for automatic error-to-HTTP status mapping
- Use `httputil.MakeJSONResponse()` for consistent JSON responses
- Depend on UseCase interfaces, not concrete implementations

**DTO Structure**:
- `dto/request.go` - Request DTOs with validation
- `dto/response.go` - Response DTOs with JSON tags
- `dto/mapper.go` - Conversion functions between DTOs and domain models

**Example**:
```go
// internal/user/http/user_handler.go
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
    var req dto.RegisterUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.HandleValidationError(w, err, h.logger)
        return
    }
    
    if err := req.Validate(); err != nil {
        httputil.HandleError(w, err, h.logger)
        return
    }
    
    input := dto.ToRegisterUserInput(req)
    user, err := h.userUseCase.RegisterUser(r.Context(), input)
    if err != nil {
        httputil.HandleError(w, err, h.logger)  // Auto-maps domain errors to HTTP status
        return
    }
    
    response := dto.ToUserResponse(user)
    httputil.MakeJSONResponse(w, http.StatusCreated, response)
}
```

### 5. Utility Layer 🛠️
**Location**: `internal/{httputil,errors,validation}/`

Provides shared utilities for error handling, HTTP responses, and validation.

**Components**:
- **`internal/errors/`** - Standardized domain errors (ErrNotFound, ErrConflict, etc.)
- **`internal/httputil/`** - HTTP utilities (JSON responses, error mapping)
- **`internal/validation/`** - Custom validation rules (email, password strength, etc.)

## 🔄 Dependency Inversion Principle

The project follows the Dependency Inversion Principle where:
- **High-level modules** (use cases) define interfaces
- **Low-level modules** (repositories, handlers) implement those interfaces
- **Dependencies point inward** towards the domain

```
┌─────────────────────────────────────────┐
│      Presentation Layer (HTTP)          │
│  - Depends on UseCase interfaces        │
└──────────────────┬──────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────┐
│      Use Case Layer                      │
│  - Defines interfaces                    │
│  - Implements business logic             │
└──────────────────┬──────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────┐
│      Domain Layer                        │
│  - Pure business entities                │
│  - No external dependencies              │
└──────────────────┬──────────────────────┘
                   │
                   ▲
┌──────────────────┴──────────────────────┐
│      Repository Layer                    │
│  - Implements repository interfaces      │
│  - Depends on domain entities            │
└─────────────────────────────────────────┘
```

## 📦 Modular Domain Architecture

Each business domain is organized in its own directory with clear separation of concerns:

```
internal/
├── user/                       # User domain module
│   ├── domain/                 # User entities and domain errors
│   │   └── user.go
│   ├── usecase/                # User business logic
│   │   └── user_usecase.go
│   ├── repository/             # User data access
│   │   ├── mysql_user_repository.go
│   │   └── postgresql_user_repository.go
│   └── http/                   # User HTTP handlers
│       ├── dto/                # Request/response DTOs
│       │   ├── request.go
│       │   ├── response.go
│       │   └── mapper.go
│       └── user_handler.go
├── outbox/                     # Outbox domain module
│   ├── domain/                 # Outbox entities and domain errors
│   ├── usecase/                # Outbox event processing logic
│   └── repository/             # Outbox data access
└── {new-domain}/               # Easy to add new domains
```

**Benefits**:
1. 🎯 **Scalability** - Easy to add new domains without affecting existing code
2. 🔒 **Encapsulation** - Each domain is self-contained with clear boundaries
3. 👥 **Team Collaboration** - Teams can work on different domains independently
4. 🔧 **Maintainability** - Related code is co-located

## 🔌 Dependency Injection Container

The DI container (`internal/app/`) manages all application components with:

- **Centralized component wiring** - All dependencies assembled in one place
- **Lazy initialization** - Components created only when first accessed
- **Singleton pattern** - Each component initialized once and reused
- **Clean resource management** - Unified shutdown for all resources
- **Thread-safe** - Safe for concurrent access across goroutines

**Dependency Graph**:
```
Container
├── Infrastructure (Database, Logger)
├── Repositories (User, Outbox)
├── Use Cases (User, Outbox)
└── Presentation (HTTP Server)
```

**Example**:
```go
// Create container with configuration
container := app.NewContainer(cfg)

// Get HTTP server (automatically initializes all dependencies)
server, err := container.HTTPServer()
if err != nil {
    return fmt.Errorf("failed to initialize HTTP server: %w", err)
}

// Clean shutdown
defer container.Shutdown(ctx)
```

For more details, see [`internal/app/README.md`](../internal/app/README.md).

## 🆔 UUIDv7 Primary Keys

The project uses **UUIDv7** for all primary keys instead of auto-incrementing integers.

**Benefits**:
- ⏱️ **Time-ordered**: UUIDs include timestamp information
- 🌍 **Globally unique**: No collision risk across distributed systems
- 📊 **Database friendly**: Better index performance than random UUIDs (v4)
- 📈 **Scalability**: No need for centralized ID generation
- 🔀 **Merge-friendly**: Databases can be merged without ID conflicts

**Implementation**:
```go
import "github.com/google/uuid"

user := &domain.User{
    ID:       uuid.Must(uuid.NewV7()),
    Name:     input.Name,
    Email:    input.Email,
    Password: hashedPassword,
}
```

**Database Storage**:
- **PostgreSQL**: `UUID` type (native support)
- **MySQL**: `BINARY(16)` type (16-byte storage)

## 📋 Data Transfer Objects (DTOs)

The project enforces clear boundaries between internal domain models and external API contracts.

**Domain Models** (`internal/user/domain/user.go`):
- Pure internal representation of business entities
- No JSON tags - completely decoupled from API serialization
- Focus on business rules and domain logic

**DTOs** (`internal/user/http/dto/`):
- `request.go` - API request structures with validation
- `response.go` - API response structures with JSON tags
- `mapper.go` - Conversion functions between DTOs and domain models

**Benefits**:
1. 🔒 **Separation of Concerns** - Domain models evolve independently from API contracts
2. 🛡️ **Security** - Sensitive fields (like passwords) never exposed in API responses
3. 🔄 **Flexibility** - Different API views of same domain model
4. 📚 **Versioning** - Easy to maintain multiple API versions
5. ✅ **Validation** - Request validation happens at DTO level before reaching domain logic

## 🔄 Transaction Management

The template implements a TxManager interface for handling database transactions:

```go
type TxManager interface {
    WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

**Usage**:
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

The transaction is automatically injected into the context and used by repositories via `database.GetTx()`.

## 📤 Transactional Outbox Pattern

The project demonstrates the transactional outbox pattern for reliable event delivery using a use case-based architecture:

1. 📝 Business operation (e.g., user creation) is executed
2. 📬 Event is stored in outbox table in **same transaction**
3. 🚀 Outbox use case processes pending events with configurable retry logic
4. ✅ Events are marked as processed or failed
5. 🔌 Extensible via the `EventProcessor` interface for custom event handling

**Benefits**:
- 🔒 **Guaranteed delivery** - Events never lost due to transaction rollback
- 🔁 **At-least-once delivery** - Events processed at least once
- 🎯 **Consistency** - Business operations and events always in sync
- 🔧 **Extensibility** - Custom event processors for different event types

**Example (User Registration)**:
```go
err = uc.txManager.WithTx(ctx, func(ctx context.Context) error {
    // Create user
    if err := uc.userRepo.Create(ctx, user); err != nil {
        return err
    }
    
    // Create event in same transaction
    event := &outboxDomain.OutboxEvent{
        ID:        uuid.Must(uuid.NewV7()),
        EventType: "user.created",
        Payload:   userPayload,
        Status:    outboxDomain.StatusPending,
    }
    return uc.outboxRepo.Create(ctx, event)
})
```

**Processing Events**:

The outbox use case (`internal/outbox/usecase/outbox_usecase.go`) processes these events asynchronously:

```go
// Start the outbox event processor
outboxUseCase, err := container.OutboxUseCase()
if err != nil {
    return fmt.Errorf("failed to initialize outbox use case: %w", err)
}

// Processes events in background
err = outboxUseCase.Start(ctx)
```

**Custom Event Processing**:

You can create custom event processors by implementing the `EventProcessor` interface:

```go
type CustomEventProcessor struct {
    logger *slog.Logger
    // Add your dependencies here (e.g., message queue client)
}

func (p *CustomEventProcessor) Process(ctx context.Context, event *domain.OutboxEvent) error {
    // Your custom event processing logic
    switch event.EventType {
    case "user.created":
        // Send to message queue, send notification, etc.
        return p.publishToQueue(ctx, event)
    default:
        return fmt.Errorf("unknown event type: %s", event.EventType)
    }
}
```

Then register it in the DI container when initializing the outbox use case.
