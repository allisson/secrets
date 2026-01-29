// Package app provides dependency injection container for assembling application components.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/allisson/go-project-template/internal/config"
	"github.com/allisson/go-project-template/internal/database"
	"github.com/allisson/go-project-template/internal/http"
	outboxRepository "github.com/allisson/go-project-template/internal/outbox/repository"
	outboxUsecase "github.com/allisson/go-project-template/internal/outbox/usecase"
	userRepository "github.com/allisson/go-project-template/internal/user/repository"
	userUsecase "github.com/allisson/go-project-template/internal/user/usecase"
)

// Container holds all application dependencies and provides methods to access them.
// It follows the lazy initialization pattern - components are created on first access.
type Container struct {
	// Configuration
	config *config.Config

	// Infrastructure
	logger *slog.Logger
	db     *sql.DB

	// Managers
	txManager database.TxManager

	// Repositories
	userRepo   userUsecase.UserRepository
	outboxRepo userUsecase.OutboxEventRepository

	// Use Cases
	userUseCase   userUsecase.UseCase
	outboxUseCase outboxUsecase.UseCase

	// Servers and Workers
	httpServer *http.Server

	// Initialization flags and mutex for thread-safety
	mu                sync.Mutex
	loggerInit        sync.Once
	dbInit            sync.Once
	txManagerInit     sync.Once
	userRepoInit      sync.Once
	outboxRepoInit    sync.Once
	userUseCaseInit   sync.Once
	outboxUseCaseInit sync.Once
	httpServerInit    sync.Once
	initErrors        map[string]error
}

// NewContainer creates a new dependency injection container with the provided configuration.
func NewContainer(cfg *config.Config) *Container {
	return &Container{
		config:     cfg,
		initErrors: make(map[string]error),
	}
}

// Config returns the application configuration.
func (c *Container) Config() *config.Config {
	return c.config
}

// Logger returns the configured logger instance.
// It creates a new logger on first access based on the log level in configuration.
func (c *Container) Logger() *slog.Logger {
	c.loggerInit.Do(func() {
		c.logger = c.initLogger()
	})
	return c.logger
}

