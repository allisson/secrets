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
	logger *slog.Logger,
) *Server {
	return &Server{
		db:     db,
		logger: logger,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", host, port),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// SetupRouter configures the Gin router with all routes and middleware.
// This method is called during server initialization with all required dependencies.
func (s *Server) SetupRouter(
	cfg *config.Config,
	clientHandler *authHTTP.ClientHandler,
	tokenHandler *authHTTP.TokenHandler,
	auditLogHandler *authHTTP.AuditLogHandler,
	secretHandler *secretsHTTP.SecretHandler,
	transitKeyHandler *transitHTTP.TransitKeyHandler,
	cryptoHandler *transitHTTP.CryptoHandler,
	tokenizationKeyHandler *tokenizationHTTP.TokenizationKeyHandler,
	tokenizationHandler *tokenizationHTTP.TokenizationHandler,
	tokenUseCase authUseCase.TokenUseCase,
	tokenService authService.TokenService,
	auditLogUseCase authUseCase.AuditLogUseCase,
	metricsProvider *metrics.Provider,
	metricsNamespace string,
) {
	// Create Gin engine without default middleware
	router := gin.New()

	// Apply custom middleware
	router.Use(gin.Recovery()) // Gin's panic recovery

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
	if metricsProvider != nil {
		router.Use(metrics.HTTPMetricsMiddleware(metricsProvider.MeterProvider(), metricsNamespace))
	}

	// Health and readiness endpoints (outside API versioning)
	router.GET("/health", s.healthHandler)
	router.GET("/ready", s.readinessHandler)

	// Create authentication middleware
	authMiddleware := authHTTP.AuthenticationMiddleware(
		tokenUseCase,
		tokenService,
		s.logger,
	)

	// Create rate limit middleware (applied to authenticated routes only)
	var rateLimitMiddleware gin.HandlerFunc
	if cfg.RateLimitEnabled {
		rateLimitMiddleware = authHTTP.RateLimitMiddleware(
			cfg.RateLimitRequestsPerSec,
			cfg.RateLimitBurst,
			s.logger,
		)
	}

	// Create token rate limit middleware (IP-based, for unauthenticated token endpoint)
	var tokenRateLimitMiddleware gin.HandlerFunc
	if cfg.RateLimitTokenEnabled {
		tokenRateLimitMiddleware = authHTTP.TokenRateLimitMiddleware(
			cfg.RateLimitTokenRequestsPerSec,
			cfg.RateLimitTokenBurst,
			s.logger,
		)
	}

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Token issuance endpoint (no authentication required, IP-based rate limiting)
		if tokenRateLimitMiddleware != nil {
			v1.POST("/token", tokenRateLimitMiddleware, tokenHandler.IssueTokenHandler)
		} else {
			v1.POST("/token", tokenHandler.IssueTokenHandler)
		}

		// Client management endpoints
		clients := v1.Group("/clients")
		clients.Use(authMiddleware) // All client routes require authentication
		if rateLimitMiddleware != nil {
			clients.Use(rateLimitMiddleware) // Apply rate limiting to authenticated clients
		}
		{
			clients.POST("",
				authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, auditLogUseCase, s.logger),
				clientHandler.CreateHandler,
			)
			clients.GET("",
				authHTTP.AuthorizationMiddleware(authDomain.ReadCapability, auditLogUseCase, s.logger),
				clientHandler.ListHandler,
			)
			clients.GET("/:id",
				authHTTP.AuthorizationMiddleware(authDomain.ReadCapability, auditLogUseCase, s.logger),
				clientHandler.GetHandler,
			)
			clients.PUT("/:id",
				authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, auditLogUseCase, s.logger),
				clientHandler.UpdateHandler,
			)
			clients.DELETE("/:id",
				authHTTP.AuthorizationMiddleware(authDomain.DeleteCapability, auditLogUseCase, s.logger),
				clientHandler.DeleteHandler,
			)
			clients.POST("/:id/unlock",
				authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, auditLogUseCase, s.logger),
				clientHandler.UnlockHandler,
			)
		}

		// Audit log endpoints
		auditLogs := v1.Group("/audit-logs")
		auditLogs.Use(authMiddleware) // All audit log routes require authentication
		if rateLimitMiddleware != nil {
			auditLogs.Use(rateLimitMiddleware) // Apply rate limiting to authenticated clients
		}
		{
			auditLogs.GET("",
				authHTTP.AuthorizationMiddleware(authDomain.ReadCapability, auditLogUseCase, s.logger),
				auditLogHandler.ListHandler,
			)
		}

		// Secret management endpoints
		secrets := v1.Group("/secrets")
		secrets.Use(authMiddleware) // All secret routes require authentication
		if rateLimitMiddleware != nil {
			secrets.Use(rateLimitMiddleware) // Apply rate limiting to authenticated clients
		}
		{
			secrets.POST("/*path",
				authHTTP.AuthorizationMiddleware(authDomain.EncryptCapability, auditLogUseCase, s.logger),
				secretHandler.CreateOrUpdateHandler,
			)
			secrets.GET("/*path",
				authHTTP.AuthorizationMiddleware(authDomain.DecryptCapability, auditLogUseCase, s.logger),
				secretHandler.GetHandler,
			)
			secrets.DELETE("/*path",
				authHTTP.AuthorizationMiddleware(authDomain.DeleteCapability, auditLogUseCase, s.logger),
				secretHandler.DeleteHandler,
			)
		}

		// Transit encryption endpoints
		transit := v1.Group("/transit")
		transit.Use(authMiddleware) // All transit routes require authentication
		if rateLimitMiddleware != nil {
			transit.Use(rateLimitMiddleware) // Apply rate limiting to authenticated clients
		}
		{
			keys := transit.Group("/keys")
			{
				// Create new transit key
				keys.POST("",
					authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, auditLogUseCase, s.logger),
					transitKeyHandler.CreateHandler,
				)

				// Rotate transit key to new version
				keys.POST("/:name/rotate",
					authHTTP.AuthorizationMiddleware(authDomain.RotateCapability, auditLogUseCase, s.logger),
					transitKeyHandler.RotateHandler,
				)

				// Delete transit key
				keys.DELETE("/:id",
					authHTTP.AuthorizationMiddleware(authDomain.DeleteCapability, auditLogUseCase, s.logger),
					transitKeyHandler.DeleteHandler,
				)

				// Encrypt plaintext with transit key
				keys.POST("/:name/encrypt",
					authHTTP.AuthorizationMiddleware(authDomain.EncryptCapability, auditLogUseCase, s.logger),
					cryptoHandler.EncryptHandler,
				)

				// Decrypt ciphertext with transit key
				keys.POST("/:name/decrypt",
					authHTTP.AuthorizationMiddleware(authDomain.DecryptCapability, auditLogUseCase, s.logger),
					cryptoHandler.DecryptHandler,
				)
			}
		}

		// Tokenization endpoints
		tokenization := v1.Group("/tokenization")
		tokenization.Use(authMiddleware) // All tokenization routes require authentication
		if rateLimitMiddleware != nil {
			tokenization.Use(rateLimitMiddleware) // Apply rate limiting to authenticated clients
		}
		{
			keys := tokenization.Group("/keys")
			{
				// Create new tokenization key
				keys.POST("",
					authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, auditLogUseCase, s.logger),
					tokenizationKeyHandler.CreateHandler,
				)

				// Rotate tokenization key to new version
				keys.POST("/:name/rotate",
					authHTTP.AuthorizationMiddleware(authDomain.RotateCapability, auditLogUseCase, s.logger),
					tokenizationKeyHandler.RotateHandler,
				)

				// Delete tokenization key
				keys.DELETE("/:id",
					authHTTP.AuthorizationMiddleware(authDomain.DeleteCapability, auditLogUseCase, s.logger),
					tokenizationKeyHandler.DeleteHandler,
				)

				// Tokenize plaintext with tokenization key
				keys.POST("/:name/tokenize",
					authHTTP.AuthorizationMiddleware(authDomain.EncryptCapability, auditLogUseCase, s.logger),
					tokenizationHandler.TokenizeHandler,
				)
			}

			// Detokenize token to retrieve plaintext
			tokenization.POST("/detokenize",
				authHTTP.AuthorizationMiddleware(authDomain.DecryptCapability, auditLogUseCase, s.logger),
				tokenizationHandler.DetokenizeHandler,
			)

			// Validate token existence and validity
			tokenization.POST("/validate",
				authHTTP.AuthorizationMiddleware(authDomain.ReadCapability, auditLogUseCase, s.logger),
				tokenizationHandler.ValidateHandler,
			)

			// Revoke token to prevent further detokenization
			tokenization.POST("/revoke",
				authHTTP.AuthorizationMiddleware(authDomain.DeleteCapability, auditLogUseCase, s.logger),
				tokenizationHandler.RevokeHandler,
			)
		}
	}

	s.router = router
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

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.server.Shutdown(ctx)
}

// healthHandler returns a simple health check response.
func (s *Server) healthHandler(c *gin.Context) {
	v, _, _ := s.reqGroup.Do("health", func() (interface{}, error) {
		return gin.H{"status": "healthy"}, nil
	})
	c.JSON(http.StatusOK, v)
}

type readinessResponse struct {
	StatusCode int
	Body       gin.H
}

// readinessHandler returns a simple readiness check response.
func (s *Server) readinessHandler(c *gin.Context) {
	v, _, _ := s.reqGroup.Do("readiness", func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	res := v.(readinessResponse)
	c.JSON(res.StatusCode, res.Body)
}
