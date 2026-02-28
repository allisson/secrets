package commands

import (
	"context"
	"errors"
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
func RunServer(ctx context.Context, version string) error {
	// Load configuration
	cfg := config.Load()

	// Set Gin mode based on log level
	gin.SetMode(cfg.GetGinMode())

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("starting server", slog.String("version", version))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get HTTP server from container (this initializes all dependencies)
	server, err := container.HTTPServer()
	if err != nil {
		return fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	// Get Metrics server from container
	metricsServer, err := container.MetricsServer()
	if err != nil {
		return fmt.Errorf("failed to initialize metrics server: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start servers in goroutines
	serverErr := make(chan error, 2)
	go func() {
		if err := server.Start(ctx); err != nil {
			serverErr <- fmt.Errorf("api server error: %w", err)
		}
	}()

	if metricsServer != nil {
		go func() {
			if err := metricsServer.Start(ctx); err != nil {
				serverErr <- fmt.Errorf("metrics server error: %w", err)
			}
		}()
	}

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.DBConnMaxLifetime)
		defer shutdownCancel()

		var shutdownErrors []error

		if err := server.Shutdown(shutdownCtx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("api server shutdown: %w", err))
		}

		if metricsServer != nil {
			if err := metricsServer.Shutdown(shutdownCtx); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics server shutdown: %w", err))
			}
		}

		if len(shutdownErrors) > 0 {
			return errors.Join(shutdownErrors...)
		}
	case err := <-serverErr:
		// Attempt graceful shutdown if one server fails
		logger.Error("server error, initiating shutdown", slog.Any("error", err))
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.DBConnMaxLifetime)
		defer shutdownCancel()

		var shutdownErrors []error
		shutdownErrors = append(shutdownErrors, err)

		if server != nil {
			if shutErr := server.Shutdown(shutdownCtx); shutErr != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("api server shutdown: %w", shutErr))
			}
		}

		if metricsServer != nil {
			if shutErr := metricsServer.Shutdown(shutdownCtx); shutErr != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("metrics server shutdown: %w", shutErr))
			}
		}

		return errors.Join(shutdownErrors...)
	}

	return nil
}
