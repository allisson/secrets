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

	"github.com/allisson/secrets/internal/config"
	"github.com/allisson/secrets/internal/metrics"
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

// RouteMiddlewares carries the shared per-route middleware a feature module
// applies to its route groups. Both fields are plain gin.HandlerFunc so this
// package stays free of any feature-specific types. RateLimit may be nil when
// rate limiting is disabled.
type RouteMiddlewares struct {
	Auth      gin.HandlerFunc
	RateLimit gin.HandlerFunc
}

// RouteRegistrar is implemented by each feature's route module. The composition
// root builds the concrete modules (capturing their handlers, authorizer, and
// business metrics) and hands them to SetupRouter, which knows nothing about
// any feature package.
type RouteRegistrar interface {
	Register(v1 *gin.RouterGroup, mw RouteMiddlewares)
}

// SetupRouter configures the Gin router with the global middleware chain and
// mounts each feature module under /v1. All feature-specific wiring lives in the
// registrars; this method is feature-agnostic.
func (s *Server) SetupRouter(
	cfg *config.Config,
	registrars []RouteRegistrar,
	mw RouteMiddlewares,
	metricsProvider *metrics.Provider,
	metricsNamespace string,
) {
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
	if metricsProvider != nil {
		router.Use(metrics.HTTPMetricsMiddleware(metricsProvider.MeterProvider(), metricsNamespace))
	}

	// Health and readiness endpoints (outside API versioning)
	router.GET("/health", s.healthHandler)
	router.GET("/ready", s.readinessHandler)

	// API v1 routes: each feature module mounts itself.
	v1 := router.Group("/v1")
	for _, r := range registrars {
		r.Register(v1, mw)
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
