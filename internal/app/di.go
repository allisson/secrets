// Package app provides dependency injection container for assembling application components.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"

	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
	"github.com/allisson/secrets/internal/database"
	"github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/metrics"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
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

	// Metrics
	metricsProvider *metrics.Provider
	businessMetrics metrics.BusinessMetrics

	// Services
	aeadManager   cryptoService.AEADManager
	keyManager    cryptoService.KeyManager
	kmsService    cryptoService.KMSService
	secretService authService.SecretService
	tokenService  authService.TokenService

	// Repositories
	kekRepository               cryptoUseCase.KekRepository
	cryptoDekRepository         cryptoUseCase.DekRepository
	dekRepository               secretsUseCase.DekRepository
	secretRepository            secretsUseCase.SecretRepository
	clientRepository            authUseCase.ClientRepository
	tokenRepository             authUseCase.TokenRepository
	auditLogRepository          authUseCase.AuditLogRepository
	transitKeyRepository        transitUseCase.TransitKeyRepository
	transitDekRepository        transitUseCase.DekRepository
	tokenizationKeyRepository   tokenizationUseCase.TokenizationKeyRepository
	tokenizationTokenRepository tokenizationUseCase.TokenRepository
	tokenizationDekRepository   tokenizationUseCase.DekRepository

	// Use Cases
	kekUseCase             cryptoUseCase.KekUseCase
	cryptoDekUseCase       cryptoUseCase.DekUseCase
	secretUseCase          secretsUseCase.SecretUseCase
	clientUseCase          authUseCase.ClientUseCase
	tokenUseCase           authUseCase.TokenUseCase
	auditLogUseCase        authUseCase.AuditLogUseCase
	transitKeyUseCase      transitUseCase.TransitKeyUseCase
	tokenizationKeyUseCase tokenizationUseCase.TokenizationKeyUseCase
	tokenizationUseCase    tokenizationUseCase.TokenizationUseCase

	// HTTP Handlers
	clientHandler          *authHTTP.ClientHandler
	tokenHandler           *authHTTP.TokenHandler
	auditLogHandler        *authHTTP.AuditLogHandler
	secretHandler          *secretsHTTP.SecretHandler
	transitKeyHandler      *transitHTTP.TransitKeyHandler
	cryptoHandler          *transitHTTP.CryptoHandler
	tokenizationKeyHandler *tokenizationHTTP.TokenizationKeyHandler
	tokenizationHandler    *tokenizationHTTP.TokenizationHandler

	// Servers and Workers
	httpServer    *http.Server
	metricsServer *http.MetricsServer

	// Initialization flags and sync.Once for thread-safety
	loggerInit                      sync.Once
	dbInit                          sync.Once
	masterKeyChainInit              sync.Once
	txManagerInit                   sync.Once
	metricsProviderInit             sync.Once
	businessMetricsInit             sync.Once
	aeadManagerInit                 sync.Once
	keyManagerInit                  sync.Once
	kmsServiceInit                  sync.Once
	secretServiceInit               sync.Once
	tokenServiceInit                sync.Once
	kekRepositoryInit               sync.Once
	cryptoDekRepositoryInit         sync.Once
	dekRepositoryInit               sync.Once
	secretRepositoryInit            sync.Once
	clientRepositoryInit            sync.Once
	tokenRepositoryInit             sync.Once
	auditLogRepositoryInit          sync.Once
	transitKeyRepositoryInit        sync.Once
	transitDekRepositoryInit        sync.Once
	tokenizationKeyRepositoryInit   sync.Once
	tokenizationTokenRepositoryInit sync.Once
	tokenizationDekRepositoryInit   sync.Once
	kekUseCaseInit                  sync.Once
	cryptoDekUseCaseInit            sync.Once
	secretUseCaseInit               sync.Once
	clientUseCaseInit               sync.Once
	tokenUseCaseInit                sync.Once
	auditLogUseCaseInit             sync.Once
	transitKeyUseCaseInit           sync.Once
	tokenizationKeyUseCaseInit      sync.Once
	tokenizationUseCaseInit         sync.Once
	clientHandlerInit               sync.Once
	tokenHandlerInit                sync.Once
	auditLogHandlerInit             sync.Once
	secretHandlerInit               sync.Once
	transitKeyHandlerInit           sync.Once
	cryptoHandlerInit               sync.Once
	tokenizationKeyHandlerInit      sync.Once
	tokenizationHandlerInit         sync.Once
	httpServerInit                  sync.Once
	metricsServerInit               sync.Once
	initErrors                      map[string]error
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

