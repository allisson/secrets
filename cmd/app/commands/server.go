package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunServer starts the HTTP server with graceful shutdown support.
// Loads configuration, initializes the DI container, and starts the Gin HTTP server.
// Blocks until receiving SIGINT/SIGTERM or encountering a fatal error. On shutdown
// signal, gracefully stops the server within DBConnMaxLifetime timeout.
func RunServer(ctx context.Context) error {
	// Load configuration
	cfg := config.Load()

	// Set Gin mode based on log level
	gin.SetMode(cfg.GetGinMode())

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("starting server", slog.String("version", "1.0.0"))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get HTTP server from container (this initializes all dependencies)
	server, err := container.HTTPServer()
	if err != nil {
		return fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(ctx); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.DBConnMaxLifetime)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	case err := <-serverErr:
		return err
	}

	return nil
}
