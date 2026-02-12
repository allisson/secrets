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
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
)

// Server represents the HTTP server.
type Server struct {
	server *http.Server
	logger *slog.Logger
	router *gin.Engine
}

// NewServer creates a new HTTP server.
func NewServer(
	host string,
	port int,
	logger *slog.Logger,
) *Server {
	return &Server{
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
	clientHandler *authHTTP.ClientHandler,
	tokenHandler *authHTTP.TokenHandler,
	secretHandler *secretsHTTP.SecretHandler,
	tokenUseCase authUseCase.TokenUseCase,
	tokenService authService.TokenService,
	auditLogUseCase authUseCase.AuditLogUseCase,
) {
	// Create Gin engine without default middleware
	router := gin.New()

	// Apply custom middleware
	router.Use(gin.Recovery()) // Gin's panic recovery
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	}))) // Request ID with UUIDv7
	router.Use(CustomLoggerMiddleware(s.logger)) // Custom slog logger

	// Health and readiness endpoints (outside API versioning)
	router.GET("/health", s.healthHandler)
	router.GET("/ready", s.readinessHandler)

	// Create authentication middleware
	authMiddleware := authHTTP.AuthenticationMiddleware(
		tokenUseCase,
		tokenService,
		s.logger,
	)

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Token issuance endpoint (no authentication required)
		v1.POST("/token", tokenHandler.IssueTokenHandler)

		// Client management endpoints
		clients := v1.Group("/clients")
		clients.Use(authMiddleware) // All client routes require authentication
		{
			clients.POST("",
				authHTTP.AuthorizationMiddleware(authDomain.WriteCapability, auditLogUseCase, s.logger),
				clientHandler.CreateHandler,
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
		}

		// Secret management endpoints
		secrets := v1.Group("/secrets")
		secrets.Use(authMiddleware) // All secret routes require authentication
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
	}

	s.router = router
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
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// readinessHandler returns a simple readiness check response.
func (s *Server) readinessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