// MetricsProvider returns the metrics provider for Prometheus export.
func (c *Container) MetricsProvider() (*metrics.Provider, error) {
	var err error
	c.metricsProviderInit.Do(func() {
		c.metricsProvider, err = c.initMetricsProvider()
		if err != nil {
			c.initErrors["metricsProvider"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["metricsProvider"]; exists {
		return nil, storedErr
	}
	return c.metricsProvider, nil
}

// BusinessMetrics returns the business metrics recorder.
func (c *Container) BusinessMetrics() (metrics.BusinessMetrics, error) {
	var err error
	c.businessMetricsInit.Do(func() {
		c.businessMetrics, err = c.initBusinessMetrics()
		if err != nil {
			c.initErrors["businessMetrics"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["businessMetrics"]; exists {
		return nil, storedErr
	}
	return c.businessMetrics, nil
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

// MetricsServer returns the Metrics server instance.
func (c *Container) MetricsServer() (*http.MetricsServer, error) {
	var err error
	c.metricsServerInit.Do(func() {
		c.metricsServer, err = c.initMetricsServer()
		if err != nil {
			c.initErrors["metricsServer"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["metricsServer"]; exists {
		return nil, storedErr
	}
	return c.metricsServer, nil
}

// Shutdown performs cleanup of all initialized resources.
func (c *Container) Shutdown(ctx context.Context) error {
	var shutdownErrors []error

	// Shutdown HTTP server if initialized
	if c.httpServer != nil {
		if err := c.httpServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("http server shutdown: %w", err))
		}
	}

	// Shutdown metrics provider if initialized
	if c.metricsProvider != nil {
		if err := c.metricsProvider.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics provider shutdown: %w", err))
		}
	}

	// Shutdown metrics server if initialized
	if c.metricsServer != nil {
		if err := c.metricsServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics server shutdown: %w", err))
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

// initTxManager creates the transaction manager using the database connection.
func (c *Container) initTxManager() (database.TxManager, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tx manager: %w", err)
	}
	return database.NewTxManager(db), nil
}

// initMetricsProvider creates the metrics provider if metrics are enabled.
func (c *Container) initMetricsProvider() (*metrics.Provider, error) {
	if !c.config.MetricsEnabled {
		return nil, nil
	}

	provider, err := metrics.NewProvider(c.config.MetricsNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics provider: %w", err)
	}
	return provider, nil
}

// initBusinessMetrics creates the business metrics recorder if metrics are enabled.
func (c *Container) initBusinessMetrics() (metrics.BusinessMetrics, error) {
	if !c.config.MetricsEnabled {
		return metrics.NewNoOpBusinessMetrics(), nil
	}

	provider, err := c.MetricsProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics provider: %w", err)
	}
	if provider == nil {
		return metrics.NewNoOpBusinessMetrics(), nil
	}

	businessMetrics, err := metrics.NewBusinessMetrics(provider.MeterProvider(), c.config.MetricsNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create business metrics: %w", err)
	}
	return businessMetrics, nil
}

// initHTTPServer creates the HTTP server with all its dependencies.
func (c *Container) initHTTPServer() (*http.Server, error) {
	logger := c.Logger()
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for http server: %w", err)
	}

	server := http.NewServer(
		db,
		c.config.ServerHost,
		c.config.ServerPort,
		logger,
	)

	// Get dependencies for routing
	clientHandler, err := c.ClientHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get client handler: %w", err)
	}

	tokenHandler, err := c.TokenHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get token handler: %w", err)
	}

	auditLogHandler, err := c.AuditLogHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log handler: %w", err)
	}

	secretHandler, err := c.SecretHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret handler: %w", err)
	}

	transitKeyHandler, err := c.TransitKeyHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key handler: %w", err)
	}

	cryptoHandler, err := c.CryptoHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get crypto handler: %w", err)
	}

	tokenizationKeyHandler, err := c.TokenizationKeyHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization key handler: %w", err)
	}

	tokenizationHandler, err := c.TokenizationHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization handler: %w", err)
	}

	tokenUseCase, err := c.TokenUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case: %w", err)
	}

	tokenService := c.TokenService()

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case: %w", err)
	}

	// Get metrics provider (may be nil if metrics are disabled)
	metricsProvider, err := c.MetricsProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics provider: %w", err)
	}

	// Setup router with dependencies
	server.SetupRouter(
		c.config,
		clientHandler,
		tokenHandler,
		auditLogHandler,
		secretHandler,
		transitKeyHandler,
		cryptoHandler,
		tokenizationKeyHandler,
		tokenizationHandler,
		tokenUseCase,
		tokenService,
		auditLogUseCase,
		metricsProvider,
		c.config.MetricsNamespace,
	)

	return server, nil
}

// initMetricsServer creates the Metrics server if metrics are enabled.
func (c *Container) initMetricsServer() (*http.MetricsServer, error) {
	if !c.config.MetricsEnabled {
		return nil, nil
	}

	logger := c.Logger()
	// Get metrics provider using existing accessor
	provider, err := c.MetricsProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics provider: %w", err)
	}

	server := http.NewMetricsServer(
		c.config.ServerHost,
		c.config.MetricsPort,
		logger,
		provider,
	)

	return server, nil
}
