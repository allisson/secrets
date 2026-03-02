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

	"github.com/allisson/secrets/internal/metrics"
)

// MetricsServer represents the HTTP server for Prometheus metrics.
type MetricsServer struct {
	server *http.Server // The underlying HTTP server
	logger *slog.Logger // Structured logger
}

// NewMetricsServer creates a new MetricsServer.
func NewMetricsServer(
	host string,
	port int,
	logger *slog.Logger,
	metricsProvider *metrics.Provider,
	readTimeout time.Duration,
	writeTimeout time.Duration,
	idleTimeout time.Duration,
) *MetricsServer {
	router := gin.New()
	router.Use(CustomRecoveryMiddleware(logger))
	router.Use(requestid.New(requestid.WithGenerator(func() string {
		return uuid.Must(uuid.NewV7()).String()
	})))
	router.Use(CustomLoggerMiddleware(logger))

	if metricsProvider != nil {
		router.GET("/metrics", gin.WrapH(metricsProvider.Handler()))
	}

	return &MetricsServer{
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", host, port),
			Handler:      router,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		logger: logger,
	}
}

// NewDefaultMetricsServer creates a new MetricsServer with default timeouts.
func NewDefaultMetricsServer(
	host string,
	port int,
	logger *slog.Logger,
	metricsProvider *metrics.Provider,
) *MetricsServer {
	return NewMetricsServer(
		host,
		port,
		logger,
		metricsProvider,
		15*time.Second,
		15*time.Second,
		60*time.Second,
	)
}

// GetHandler returns the http.Handler for testing purposes.
func (s *MetricsServer) GetHandler() http.Handler {
	return s.server.Handler
}

// Start starts the metrics HTTP server and handles context cancellation.
func (s *MetricsServer) Start(ctx context.Context) error {
	s.logger.Info("starting metrics server", slog.String("addr", s.server.Addr))

	// Channel to receive errors from ListenAndServe
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start metrics server: %w", err)
		}
		close(errChan)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("context cancelled, shutting down metrics server")
		return s.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the metrics HTTP server.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down metrics server")
	return s.server.Shutdown(ctx)
}
