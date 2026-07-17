// Package app provides dependency injection container for assembling application components.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/allisson/go-pwdhash"
	"github.com/gin-gonic/gin"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/config"
	"github.com/allisson/secrets/internal/database"
	"github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/keyring"
	"github.com/allisson/secrets/internal/metrics"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
)

// Container holds all application dependencies with lazy initialization.
type Container struct {
	config *config.Config

	// Infallible singletons — kept as plain sync.Once because they cannot fail.
	logger             *slog.Logger
	loggerInit         sync.Once
	kmsService         keyring.KMSService
	kmsServiceInit     sync.Once
	passwordHasher     *pwdhash.PasswordHasher
	passwordHasherInit sync.Once

	// Infrastructure
	db             once[*sql.DB]
	masterKeyChain once[*keyring.MasterKeyChain]
	txManager      once[database.TxManager]

	// Metrics
	metricsProvider once[*metrics.Provider]
	businessMetrics once[metrics.BusinessMetrics]

	// Keyring (envelope encryption)
	keyring once[keyring.Keyring]

	// Repositories
	secretRepository            once[secretsDomain.SecretRepository]
	clientRepository            once[authDomain.ClientRepository]
	tokenRepository             once[authDomain.TokenRepository]
	auditLogRepository          once[authDomain.AuditLogRepository]
	transitKeyRepository        once[transitDomain.TransitKeyRepository]
	tokenizationKeyRepository   once[tokenizationDomain.TokenizationKeyRepository]
	tokenizationTokenRepository once[tokenizationDomain.TokenRepository]

	// Use Cases
	kekUseCase             once[keyring.KekUseCase]
	secretUseCase          once[secretsUseCase.SecretUseCase]
	clientUseCase          once[authUseCase.ClientUseCase]
	tokenUseCase           once[authUseCase.TokenUseCase]
	auditLogUseCase        once[authUseCase.AuditLogUseCase]
	transitKeyUseCase      once[transitUseCase.TransitKeyUseCase]
	tokenizationKeyUseCase once[tokenizationUseCase.TokenizationKeyUseCase]
	tokenizationUseCase    once[tokenizationUseCase.TokenizationUseCase]

	// HTTP Handlers
	clientHandler          once[*authHTTP.ClientHandler]
	tokenHandler           once[*authHTTP.TokenHandler]
	auditLogHandler        once[*authHTTP.AuditLogHandler]
	secretHandler          once[*secretsHTTP.SecretHandler]
	transitKeyHandler      once[*transitHTTP.TransitKeyHandler]
	cryptoHandler          once[*transitHTTP.CryptoHandler]
	tokenizationKeyHandler once[*tokenizationHTTP.TokenizationKeyHandler]
	tokenizationHandler    once[*tokenizationHTTP.TokenizationHandler]

	// Servers
	httpServer    once[*http.Server]
	metricsServer once[*http.MetricsServer]
}