// DB returns the database connection.
// It creates and configures the database connection on first access.
func (c *Container) DB() (*sql.DB, error) {
	var err error
	c.dbInit.Do(func() {
		c.db, err = c.initDB()
		if err != nil {
			c.initErrors["db"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["db"]; exists {
		return nil, storedErr
	}
	return c.db, nil
}

// TxManager returns the transaction manager.
// It requires a database connection to be initialized first.
func (c *Container) TxManager() (database.TxManager, error) {
	var err error
	c.txManagerInit.Do(func() {
		c.txManager, err = c.initTxManager()
		if err != nil {
			c.initErrors["txManager"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["txManager"]; exists {
		return nil, storedErr
	}
	return c.txManager, nil
}

// UserRepository returns the user repository instance.
func (c *Container) UserRepository() (userUsecase.UserRepository, error) {
	var err error
	c.userRepoInit.Do(func() {
		c.userRepo, err = c.initUserRepository()
		if err != nil {
			c.initErrors["userRepo"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["userRepo"]; exists {
		return nil, storedErr
	}
	return c.userRepo, nil
}

// OutboxRepository returns the outbox event repository instance.
func (c *Container) OutboxRepository() (userUsecase.OutboxEventRepository, error) {
	var err error
	c.outboxRepoInit.Do(func() {
		c.outboxRepo, err = c.initOutboxRepository()
		if err != nil {
			c.initErrors["outboxRepo"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["outboxRepo"]; exists {
		return nil, storedErr
	}
	return c.outboxRepo, nil
}

// UserUseCase returns the user use case instance.
func (c *Container) UserUseCase() (userUsecase.UseCase, error) {
	var err error
	c.userUseCaseInit.Do(func() {
		c.userUseCase, err = c.initUserUseCase()
		if err != nil {
			c.initErrors["userUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["userUseCase"]; exists {
		return nil, storedErr
	}
	return c.userUseCase, nil
}

// HTTPServer returns the HTTP server instance.
func (c *Container) HTTPServer() (*http.Server, error) {
	var err error
	c.httpServerInit.Do(func() {
		c.httpServer, err = c.initHTTPServer()
		if err != nil {
			c.initErrors["httpServer"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["httpServer"]; exists {
		return nil, storedErr
	}
	return c.httpServer, nil
}

// OutboxUseCase returns the outbox use case instance.
func (c *Container) OutboxUseCase() (outboxUsecase.UseCase, error) {
	var err error
	c.outboxUseCaseInit.Do(func() {
		c.outboxUseCase, err = c.initOutboxUseCase()
		if err != nil {
			c.initErrors["outboxUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["outboxUseCase"]; exists {
		return nil, storedErr
	}
	return c.outboxUseCase, nil
}

// Shutdown performs cleanup of all initialized resources.
// It should be called when the application is shutting down.
func (c *Container) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var shutdownErrors []error

	// Shutdown HTTP server if initialized
	if c.httpServer != nil {
		if err := c.httpServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("http server shutdown: %w", err))
		}
	}

	// Close database connection if initialized
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close: %w", err))
		}
	}

	// Return combined errors if any occurred
	if len(shutdownErrors) > 0 {
		return fmt.Errorf("shutdown errors: %v", shutdownErrors)
	}

	return nil
}

// initLogger creates and configures a structured logger based on the log level.
func (c *Container) initLogger() *slog.Logger {
	var logLevel slog.Level
	switch c.config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}

// initDB creates and configures the database connection.
func (c *Container) initDB() (*sql.DB, error) {
	db, err := database.Connect(database.Config{
		Driver:             c.config.DBDriver,
		ConnectionString:   c.config.DBConnectionString,
		MaxOpenConnections: c.config.DBMaxOpenConnections,
		MaxIdleConnections: c.config.DBMaxIdleConnections,
		ConnMaxLifetime:    c.config.DBConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// initTxManager creates the transaction manager using the database connection.
func (c *Container) initTxManager() (database.TxManager, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tx manager: %w", err)
	}
	return database.NewTxManager(db), nil
}

// initUserRepository creates the user repository instance.
func (c *Container) initUserRepository() (userUsecase.UserRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for user repository: %w", err)
	}

	// Select the appropriate repository based on the database driver
	switch c.config.DBDriver {
	case "mysql":
		return userRepository.NewMySQLUserRepository(db), nil
	case "postgres":
		return userRepository.NewPostgreSQLUserRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initOutboxRepository creates the outbox event repository instance.
func (c *Container) initOutboxRepository() (userUsecase.OutboxEventRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for outbox repository: %w", err)
	}

	// Select the appropriate repository based on the database driver
	switch c.config.DBDriver {
	case "mysql":
		return outboxRepository.NewMySQLOutboxEventRepository(db), nil
	case "postgres":
		return outboxRepository.NewPostgreSQLOutboxEventRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initUserUseCase creates the user use case with all its dependencies.
func (c *Container) initUserUseCase() (userUsecase.UseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for user use case: %w", err)
	}

	userRepo, err := c.UserRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get user repository for user use case: %w", err)
	}

	outboxRepo, err := c.OutboxRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox repository for user use case: %w", err)
	}

	useCase, err := userUsecase.NewUserUseCase(txManager, userRepo, outboxRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create user use case: %w", err)
	}

	return useCase, nil
}

// initHTTPServer creates the HTTP server with all its dependencies.
func (c *Container) initHTTPServer() (*http.Server, error) {
	logger := c.Logger()

	userUseCase, err := c.UserUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get user use case for http server: %w", err)
	}

	server := http.NewServer(
		c.config.ServerHost,
		c.config.ServerPort,
		logger,
		userUseCase,
	)

	return server, nil
}

// initOutboxUseCase creates the outbox use case with all its dependencies.
func (c *Container) initOutboxUseCase() (outboxUsecase.UseCase, error) {
	logger := c.Logger()

	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for outbox use case: %w", err)
	}

	outboxRepo, err := c.OutboxRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox repository for outbox use case: %w", err)
	}

	useCaseConfig := outboxUsecase.Config{
		Interval:      c.config.WorkerInterval,
		BatchSize:     c.config.WorkerBatchSize,
		MaxRetries:    c.config.WorkerMaxRetries,
		RetryInterval: c.config.WorkerRetryInterval,
	}

	eventProcessor := outboxUsecase.NewDefaultEventProcessor(logger)
	useCase := outboxUsecase.NewOutboxUseCase(useCaseConfig, txManager, outboxRepo, eventProcessor, logger)

	return useCase, nil
}
