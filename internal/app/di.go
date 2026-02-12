// Package app provides dependency injection container for assembling application components.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"

	authRepository "github.com/allisson/secrets/internal/auth/repository"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoRepository "github.com/allisson/secrets/internal/crypto/repository"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
	"github.com/allisson/secrets/internal/database"
	"github.com/allisson/secrets/internal/http"
)

// Container holds all application dependencies with lazy initialization.
type Container struct {
	// Configuration
	config *config.Config

	// Infrastructure
	logger         *slog.Logger
	db             *sql.DB
	masterKeyChain *cryptoDomain.MasterKeyChain

	// Managers
	txManager database.TxManager

	// Services
	aeadManager   cryptoService.AEADManager
	keyManager    cryptoService.KeyManager
	secretService authService.SecretService

	// Repositories
	kekRepository    cryptoUseCase.KekRepository
	clientRepository authUseCase.ClientRepository

	// Use Cases
	kekUseCase    cryptoUseCase.KekUseCase
	clientUseCase authUseCase.ClientUseCase

	// Servers and Workers
	httpServer *http.Server

	// Initialization flags and mutex for thread-safety
	mu                   sync.Mutex
	loggerInit           sync.Once
	dbInit               sync.Once
	masterKeyChainInit   sync.Once
	txManagerInit        sync.Once
	aeadManagerInit      sync.Once
	keyManagerInit       sync.Once
	secretServiceInit    sync.Once
	kekRepositoryInit    sync.Once
	clientRepositoryInit sync.Once
	kekUseCaseInit       sync.Once
	clientUseCaseInit    sync.Once
	httpServerInit       sync.Once
	initErrors           map[string]error
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
func (c *Container) Logger() *slog.Logger {
	c.loggerInit.Do(func() {
		c.logger = c.initLogger()
	})
	return c.logger
}

// DB returns the database connection.
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

// MasterKeyChain returns the master key chain loaded from environment variables.
func (c *Container) MasterKeyChain() (*cryptoDomain.MasterKeyChain, error) {
	var err error
	c.masterKeyChainInit.Do(func() {
		c.masterKeyChain, err = c.initMasterKeyChain()
		if err != nil {
			c.initErrors["masterKeyChain"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["masterKeyChain"]; exists {
		return nil, storedErr
	}
	return c.masterKeyChain, nil
}

// TxManager returns the transaction manager.
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

// AEADManager returns the AEAD manager service.
func (c *Container) AEADManager() cryptoService.AEADManager {
	c.aeadManagerInit.Do(func() {
		c.aeadManager = c.initAEADManager()
	})
	return c.aeadManager
}

// KeyManager returns the key manager service.
func (c *Container) KeyManager() cryptoService.KeyManager {
	c.keyManagerInit.Do(func() {
		c.keyManager = c.initKeyManager()
	})
	return c.keyManager
}

// KekRepository returns the KEK repository.
func (c *Container) KekRepository() (cryptoUseCase.KekRepository, error) {
	var err error
	c.kekRepositoryInit.Do(func() {
		c.kekRepository, err = c.initKekRepository()
		if err != nil {
			c.initErrors["kekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["kekRepository"]; exists {
		return nil, storedErr
	}
	return c.kekRepository, nil
}

// KekUseCase returns the KEK use case.
func (c *Container) KekUseCase() (cryptoUseCase.KekUseCase, error) {
	var err error
	c.kekUseCaseInit.Do(func() {
		c.kekUseCase, err = c.initKekUseCase()
		if err != nil {
			c.initErrors["kekUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["kekUseCase"]; exists {
		return nil, storedErr
	}
	return c.kekUseCase, nil
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

// SecretService returns the secret service for authentication operations.
func (c *Container) SecretService() authService.SecretService {
	c.secretServiceInit.Do(func() {
		c.secretService = c.initSecretService()
	})
	return c.secretService
}

// ClientRepository returns the client repository based on database driver.
func (c *Container) ClientRepository() (authUseCase.ClientRepository, error) {
	var err error
	c.clientRepositoryInit.Do(func() {
		c.clientRepository, err = c.initClientRepository()
		if err != nil {
			c.initErrors["clientRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["clientRepository"]; exists {
		return nil, storedErr
	}
	return c.clientRepository, nil
}

// ClientUseCase returns the client use case.
func (c *Container) ClientUseCase() (authUseCase.ClientUseCase, error) {
	var err error
	c.clientUseCaseInit.Do(func() {
		c.clientUseCase, err = c.initClientUseCase()
		if err != nil {
			c.initErrors["clientUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["clientUseCase"]; exists {
		return nil, storedErr
	}
	return c.clientUseCase, nil
}

// Shutdown performs cleanup of all initialized resources.
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

	// Close master key chain if initialized
	if c.masterKeyChain != nil {
		c.masterKeyChain.Close()
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

// initMasterKeyChain loads the master key chain from environment variables.
func (c *Container) initMasterKeyChain() (*cryptoDomain.MasterKeyChain, error) {
	masterKeyChain, err := cryptoDomain.LoadMasterKeyChainFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load master key chain: %w", err)
	}
	return masterKeyChain, nil
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

// initAEADManager creates the AEAD manager service.
func (c *Container) initAEADManager() cryptoService.AEADManager {
	return cryptoService.NewAEADManager()
}

// initKeyManager creates the key manager service using the AEAD manager.
func (c *Container) initKeyManager() cryptoService.KeyManager {
	aeadManager := c.AEADManager()
	return cryptoService.NewKeyManager(aeadManager)
}

// initKekRepository creates the KEK repository based on the database driver.
func (c *Container) initKekRepository() (cryptoUseCase.KekRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for kek repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoRepository.NewPostgreSQLKekRepository(db), nil
	case "mysql":
		return cryptoRepository.NewMySQLKekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initKekUseCase creates the KEK use case with all its dependencies.
func (c *Container) initKekUseCase() (cryptoUseCase.KekUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for kek use case: %w", err)
	}

	kekRepository, err := c.KekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get kek repository for kek use case: %w", err)
	}

	keyManager := c.KeyManager()

	return cryptoUseCase.NewKekUseCase(txManager, kekRepository, keyManager), nil
}

// initSecretService creates the secret service for authentication.
func (c *Container) initSecretService() authService.SecretService {
	return authService.NewSecretService()
}

// initClientRepository creates the client repository based on the database driver.
func (c *Container) initClientRepository() (authUseCase.ClientRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for client repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return authRepository.NewPostgreSQLClientRepository(db), nil
	case "mysql":
		return authRepository.NewMySQLClientRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initClientUseCase creates the client use case with all its dependencies.
func (c *Container) initClientUseCase() (authUseCase.ClientUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for client use case: %w", err)
	}

	clientRepository, err := c.ClientRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get client repository for client use case: %w", err)
	}

	secretService := c.SecretService()

	return authUseCase.NewClientUseCase(txManager, clientRepository, secretService), nil
}
