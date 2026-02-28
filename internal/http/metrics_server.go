package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/allisson/secrets/internal/metrics"
)

// MetricsServer represents the HTTP server for Prometheus metrics.
type MetricsServer struct {
	server *http.Server
	logger *slog.Logger
}

// NewMetricsServer creates a new MetricsServer.
func NewMetricsServer(
	host string,
	port int,
	logger *slog.Logger,
	metricsProvider *metrics.Provider,
) *MetricsServer {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(CustomLoggerMiddleware(logger))

	if metricsProvider != nil {
		router.GET("/metrics", gin.WrapH(metricsProvider.Handler()))
	}

	return &MetricsServer{
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", host, port),
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// GetHandler returns the http.Handler for testing purposes.
func (s *MetricsServer) GetHandler() http.Handler {
	return s.server.Handler
}

// Start starts the metrics HTTP server.
func (s *MetricsServer) Start(ctx context.Context) error {
	s.logger.Info("starting metrics server", slog.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the metrics HTTP server.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down metrics server")
	return s.server.Shutdown(ctx)
}
