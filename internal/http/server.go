// Package http provides HTTP server implementation and request handlers using Gin web framework.
// The server uses Clean Architecture principles with structured logging (slog) and graceful shutdown.
//
// This server uses Gin (github.com/gin-gonic/gin) for HTTP routing while maintaining
// compatibility with the application's existing patterns:
//   - Custom slog-based logging middleware (instead of Gin's default logger)
//   - Gin-compatible error handling utilities (httputil.HandleErrorGin)
//   - Manual http.Server configuration for timeout and graceful shutdown control
package http

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/config"
	"github.com/allisson/secrets/internal/metrics"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	transitHTTP "github.com/allisson/secrets/internal/transit/http"
)

// Server represents the HTTP server.
type Server struct {
	db       *sql.DB
	server   *http.Server
	logger   *slog.Logger
	router   *gin.Engine
	reqGroup singleflight.Group
}

// NewServer creates a new HTTP server.
func NewServer(
	db *sql.DB,
	host string,
	port int,
	readTimeout time.Duration,
	writeTimeout time.Duration,
	idleTimeout time.Duration,
	logger *slog.Logger,
) *Server {
	return &Server{
		db:     db,
		logger: logger,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", host, port),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}
}

// RouterDeps bundles the handler and usecase dependencies required by SetupRouter.
// Using a struct instead of positional parameters prevents silent field-swap bugs
// and makes adding a new handler a named-field change rather than a positional one.
type RouterDeps struct {
	ClientHandler          *authHTTP.ClientHandler
	TokenHandler           *authHTTP.TokenHandler
	AuditLogHandler        *authHTTP.AuditLogHandler
	SecretHandler          *secretsHTTP.SecretHandler
	TransitKeyHandler      *transitHTTP.TransitKeyHandler
	CryptoHandler          *transitHTTP.CryptoHandler
	TokenizationKeyHandler *tokenizationHTTP.TokenizationKeyHandler
	TokenizationHandler    *tokenizationHTTP.TokenizationHandler
	TokenUseCase           authUseCase.TokenUseCase
	TokenService           authService.TokenService
	AuditLogUseCase        authUseCase.AuditLogUseCase
	MetricsProvider        *metrics.Provider
	MetricsNamespace       string
}

// SetupRouter configures the Gin router with all routes and middleware.
// This method is called during server initialization with all required dependencies.
func (s *Server) SetupRouter(ctx context.Context, cfg *config.Config, deps RouterDeps) {
	// Create Gin engine without default middleware
	router := gin.New()

	// Apply custom middleware
	router.Use(CustomRecoveryMiddleware(s.logger)) // Custom slog panic recovery
	router.Use(MaxRequestBodySizeMiddleware(cfg.MaxRequestBodySize))

	// Add CORS middleware if enabled
	if corsMiddleware := createCORSMiddleware(
		cfg.CORSEnabled,
		cfg.CORSAllowOrigins,
		s.logger,
	); corsMiddleware != nil {
		router.Use(corsMiddleware)
	}

	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	}))) // Request ID with UUIDv7
	router.Use(CustomLoggerMiddleware(s.logger)) // Custom slog logger

	// Add HTTP metrics middleware if metrics are enabled
	if deps.MetricsProvider != nil {
		router.Use(metrics.HTTPMetricsMiddleware(deps.MetricsProvider.MeterProvider(), deps.MetricsNamespace))
	}

	// Health and readiness endpoints (outside API versioning)
	router.GET("/health", s.healthHandler)
	router.GET("/ready", s.readinessHandler)

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Create authentication middleware
		authMiddleware := authHTTP.AuthenticationMiddleware(
			deps.TokenUseCase,
			deps.TokenService,
			s.logger,
		)

		// Create rate limit middleware
		var rateLimitMiddleware gin.HandlerFunc
		if cfg.RateLimitEnabled {
			rateLimitMiddleware = authHTTP.RateLimitMiddleware(
				ctx,
				cfg.RateLimitRequestsPerSec,
				cfg.RateLimitBurst,
				s.logger,
			)
		}

		// Build the per-route authorizer once; it pre-binds audit + logger so
		// route registrations only carry the capability.
		authz := authHTTP.NewAuthorizer(deps.AuditLogUseCase, s.logger)

		s.registerAuthRoutes(
			ctx,
			v1,
			cfg,
			deps.ClientHandler,
			deps.TokenHandler,
			deps.AuditLogHandler,
			deps.TokenUseCase,
			deps.TokenService,
			authMiddleware,
			rateLimitMiddleware,
			authz,
		)
		s.registerSecretRoutes(v1, deps.SecretHandler, authMiddleware, rateLimitMiddleware, authz)
		s.registerTransitRoutes(
			v1,
			deps.TransitKeyHandler,
			deps.CryptoHandler,
			authMiddleware,
			rateLimitMiddleware,
			authz,
		)
		s.registerTokenizationRoutes(
			v1,
			deps.TokenizationKeyHandler,
			deps.TokenizationHandler,
			authMiddleware,
			rateLimitMiddleware,
			authz,
		)
	}

	s.router = router
}