// NewContainer creates a new dependency injection container with the provided configuration.
func NewContainer(cfg *config.Config) *Container {
	return &Container{
		config: cfg,
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
func (c *Container) DB(ctx context.Context) (*sql.DB, error) {
	return c.db.get(func() (*sql.DB, error) {
		return c.initDB(ctx)
	})
}

// TxManager returns the transaction manager.
func (c *Container) TxManager(ctx context.Context) (database.TxManager, error) {
	return c.txManager.get(func() (database.TxManager, error) {
		return c.initTxManager(ctx)
	})
}

// MetricsProvider returns the metrics provider for Prometheus export.
func (c *Container) MetricsProvider(ctx context.Context) (*metrics.Provider, error) {
	return c.metricsProvider.get(func() (*metrics.Provider, error) {
		return c.initMetricsProvider(ctx)
	})
}

// BusinessMetrics returns the business metrics recorder.
func (c *Container) BusinessMetrics(ctx context.Context) (metrics.BusinessMetrics, error) {
	return c.businessMetrics.get(func() (metrics.BusinessMetrics, error) {
		return c.initBusinessMetrics(ctx)
	})
}

// HTTPServer returns the HTTP server instance.
func (c *Container) HTTPServer(ctx context.Context) (*http.Server, error) {
	return c.httpServer.get(func() (*http.Server, error) {
		return c.initHTTPServer(ctx)
	})
}

// MetricsServer returns the Metrics server instance.
func (c *Container) MetricsServer(ctx context.Context) (*http.MetricsServer, error) {
	return c.metricsServer.get(func() (*http.MetricsServer, error) {
		return c.initMetricsServer(ctx)
	})
}

// Shutdown performs cleanup of all initialized resources.
func (c *Container) Shutdown(ctx context.Context) error {
	var shutdownErrors []error

	if c.httpServer.val != nil {
		if err := c.httpServer.val.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("http server shutdown: %w", err))
		}
	}

	if c.metricsProvider.val != nil {
		if err := c.metricsProvider.val.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics provider shutdown: %w", err))
		}
	}

	if c.metricsServer.val != nil {
		if err := c.metricsServer.val.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics server shutdown: %w", err))
		}
	}

	if c.masterKeyChain.val != nil {
		c.masterKeyChain.val.Close()
	}

	if c.db.val != nil {
		if err := c.db.val.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close: %w", err))
		}
	}

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
func (c *Container) initDB(ctx context.Context) (*sql.DB, error) {
	db, err := database.Connect(database.Config{
		ConnectionString:   c.config.DBConnectionString,
		MaxOpenConnections: c.config.DBMaxOpenConnections,
		MaxIdleConnections: c.config.DBMaxIdleConnections,
		ConnMaxLifetime:    c.config.DBConnMaxLifetime,
		ConnMaxIdleTime:    c.config.DBConnMaxIdleTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// initTxManager creates the transaction manager using the database connection.
func (c *Container) initTxManager(ctx context.Context) (database.TxManager, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tx manager: %w", err)
	}
	return database.NewTxManager(db), nil
}

// initMetricsProvider creates the metrics provider if metrics are enabled.
func (c *Container) initMetricsProvider(ctx context.Context) (*metrics.Provider, error) {
	if !c.config.MetricsEnabled {
		return nil, nil
	}

	provider, err := metrics.NewProvider(c.config.MetricsNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics provider: %w", err)
	}
	return provider, nil
}

// initBusinessMetrics creates the business metrics recorder.
// Returns a no-op implementation when metrics are disabled so callers never receive nil.
func (c *Container) initBusinessMetrics(ctx context.Context) (metrics.BusinessMetrics, error) {
	if !c.config.MetricsEnabled {
		return metrics.NewNopBusinessMetrics(), nil
	}

	provider, err := c.MetricsProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics provider: %w", err)
	}

	bm, err := metrics.NewBusinessMetrics(provider.MeterProvider(), c.config.MetricsNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create business metrics: %w", err)
	}
	return bm, nil
}

// initHTTPServer creates the HTTP server with all its dependencies.
func (c *Container) initHTTPServer(ctx context.Context) (*http.Server, error) {
	logger := c.Logger()
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for http server: %w", err)
	}

	server := http.NewServer(
		db,
		c.config.ServerHost,
		c.config.ServerPort,
		c.config.ServerReadTimeout,
		c.config.ServerWriteTimeout,
		c.config.ServerIdleTimeout,
		logger,
	)

	clientHandler, err := c.ClientHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client handler: %w", err)
	}

	tokenHandler, err := c.TokenHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token handler: %w", err)
	}

	auditLogHandler, err := c.AuditLogHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log handler: %w", err)
	}

	secretHandler, err := c.SecretHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret handler: %w", err)
	}

	transitKeyHandler, err := c.TransitKeyHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key handler: %w", err)
	}

	cryptoHandler, err := c.CryptoHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get crypto handler: %w", err)
	}

	tokenizationKeyHandler, err := c.TokenizationKeyHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization key handler: %w", err)
	}

	tokenizationHandler, err := c.TokenizationHandler(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization handler: %w", err)
	}

	tokenUseCase, err := c.TokenUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case: %w", err)
	}

	metricsProvider, err := c.MetricsProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics provider: %w", err)
	}

	businessMetrics, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics: %w", err)
	}

	// Build the shared per-route middleware. Authentication is always present;
	// the two rate limiters are optional and stay nil when disabled.
	authMiddleware := authHTTP.AuthenticationMiddleware(tokenUseCase, logger)

	var rateLimitMiddleware gin.HandlerFunc
	if c.config.RateLimitEnabled {
		rateLimitMiddleware = authHTTP.RateLimitMiddleware(
			ctx,
			c.config.RateLimitRequestsPerSec,
			c.config.RateLimitBurst,
			logger,
		)
	}

	var tokenRateLimitMiddleware gin.HandlerFunc
	if c.config.RateLimitTokenEnabled {
		tokenRateLimitMiddleware = authHTTP.TokenRateLimitMiddleware(
			ctx,
			c.config.RateLimitTokenRequestsPerSec,
			c.config.RateLimitTokenBurst,
			logger,
		)
	}

	// Build the per-route authorizer once; each module captures it so route
	// registrations only carry the capability.
	authz := authHTTP.NewAuthorizer(auditLogUseCase, logger)

	// Each feature owns its route registration; the composition root assembles
	// the modules and the server mounts them without knowing any feature type.
	registrars := []http.RouteRegistrar{
		authHTTP.NewModule(
			clientHandler,
			tokenHandler,
			auditLogHandler,
			authz,
			businessMetrics,
			tokenRateLimitMiddleware,
		),
		secretsHTTP.NewModule(secretHandler, authz, businessMetrics),
		transitHTTP.NewModule(transitKeyHandler, cryptoHandler, authz, businessMetrics),
		tokenizationHTTP.NewModule(tokenizationKeyHandler, tokenizationHandler, authz, businessMetrics),
	}

	server.SetupRouter(
		c.config,
		registrars,
		http.RouteMiddlewares{Auth: authMiddleware, RateLimit: rateLimitMiddleware},
		metricsProvider,
		c.config.MetricsNamespace,
	)

	return server, nil
}

// initMetricsServer creates the Metrics server if metrics are enabled.
func (c *Container) initMetricsServer(ctx context.Context) (*http.MetricsServer, error) {
	if !c.config.MetricsEnabled {
		return nil, nil
	}

	logger := c.Logger()
	provider, err := c.MetricsProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics provider: %w", err)
	}

	server := http.NewDefaultMetricsServer(
		c.config.ServerHost,
		c.config.MetricsPort,
		logger,
		provider,
		c.config.MetricsServerReadTimeout,
		c.config.MetricsServerWriteTimeout,
		c.config.MetricsServerIdleTimeout,
	)

	return server, nil
}
