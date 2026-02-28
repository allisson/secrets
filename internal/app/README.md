# Dependency Injection Container

> Last updated: 2026-02-28

This package provides a dependency injection (DI) container for assembling and managing application components following Clean Architecture principles.

## Overview

The DI container centralizes the creation and wiring of all application dependencies, including:

- Infrastructure components (database, logger)
- Repositories (data access layer)
- Use cases (business logic layer)
- HTTP servers and handlers
- Background workers

## Key Features

### 1. Lazy Initialization
Components are only created when first accessed, improving startup time and memory usage.

```go
container := app.NewContainer(cfg)
// Nothing is initialized yet

server, err := container.HTTPServer()
// Now database, repositories, use cases, and server are initialized
```

### 2. Singleton Pattern
Each component is initialized only once and reused for subsequent calls.

```go
logger1 := container.Logger()
logger2 := container.Logger()
// logger1 == logger2 (same instance)
```

### 3. Error Handling
Initialization errors are captured and returned consistently:

```go
server, err := container.HTTPServer()
if err != nil {
    // Handle initialization error
}
```

### 4. Clean Shutdown
The container provides a unified shutdown method to clean up all resources:

```go
defer container.Shutdown(ctx)
```

## Architecture

### Dependency Graph

```
Container
├── Config (provided)
├── Logger
│   └── depends on: Config.LogLevel
├── Database
│   └── depends on: Config.DB*
├── TxManager
│   └── depends on: Database
├── Repositories
│   ├── UserRepository (interface from usecase package)
│   │   ├── MySQLUserRepository (concrete implementation)
│   │   ├── PostgreSQLUserRepository (concrete implementation)
│   │   └── depends on: Database
│   └── OutboxRepository (interface from usecase package)
│       ├── MySQLOutboxEventRepository (concrete implementation)
│       ├── PostgreSQLOutboxEventRepository (concrete implementation)
│       └── depends on: Database
├── Use Cases
│   └── UserUseCase
│       ├── depends on: TxManager
│       ├── depends on: UserRepository
│       └── depends on: OutboxRepository
├── HTTP Server
│   ├── depends on: Logger
│   └── depends on: UserUseCase
└── Event Worker
    ├── depends on: Logger
    ├── depends on: TxManager
    └── depends on: OutboxRepository
```

### Layer Separation

The container enforces clean architecture by managing dependencies at each layer:

1. **Infrastructure Layer**: Database connections, logger, transaction manager
2. **Data Layer**: Repositories for data access
3. **Business Layer**: Use cases with business logic
4. **Presentation Layer**: HTTP handlers and workers

## Usage Examples

### Starting the HTTP Server

```go
func runServer(ctx context.Context) error {
    // Load configuration
    cfg := config.Load()

    // Create DI container
    container := app.NewContainer(cfg)

    // Get logger
    logger := container.Logger()
    logger.Info("starting server")

    // Ensure cleanup on exit
    defer closeContainer(container, logger)

    // Get HTTP server (initializes all dependencies)
    server, err := container.HTTPServer()
    if err != nil {
        return fmt.Errorf("failed to initialize HTTP server: %w", err)
    }

    // Start server
    return server.Start(ctx)
}
```

### Starting the Worker

```go
func runWorker(ctx context.Context) error {
    cfg := config.Load()
    container := app.NewContainer(cfg)
    logger := container.Logger()

    defer closeContainer(container, logger)

    // Get event worker (initializes required dependencies)
    eventWorker, err := container.EventWorker()
    if err != nil {
        return fmt.Errorf("failed to initialize event worker: %w", err)
    }

    return eventWorker.Start(ctx)
}
```

## Testing

The container is designed to be easily testable:

### Unit Testing the Container

```go
func TestContainer(t *testing.T) {
    cfg := &config.Config{
        LogLevel: "info",
        // ... other config
    }

    container := app.NewContainer(cfg)
    logger := container.Logger()

    if logger == nil {
        t.Fatal("expected non-nil logger")
    }
}
```

### Integration Testing with Container

For integration tests, you can create a container with test configuration:

```go
func setupTestContainer(t *testing.T) *app.Container {
    cfg := &config.Config{
        DBDriver:           "postgres",
        DBConnectionString: "postgres://test:test@localhost:5432/test_db",
        LogLevel:          "debug",
    }

    container := app.NewContainer(cfg)
    t.Cleanup(func() {
        container.Shutdown(context.Background())
    })

    return container
}
```

## Adding New Components

To add a new component to the container:

### 1. Add field to Container struct