// registerAuthRoutes configures the v1 authentication-related endpoints.
func (s *Server) registerAuthRoutes(
	ctx context.Context,
	v1 *gin.RouterGroup,
	cfg *config.Config,
	clientHandler *authHTTP.ClientHandler,
	tokenHandler *authHTTP.TokenHandler,
	auditLogHandler *authHTTP.AuditLogHandler,
	tokenUseCase authUseCase.TokenUseCase,
	tokenService authService.TokenService,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	authz *authHTTP.Authorizer,
) {
	// Create token rate limit middleware (IP-based, for unauthenticated token endpoint)
	var tokenRateLimitMiddleware gin.HandlerFunc
	if cfg.RateLimitTokenEnabled {
		tokenRateLimitMiddleware = authHTTP.TokenRateLimitMiddleware(
			ctx,
			cfg.RateLimitTokenRequestsPerSec,
			cfg.RateLimitTokenBurst,
			s.logger,
		)
	}

	// Token issuance endpoint (no authentication required, IP-based rate limiting)
	if tokenRateLimitMiddleware != nil {
		v1.POST("/token", tokenRateLimitMiddleware, tokenHandler.IssueTokenHandler)
	} else {
		v1.POST("/token", tokenHandler.IssueTokenHandler)
	}

	// Token revocation endpoint (requires authentication)
	v1.DELETE("/token", authMiddleware, tokenHandler.RevokeTokenHandler)

	// Client management endpoints
	clients := v1.Group("/clients")
	clients.Use(authMiddleware)
	if rateLimitMiddleware != nil {
		clients.Use(rateLimitMiddleware)
	}
	{
		clients.POST("",
			authz.Require(authDomain.WriteCapability),
			clientHandler.CreateHandler,
		)
		clients.GET("",
			authz.Require(authDomain.ReadCapability),
			clientHandler.ListHandler,
		)
		clients.GET("/:id",
			authz.Require(authDomain.ReadCapability),
			clientHandler.GetHandler,
		)
		clients.PUT("/:id",
			authz.Require(authDomain.WriteCapability),
			clientHandler.UpdateHandler,
		)
		clients.DELETE("/:id",
			authz.Require(authDomain.DeleteCapability),
			clientHandler.DeleteHandler,
		)
		clients.POST("/:id/unlock",
			authz.Require(authDomain.WriteCapability),
			clientHandler.UnlockHandler,
		)
		clients.POST("/:id/rotate-secret",
			authz.Require(authDomain.RotateCapability),
			clientHandler.RotateSecretHandler,
		)
		clients.DELETE("/:id/tokens",
			authz.Require(authDomain.DeleteCapability),
			clientHandler.RevokeTokensHandler,
		)
	}

	// Audit log endpoints
	auditLogs := v1.Group("/audit-logs")
	auditLogs.Use(authMiddleware)
	if rateLimitMiddleware != nil {
		auditLogs.Use(rateLimitMiddleware)
	}
	{
		auditLogs.GET("",
			authz.Require(authDomain.ReadCapability),
			auditLogHandler.ListHandler,
		)
	}
}

