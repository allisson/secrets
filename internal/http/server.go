// Package http provides HTTP server implementation and request handlers.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server represents the HTTP server
type Server struct {
	server *http.Server
	logger *slog.Logger
}

// NewServer creates a new HTTP server
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

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Create router
	mux := http.NewServeMux()

	// Health and readiness endpoints
	mux.Handle("/health", HealthHandler())
	mux.Handle("/ready", ReadinessHandler(ctx))

	// Apply middleware
	handler := ChainMiddleware(
		RecoveryMiddleware(s.logger),
		LoggingMiddleware(s.logger),
	)(mux)

	s.server.Handler = handler

	s.logger.Info("starting http server", slog.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.server.Shutdown(ctx)
}