```go
type Container struct {
    // ... existing fields
    
    // New component
    orderUseCase     *orderUsecase.OrderUseCase
    orderUseCaseInit sync.Once
}
```

### 2. Add getter method

```go
func (c *Container) OrderUseCase() (*orderUsecase.OrderUseCase, error) {
    var err error
    c.orderUseCaseInit.Do(func() {
        c.orderUseCase, err = c.initOrderUseCase()
        if err != nil {
            c.initErrors["orderUseCase"] = err
        }
    })
    if err != nil {
        return nil, err
    }
    if storedErr, exists := c.initErrors["orderUseCase"]; exists {
        return nil, storedErr
    }
    return c.orderUseCase, nil
}
```

### 3. Add initialization method

```go
func (c *Container) initProductRepository() (productUsecase.ProductRepository, error) {
    db, err := c.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get database: %w", err)
    }
    
    // Select the appropriate repository based on the database driver
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

## Benefits of This Approach

### 1. Centralized Dependency Management
All component wiring is in one place (`internal/app/di.go`), making it easy to understand and maintain the application structure.

### 2. Clean main.go
The `main.go` file is significantly simpler and focused on application flow rather than dependency wiring.

**Before:**
```go
// 60+ lines of manual dependency wiring
db, err := database.Connect(...)
txManager := database.NewTxManager(db)

// Determine which repository to use
var userRepo userUsecase.UserRepository
switch cfg.DBDriver {
case "mysql":
    userRepo = userRepository.NewMySQLUserRepository(db)
case "postgres":
    userRepo = userRepository.NewPostgreSQLUserRepository(db)
}

var outboxRepo userUsecase.OutboxEventRepository
switch cfg.DBDriver {
case "mysql":
    outboxRepo = outboxRepository.NewMySQLOutboxEventRepository(db)
case "postgres":
    outboxRepo = outboxRepository.NewPostgreSQLOutboxEventRepository(db)
}

userUseCase, err := userUsecase.NewUserUseCase(txManager, userRepo, outboxRepo)
server := http.NewServer(cfg.ServerHost, cfg.ServerPort, logger, userUseCase)
```

**After:**
```go
// Clean and simple
container := app.NewContainer(cfg)
server, err := container.HTTPServer()
```

### 3. Testability
The container can be easily tested and mocked for integration tests.

### 4. Consistency
All parts of the application (server, worker, migrations) use the same dependency initialization logic.

### 5. Scalability
Adding new domains (orders, products, etc.) is straightforward - just add methods to the container.

### 6. Type Safety
All dependencies are type-checked at compile time, unlike reflection-based DI frameworks.

## Alternative Approaches

This implementation uses **manual dependency injection** with a container pattern. Other approaches include:

1. **Google Wire**: Code generation for compile-time DI
2. **Uber Fx**: Runtime reflection-based DI framework
3. **Pure Manual DI**: Direct construction in main.go (previous approach)

The current approach provides a good balance between:
- Simplicity (no external DI framework)
- Maintainability (centralized wiring)
- Performance (no reflection)
- Type safety (compile-time checking)

## Best Practices

1. **Always use defer for cleanup**: `defer closeContainer(container, logger)`
2. **Check initialization errors**: Always check errors returned by container methods
3. **Use lazy initialization**: Don't initialize components you don't need
4. **Keep interfaces**: Continue using interfaces for all dependencies
5. **Test the container**: Write tests for container initialization logic
6. **Document dependencies**: Keep the dependency graph documentation updated

## Thread Safety

The container uses `sync.Once` to ensure thread-safe lazy initialization. Multiple goroutines can safely call container methods concurrently.

## Performance Considerations

- **Lazy initialization** reduces startup time for commands that don't need all components
- **Singleton pattern** prevents creating duplicate instances
- **No reflection** ensures fast performance compared to reflection-based DI
- **Compile-time safety** catches dependency errors at build time

## Future Enhancements

Potential improvements for the container:

1. **Component lifecycle hooks**: Add `OnStart` and `OnStop` hooks
2. **Health checks**: Integrate health checking into the container
3. **Metrics**: Add metrics for component initialization time
4. **Configuration validation**: Validate configuration before initializing components
5. **Graceful degradation**: Support optional dependencies that can fail gracefully

## See also

- [Documentation index](../../docs/README.md)
- [Architecture concepts](../../docs/concepts/architecture.md)
- [Local development](../../docs/getting-started/local-development.md)
- [Development and testing](../../docs/contributing.md#development-and-testing)
