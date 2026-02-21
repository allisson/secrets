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
	"github.com/allisson/secrets/internal/metrics"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	secretsRepository "github.com/allisson/secrets/internal/secrets/repository"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationRepository "github.com/allisson/secrets/internal/tokenization/repository"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitRepository "github.com/allisson/secrets/internal/transit/repository"
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
	httpServer *http.Server

	// Initialization flags and mutex for thread-safety
	mu                              sync.Mutex
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

// KMSService returns the KMS service.
func (c *Container) KMSService() cryptoService.KMSService {
	c.kmsServiceInit.Do(func() {
		c.kmsService = c.initKMSService()
	})
	return c.kmsService
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

// TokenService returns the token service for authentication operations.
func (c *Container) TokenService() authService.TokenService {
	c.tokenServiceInit.Do(func() {
		c.tokenService = c.initTokenService()
	})
	return c.tokenService
}

// TokenRepository returns the token repository based on database driver.
func (c *Container) TokenRepository() (authUseCase.TokenRepository, error) {
	var err error
	c.tokenRepositoryInit.Do(func() {
		c.tokenRepository, err = c.initTokenRepository()
		if err != nil {
			c.initErrors["tokenRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenRepository"]; exists {
		return nil, storedErr
	}
	return c.tokenRepository, nil
}

// AuditLogRepository returns the audit log repository based on database driver.
func (c *Container) AuditLogRepository() (authUseCase.AuditLogRepository, error) {
	var err error
	c.auditLogRepositoryInit.Do(func() {
		c.auditLogRepository, err = c.initAuditLogRepository()
		if err != nil {
			c.initErrors["auditLogRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["auditLogRepository"]; exists {
		return nil, storedErr
	}
	return c.auditLogRepository, nil
}

// TokenUseCase returns the token use case.
func (c *Container) TokenUseCase() (authUseCase.TokenUseCase, error) {
	var err error
	c.tokenUseCaseInit.Do(func() {
		c.tokenUseCase, err = c.initTokenUseCase()
		if err != nil {
			c.initErrors["tokenUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenUseCase"]; exists {
		return nil, storedErr
	}
	return c.tokenUseCase, nil
}

// AuditLogUseCase returns the audit log use case.
func (c *Container) AuditLogUseCase() (authUseCase.AuditLogUseCase, error) {
	var err error
	c.auditLogUseCaseInit.Do(func() {
		c.auditLogUseCase, err = c.initAuditLogUseCase()
		if err != nil {
			c.initErrors["auditLogUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["auditLogUseCase"]; exists {
		return nil, storedErr
	}
	return c.auditLogUseCase, nil
}

// ClientHandler returns the HTTP handler for client management operations.
func (c *Container) ClientHandler() (*authHTTP.ClientHandler, error) {
	var err error
	c.clientHandlerInit.Do(func() {
		c.clientHandler, err = c.initClientHandler()
		if err != nil {
			c.initErrors["clientHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["clientHandler"]; exists {
		return nil, storedErr
	}
	return c.clientHandler, nil
}

// TokenHandler returns the HTTP handler for token operations.
func (c *Container) TokenHandler() (*authHTTP.TokenHandler, error) {
	var err error
	c.tokenHandlerInit.Do(func() {
		c.tokenHandler, err = c.initTokenHandler()
		if err != nil {
			c.initErrors["tokenHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenHandler"]; exists {
		return nil, storedErr
	}
	return c.tokenHandler, nil
}

// AuditLogHandler returns the HTTP handler for audit log operations.
func (c *Container) AuditLogHandler() (*authHTTP.AuditLogHandler, error) {
	var err error
	c.auditLogHandlerInit.Do(func() {
		c.auditLogHandler, err = c.initAuditLogHandler()
		if err != nil {
			c.initErrors["auditLogHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["auditLogHandler"]; exists {
		return nil, storedErr
	}
	return c.auditLogHandler, nil
}

// DekRepository returns the DEK repository based on database driver.
func (c *Container) DekRepository() (secretsUseCase.DekRepository, error) {
	var err error
	c.dekRepositoryInit.Do(func() {
		c.dekRepository, err = c.initDekRepository()
		if err != nil {
			c.initErrors["dekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["dekRepository"]; exists {
		return nil, storedErr
	}
	return c.dekRepository, nil
}

// SecretRepository returns the secret repository based on database driver.
func (c *Container) SecretRepository() (secretsUseCase.SecretRepository, error) {
	var err error
	c.secretRepositoryInit.Do(func() {
		c.secretRepository, err = c.initSecretRepository()
		if err != nil {
			c.initErrors["secretRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["secretRepository"]; exists {
		return nil, storedErr
	}
	return c.secretRepository, nil
}

// SecretUseCase returns the secret use case.
func (c *Container) SecretUseCase() (secretsUseCase.SecretUseCase, error) {
	var err error
	c.secretUseCaseInit.Do(func() {
		c.secretUseCase, err = c.initSecretUseCase()
		if err != nil {
			c.initErrors["secretUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["secretUseCase"]; exists {
		return nil, storedErr
	}
	return c.secretUseCase, nil
}

// SecretHandler returns the HTTP handler for secret management operations.
func (c *Container) SecretHandler() (*secretsHTTP.SecretHandler, error) {
	var err error
	c.secretHandlerInit.Do(func() {
		c.secretHandler, err = c.initSecretHandler()
		if err != nil {
			c.initErrors["secretHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["secretHandler"]; exists {
		return nil, storedErr
	}
	return c.secretHandler, nil
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

	// Shutdown metrics provider if initialized
	if c.metricsProvider != nil {
		if err := c.metricsProvider.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics provider shutdown: %w", err))
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
	// Get KMS service and logger
	kmsService := c.KMSService()
	logger := c.Logger()

	// Load master key chain with KMS support and fail-fast validation
	masterKeyChain, err := cryptoDomain.LoadMasterKeyChain(
		context.Background(),
		c.config,
		kmsService,
		logger,
	)
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

	server := http.NewServer(
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

// initAEADManager creates the AEAD manager service.
func (c *Container) initAEADManager() cryptoService.AEADManager {
	return cryptoService.NewAEADManager()
}

// initKeyManager creates the key manager service using the AEAD manager.
func (c *Container) initKeyManager() cryptoService.KeyManager {
	aeadManager := c.AEADManager()
	return cryptoService.NewKeyManager(aeadManager)
}

// initKMSService creates the KMS service for encrypting/decrypting master keys.
func (c *Container) initKMSService() cryptoService.KMSService {
	return cryptoService.NewKMSService()
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

	baseUseCase := authUseCase.NewClientUseCase(txManager, clientRepository, secretService)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for client use case: %w", err)
		}
		return authUseCase.NewClientUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTokenService creates the token service for authentication.
func (c *Container) initTokenService() authService.TokenService {
	return authService.NewTokenService()
}

// initTokenRepository creates the token repository based on the database driver.
func (c *Container) initTokenRepository() (authUseCase.TokenRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for token repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return authRepository.NewPostgreSQLTokenRepository(db), nil
	case "mysql":
		return authRepository.NewMySQLTokenRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initAuditLogRepository creates the audit log repository based on the database driver.
func (c *Container) initAuditLogRepository() (authUseCase.AuditLogRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for audit log repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return authRepository.NewPostgreSQLAuditLogRepository(db), nil
	case "mysql":
		return authRepository.NewMySQLAuditLogRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTokenUseCase creates the token use case with all its dependencies.
func (c *Container) initTokenUseCase() (authUseCase.TokenUseCase, error) {
	clientRepository, err := c.ClientRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get client repository for token use case: %w", err)
	}

	tokenRepository, err := c.TokenRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for token use case: %w", err)
	}

	secretService := c.SecretService()
	tokenService := c.TokenService()

	baseUseCase := authUseCase.NewTokenUseCase(
		c.config,
		clientRepository,
		tokenRepository,
		secretService,
		tokenService,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for token use case: %w", err)
		}
		return authUseCase.NewTokenUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initAuditLogUseCase creates the audit log use case with all its dependencies.
func (c *Container) initAuditLogUseCase() (authUseCase.AuditLogUseCase, error) {
	auditLogRepository, err := c.AuditLogRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log repository for audit log use case: %w", err)
	}

	// Create audit signer service
	auditSigner := authService.NewAuditSigner()

	// Load KEK chain for signature verification
	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for audit log use case: %w", err)
	}

	baseUseCase := authUseCase.NewAuditLogUseCase(auditLogRepository, auditSigner, kekChain)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for audit log use case: %w", err)
		}
		return authUseCase.NewAuditLogUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initClientHandler creates the client HTTP handler with all its dependencies.
func (c *Container) initClientHandler() (*authHTTP.ClientHandler, error) {
	clientUseCase, err := c.ClientUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get client use case for client handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for client handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewClientHandler(clientUseCase, auditLogUseCase, logger), nil
}

// initTokenHandler creates the token HTTP handler with all its dependencies.
func (c *Container) initTokenHandler() (*authHTTP.TokenHandler, error) {
	tokenUseCase, err := c.TokenUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case for token handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewTokenHandler(tokenUseCase, logger), nil
}

// initAuditLogHandler creates the audit log HTTP handler with all its dependencies.
func (c *Container) initAuditLogHandler() (*authHTTP.AuditLogHandler, error) {
	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for audit log handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewAuditLogHandler(auditLogUseCase, logger), nil
}

// initDekRepository creates the DEK repository based on the database driver.
func (c *Container) initDekRepository() (secretsUseCase.DekRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for dek repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoRepository.NewPostgreSQLDekRepository(db), nil
	case "mysql":
		return cryptoRepository.NewMySQLDekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initSecretRepository creates the secret repository based on the database driver.
func (c *Container) initSecretRepository() (secretsUseCase.SecretRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for secret repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return secretsRepository.NewPostgreSQLSecretRepository(db), nil
	case "mysql":
		return secretsRepository.NewMySQLSecretRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initSecretUseCase creates the secret use case with all its dependencies.
func (c *Container) initSecretUseCase() (secretsUseCase.SecretUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for secret use case: %w", err)
	}

	dekRepository, err := c.DekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for secret use case: %w", err)
	}

	secretRepository, err := c.SecretRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret repository for secret use case: %w", err)
	}

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for secret use case: %w", err)
	}

	aeadManager := c.AEADManager()
	keyManager := c.KeyManager()

	baseUseCase := secretsUseCase.NewSecretUseCase(
		txManager,
		dekRepository,
		secretRepository,
		kekChain,
		aeadManager,
		keyManager,
		cryptoDomain.AESGCM,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for secret use case: %w", err)
		}
		return secretsUseCase.NewSecretUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initSecretHandler creates the secret HTTP handler with all its dependencies.
func (c *Container) initSecretHandler() (*secretsHTTP.SecretHandler, error) {
	secretUseCase, err := c.SecretUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret use case for secret handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for secret handler: %w", err)
	}

	logger := c.Logger()

	return secretsHTTP.NewSecretHandler(secretUseCase, auditLogUseCase, logger), nil
}

// loadKekChain loads all KEKs from the database and creates a KEK chain.
func (c *Container) loadKekChain() (*cryptoDomain.KekChain, error) {
	kekUseCase, err := c.KekUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get kek use case: %w", err)
	}

	masterKeyChain, err := c.MasterKeyChain()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key chain: %w", err)
	}

	// Unwrap all KEKs using the master key chain
	kekChain, err := kekUseCase.Unwrap(context.Background(), masterKeyChain)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap keks: %w", err)
	}

	return kekChain, nil
}

// TransitKeyRepository returns the transit key repository instance.
func (c *Container) TransitKeyRepository() (transitUseCase.TransitKeyRepository, error) {
	var err error
	c.transitKeyRepositoryInit.Do(func() {
		c.transitKeyRepository, err = c.initTransitKeyRepository()
		if err != nil {
			c.initErrors["transitKeyRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["transitKeyRepository"]; exists {
		return nil, storedErr
	}
	return c.transitKeyRepository, nil
}

// TransitDekRepository returns the DEK repository for transit use case.
func (c *Container) TransitDekRepository() (transitUseCase.DekRepository, error) {
	var err error
	c.transitDekRepositoryInit.Do(func() {
		c.transitDekRepository, err = c.initTransitDekRepository()
		if err != nil {
			c.initErrors["transitDekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["transitDekRepository"]; exists {
		return nil, storedErr
	}
	return c.transitDekRepository, nil
}

// TransitKeyUseCase returns the transit key use case instance.
func (c *Container) TransitKeyUseCase() (transitUseCase.TransitKeyUseCase, error) {
	var err error
	c.transitKeyUseCaseInit.Do(func() {
		c.transitKeyUseCase, err = c.initTransitKeyUseCase()
		if err != nil {
			c.initErrors["transitKeyUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["transitKeyUseCase"]; exists {
		return nil, storedErr
	}
	return c.transitKeyUseCase, nil
}

// TransitKeyHandler returns the transit key HTTP handler instance.
func (c *Container) TransitKeyHandler() (*transitHTTP.TransitKeyHandler, error) {
	var err error
	c.transitKeyHandlerInit.Do(func() {
		c.transitKeyHandler, err = c.initTransitKeyHandler()
		if err != nil {
			c.initErrors["transitKeyHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["transitKeyHandler"]; exists {
		return nil, storedErr
	}
	return c.transitKeyHandler, nil
}

// CryptoHandler returns the crypto HTTP handler instance.
func (c *Container) CryptoHandler() (*transitHTTP.CryptoHandler, error) {
	var err error
	c.cryptoHandlerInit.Do(func() {
		c.cryptoHandler, err = c.initCryptoHandler()
		if err != nil {
			c.initErrors["cryptoHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["cryptoHandler"]; exists {
		return nil, storedErr
	}
	return c.cryptoHandler, nil
}

// initTransitKeyRepository creates the transit key repository based on the database driver.
func (c *Container) initTransitKeyRepository() (transitUseCase.TransitKeyRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for transit key repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return transitRepository.NewPostgreSQLTransitKeyRepository(db), nil
	case "mysql":
		return transitRepository.NewMySQLTransitKeyRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTransitDekRepository creates the DEK repository for transit use case.
func (c *Container) initTransitDekRepository() (transitUseCase.DekRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for transit dek repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoRepository.NewPostgreSQLDekRepository(db), nil
	case "mysql":
		return cryptoRepository.NewMySQLDekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTransitKeyUseCase creates the transit key use case with all its dependencies.
func (c *Container) initTransitKeyUseCase() (transitUseCase.TransitKeyUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for transit key use case: %w", err)
	}

	transitKeyRepository, err := c.TransitKeyRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key repository for transit key use case: %w", err)
	}

	dekRepository, err := c.TransitDekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for transit key use case: %w", err)
	}

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for transit key use case: %w", err)
	}

	keyManager := c.KeyManager()
	aeadManager := c.AEADManager()

	baseUseCase := transitUseCase.NewTransitKeyUseCase(
		txManager,
		transitKeyRepository,
		dekRepository,
		keyManager,
		aeadManager,
		kekChain,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for transit key use case: %w", err)
		}
		return transitUseCase.NewTransitKeyUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTransitKeyHandler creates the transit key HTTP handler with all its dependencies.
func (c *Container) initTransitKeyHandler() (*transitHTTP.TransitKeyHandler, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for transit key handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for transit key handler: %w", err)
	}

	logger := c.Logger()

	return transitHTTP.NewTransitKeyHandler(transitKeyUseCase, auditLogUseCase, logger), nil
}

// initCryptoHandler creates the crypto HTTP handler with all its dependencies.
func (c *Container) initCryptoHandler() (*transitHTTP.CryptoHandler, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for crypto handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for crypto handler: %w", err)
	}

	logger := c.Logger()

	return transitHTTP.NewCryptoHandler(transitKeyUseCase, auditLogUseCase, logger), nil
}

// TokenizationKeyRepository returns the tokenization key repository.
func (c *Container) TokenizationKeyRepository() (tokenizationUseCase.TokenizationKeyRepository, error) {
	var err error
	c.tokenizationKeyRepositoryInit.Do(func() {
		c.tokenizationKeyRepository, err = c.initTokenizationKeyRepository()
		if err != nil {
			c.initErrors["tokenizationKeyRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationKeyRepository, c.initErrors["tokenizationKeyRepository"]
}

// TokenizationTokenRepository returns the tokenization token repository.
func (c *Container) TokenizationTokenRepository() (tokenizationUseCase.TokenRepository, error) {
	var err error
	c.tokenizationTokenRepositoryInit.Do(func() {
		c.tokenizationTokenRepository, err = c.initTokenizationTokenRepository()
		if err != nil {
			c.initErrors["tokenizationTokenRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationTokenRepository, c.initErrors["tokenizationTokenRepository"]
}

// TokenizationDekRepository returns the DEK repository for tokenization use case.
func (c *Container) TokenizationDekRepository() (tokenizationUseCase.DekRepository, error) {
	var err error
	c.tokenizationDekRepositoryInit.Do(func() {
		c.tokenizationDekRepository, err = c.initTokenizationDekRepository()
		if err != nil {
			c.initErrors["tokenizationDekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationDekRepository, c.initErrors["tokenizationDekRepository"]
}

// TokenizationKeyUseCase returns the tokenization key use case.
func (c *Container) TokenizationKeyUseCase() (tokenizationUseCase.TokenizationKeyUseCase, error) {
	var err error
	c.tokenizationKeyUseCaseInit.Do(func() {
		c.tokenizationKeyUseCase, err = c.initTokenizationKeyUseCase()
		if err != nil {
			c.initErrors["tokenizationKeyUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationKeyUseCase, c.initErrors["tokenizationKeyUseCase"]
}

// TokenizationUseCase returns the tokenization use case.
func (c *Container) TokenizationUseCase() (tokenizationUseCase.TokenizationUseCase, error) {
	var err error
	c.tokenizationUseCaseInit.Do(func() {
		c.tokenizationUseCase, err = c.initTokenizationUseCase()
		if err != nil {
			c.initErrors["tokenizationUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationUseCase, c.initErrors["tokenizationUseCase"]
}

// TokenizationKeyHandler returns the tokenization key HTTP handler.
func (c *Container) TokenizationKeyHandler() (*tokenizationHTTP.TokenizationKeyHandler, error) {
	var err error
	c.tokenizationKeyHandlerInit.Do(func() {
		c.tokenizationKeyHandler, err = c.initTokenizationKeyHandler()
		if err != nil {
			c.initErrors["tokenizationKeyHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationKeyHandler, c.initErrors["tokenizationKeyHandler"]
}

// TokenizationHandler returns the tokenization HTTP handler.
func (c *Container) TokenizationHandler() (*tokenizationHTTP.TokenizationHandler, error) {
	var err error
	c.tokenizationHandlerInit.Do(func() {
		c.tokenizationHandler, err = c.initTokenizationHandler()
		if err != nil {
			c.initErrors["tokenizationHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	return c.tokenizationHandler, c.initErrors["tokenizationHandler"]
}

// initTokenizationKeyRepository creates the tokenization key repository.
func (c *Container) initTokenizationKeyRepository() (tokenizationUseCase.TokenizationKeyRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization key repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return tokenizationRepository.NewPostgreSQLTokenizationKeyRepository(db), nil
	case "mysql":
		return tokenizationRepository.NewMySQLTokenizationKeyRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTokenizationTokenRepository creates the tokenization token repository.
func (c *Container) initTokenizationTokenRepository() (tokenizationUseCase.TokenRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization token repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return tokenizationRepository.NewPostgreSQLTokenRepository(db), nil
	case "mysql":
		return tokenizationRepository.NewMySQLTokenRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTokenizationDekRepository creates the DEK repository for tokenization use case.
func (c *Container) initTokenizationDekRepository() (tokenizationUseCase.DekRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization dek repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoRepository.NewPostgreSQLDekRepository(db), nil
	case "mysql":
		return cryptoRepository.NewMySQLDekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTokenizationKeyUseCase creates the tokenization key use case.
func (c *Container) initTokenizationKeyUseCase() (tokenizationUseCase.TokenizationKeyUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization key use case: %w", err)
	}

	tokenizationKeyRepository, err := c.TokenizationKeyRepository()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get tokenization key repository for tokenization key use case: %w",
			err,
		)
	}

	dekRepository, err := c.TokenizationDekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for tokenization key use case: %w", err)
	}

	keyManager := c.KeyManager()

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for tokenization key use case: %w", err)
	}

	baseUseCase := tokenizationUseCase.NewTokenizationKeyUseCase(
		txManager,
		tokenizationKeyRepository,
		dekRepository,
		keyManager,
		kekChain,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for tokenization key use case: %w", err)
		}
		return tokenizationUseCase.NewTokenizationKeyUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTokenizationUseCase creates the tokenization use case.
func (c *Container) initTokenizationUseCase() (tokenizationUseCase.TokenizationUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization use case: %w", err)
	}

	tokenizationKeyRepository, err := c.TokenizationKeyRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization key repository for tokenization use case: %w", err)
	}

	tokenRepository, err := c.TokenizationTokenRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for tokenization use case: %w", err)
	}

	dekRepository, err := c.TokenizationDekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for tokenization use case: %w", err)
	}

	aeadManager := c.AEADManager()

	keyManager := c.KeyManager()

	hashService := tokenizationUseCase.NewSHA256HashService()

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for tokenization use case: %w", err)
	}

	baseUseCase := tokenizationUseCase.NewTokenizationUseCase(
		txManager,
		tokenizationKeyRepository,
		tokenRepository,
		dekRepository,
		aeadManager,
		keyManager,
		hashService,
		kekChain,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for tokenization use case: %w", err)
		}
		return tokenizationUseCase.NewTokenizationUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTokenizationKeyHandler creates the tokenization key HTTP handler.
func (c *Container) initTokenizationKeyHandler() (*tokenizationHTTP.TokenizationKeyHandler, error) {
	tokenizationKeyUseCase, err := c.TokenizationKeyUseCase()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get tokenization key use case for tokenization key handler: %w",
			err,
		)
	}

	logger := c.Logger()

	return tokenizationHTTP.NewTokenizationKeyHandler(tokenizationKeyUseCase, logger), nil
}

// initTokenizationHandler creates the tokenization HTTP handler.
func (c *Container) initTokenizationHandler() (*tokenizationHTTP.TokenizationHandler, error) {
	tokenizationUseCase, err := c.TokenizationUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization use case for tokenization handler: %w", err)
	}

	logger := c.Logger()

	return tokenizationHTTP.NewTokenizationHandler(tokenizationUseCase, logger), nil
}