// registerSecretRoutes configures the v1 secret-related endpoints.
func (s *Server) registerSecretRoutes(
	v1 *gin.RouterGroup,
	secretHandler *secretsHTTP.SecretHandler,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	authz *authHTTP.Authorizer,
) {
	// Secret management endpoints
	secrets := v1.Group("/secrets")
	secrets.Use(authMiddleware)
	if rateLimitMiddleware != nil {
		secrets.Use(rateLimitMiddleware)
	}
	{
		secrets.GET("",
			authz.Require(authDomain.ReadCapability),
			secretHandler.ListHandler,
		)
		secrets.POST("/*path",
			authz.Require(authDomain.EncryptCapability),
			secretHandler.CreateOrUpdateHandler,
		)
		secrets.GET("/*path",
			authz.Require(authDomain.DecryptCapability),
			secretHandler.GetHandler,
		)
		secrets.DELETE("/*path",
			authz.Require(authDomain.DeleteCapability),
			secretHandler.DeleteHandler,
		)
	}
}

// registerTransitRoutes configures the v1 transit-encryption-related endpoints.
func (s *Server) registerTransitRoutes(
	v1 *gin.RouterGroup,
	transitKeyHandler *transitHTTP.TransitKeyHandler,
	cryptoHandler *transitHTTP.CryptoHandler,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	authz *authHTTP.Authorizer,
) {
	// Transit encryption endpoints
	transit := v1.Group("/transit")
	transit.Use(authMiddleware)
	if rateLimitMiddleware != nil {
		transit.Use(rateLimitMiddleware)
	}
	{
		keys := transit.Group("/keys")
		{
			// List transit keys
			keys.GET("",
				authz.Require(authDomain.ReadCapability),
				transitKeyHandler.ListHandler,
			)

			// Get individual transit key
			keys.GET("/:name",
				authz.Require(authDomain.ReadCapability),
				transitKeyHandler.GetHandler,
			)

			// Create new transit key
			keys.POST("",
				authz.Require(authDomain.WriteCapability),
				transitKeyHandler.CreateHandler,
			)

			// Rotate transit key to new version
			keys.POST("/:name/rotate",
				authz.Require(authDomain.RotateCapability),
				transitKeyHandler.RotateHandler,
			)

			// Delete transit key
			keys.DELETE("/:name",
				authz.Require(authDomain.DeleteCapability),
				transitKeyHandler.DeleteHandler,
			)

			// Encrypt plaintext with transit key
			keys.POST("/:name/encrypt",
				authz.Require(authDomain.EncryptCapability),
				cryptoHandler.EncryptHandler,
			)

			// Decrypt ciphertext with transit key
			keys.POST("/:name/decrypt",
				authz.Require(authDomain.DecryptCapability),
				cryptoHandler.DecryptHandler,
			)
		}
	}
}

