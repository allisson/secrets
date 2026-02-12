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

// setupRouter configures the Gin router with all routes and middleware.
func (s *Server) setupRouter(ctx context.Context) *gin.Engine {
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
	router.GET("/ready", s.readinessHandler(ctx))

	// API v1 routes group (for future use)
	v1 := router.Group("/api/v1")
	{
		// Future business endpoints will go here
		// Example: v1.POST("/secrets", authMiddleware, s.createSecretHandler)
		_ = v1 // Prevent unused variable error
	}

	return router
}

// Start starts the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	// Setup router
	s.router = s.setupRouter(ctx)
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

// readinessHandler returns a readiness check handler that's context-aware.
func (s *Server) readinessHandler(shutdownCtx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		select {
		case <-shutdownCtx.Done():
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		default:
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
