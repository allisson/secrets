// Package app provides dependency injection container for assembling application components.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/allisson/secrets/internal/config"
	"github.com/allisson/secrets/internal/database"
	"github.com/allisson/secrets/internal/http"
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

	// Servers and Workers
	httpServer *http.Server

	// Initialization flags and mutex for thread-safety
	mu             sync.Mutex
	loggerInit     sync.Once
	dbInit         sync.Once
	txManagerInit  sync.Once
	httpServerInit sync.Once
	initErrors     map[string]error
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

// initHTTPServer creates the HTTP server with all its dependencies.
func (c *Container) initHTTPServer() (*http.Server, error) {
	logger := c.Logger()

	server := http.NewServer(
		c.config.ServerHost,
		c.config.ServerPort,
		logger,
	)

	return server, nil
}