// registerTokenizationRoutes configures the v1 tokenization-related endpoints.
func (s *Server) registerTokenizationRoutes(
	v1 *gin.RouterGroup,
	tokenizationKeyHandler *tokenizationHTTP.TokenizationKeyHandler,
	tokenizationHandler *tokenizationHTTP.TokenizationHandler,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	authz *authHTTP.Authorizer,
) {
	// Tokenization endpoints
	tokenization := v1.Group("/tokenization")
	tokenization.Use(authMiddleware)
	if rateLimitMiddleware != nil {
		tokenization.Use(rateLimitMiddleware)
	}
	{
		keys := tokenization.Group("/keys")
		{
			// List tokenization keys
			keys.GET("",
				authz.Require(authDomain.ReadCapability),
				tokenizationKeyHandler.ListHandler,
			)

			// Get individual tokenization key
			keys.GET("/:name",
				authz.Require(authDomain.ReadCapability),
				tokenizationKeyHandler.GetByNameHandler,
			)

			// Create new tokenization key
			keys.POST("",
				authz.Require(authDomain.WriteCapability),
				tokenizationKeyHandler.CreateHandler,
			)

			// Rotate tokenization key to new version
			keys.POST("/:name/rotate",
				authz.Require(authDomain.RotateCapability),
				tokenizationKeyHandler.RotateHandler,
			)

			// Delete tokenization key
			keys.DELETE("/:name",
				authz.Require(authDomain.DeleteCapability),
				tokenizationKeyHandler.DeleteHandler,
			)

			// Tokenize plaintext with tokenization key
			keys.POST("/:name/tokenize",
				authz.Require(authDomain.EncryptCapability),
				tokenizationHandler.TokenizeHandler,
			)

			// Tokenize batch of plaintexts with tokenization key
			keys.POST("/:name/tokenize-batch",
				authz.Require(authDomain.EncryptCapability),
				tokenizationHandler.TokenizeBatchHandler,
			)
		}

		// Detokenize token to retrieve plaintext
		tokenization.POST("/detokenize",
			authz.Require(authDomain.DecryptCapability),
			tokenizationHandler.DetokenizeHandler,
		)

		// Detokenize batch of tokens to retrieve plaintexts
		tokenization.POST("/detokenize-batch",
			authz.Require(authDomain.DecryptCapability),
			tokenizationHandler.DetokenizeBatchHandler,
		)

		// Validate token existence and validity
		tokenization.POST("/validate",
			authz.Require(authDomain.ReadCapability),
			tokenizationHandler.ValidateHandler,
		)

		// Revoke token to prevent further detokenization
		tokenization.POST("/revoke",
			authz.Require(authDomain.DeleteCapability),
			tokenizationHandler.RevokeHandler,
		)
	}
}

// GetHandler returns the http.Handler for testing purposes.
// Returns nil if SetupRouter has not been called yet.
func (s *Server) GetHandler() http.Handler {
	return s.router
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	// Router must be set up before starting
	if s.router == nil {
		return fmt.Errorf("router not initialized - call SetupRouter first")
	}

	s.server.Handler = s.router

	s.logger.Info("starting http server", slog.String("addr", s.server.Addr))

	// Channel to receive errors from ListenAndServe
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start server: %w", err)
		}
		close(errChan)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("context cancelled, shutting down http server")
		return s.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.server.Shutdown(ctx)
}

// healthHandler returns a simple health check response.
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// readinessResponse represents the structure of the readiness check response.
type readinessResponse struct {
	StatusCode int
	Body       gin.H
}

// readinessHandler returns a simple readiness check response.
func (s *Server) readinessHandler(c *gin.Context) {
	v, err, _ := s.reqGroup.Do("readiness", func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		dbStatus := "ok"
		httpStatus := http.StatusOK

		if s.db == nil {
			s.logger.Error("readiness check failed: database not initialized")
			dbStatus = "error"
			httpStatus = http.StatusServiceUnavailable
		} else if err := s.db.PingContext(ctx); err != nil {
			s.logger.Error("readiness check failed: database ping error", slog.Any("err", err))
			dbStatus = "error"
			httpStatus = http.StatusServiceUnavailable
		}

		return readinessResponse{
			StatusCode: httpStatus,
			Body: gin.H{
				"status": map[int]string{
					http.StatusOK:                 "ready",
					http.StatusServiceUnavailable: "not_ready",
				}[httpStatus],
				"components": gin.H{
					"database": dbStatus,
				},
			},
		}, nil
	})

	if err != nil {
		s.logger.Error("readiness check failed", slog.Any("err", err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "internal_readiness_error",
		})
		return
	}

	res, ok := v.(readinessResponse)
	if !ok {
		s.logger.Error("unexpected type from readiness check", slog.Any("value", v))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "not_ready",
			"error":  "internal_type_error",
		})
		return
	}

	c.JSON(res.StatusCode, res.Body)
}
