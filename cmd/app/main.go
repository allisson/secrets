// Package main provides the entry point for the application with CLI commands.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/urfave/cli/v3"

	"github.com/allisson/go-project-template/internal/app"
	"github.com/allisson/go-project-template/internal/config"
)

// closeContainer closes all resources in the container and logs any errors.
func closeContainer(container *app.Container, logger *slog.Logger) {
	if err := container.Shutdown(context.Background()); err != nil {
		logger.Error("failed to shutdown container", slog.Any("error", err))
	}
}

// closeMigrate closes the migration instance and logs any errors.
func closeMigrate(migrate *migrate.Migrate, logger *slog.Logger) {
	sourceError, databaseError := migrate.Close()
	if sourceError != nil || databaseError != nil {
		logger.Error(
			"failed to close the migrate",
			slog.Any("source_error", sourceError),
			slog.Any("database_error", databaseError),
		)
	}
}

func main() {
	cmd := &cli.Command{
		Name:    "app",
		Usage:   "Go project template application",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start the HTTP server",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runServer(ctx)
				},
			},
			{
				Name:  "migrate",
				Usage: "Run database migrations",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runMigrations()
				},
			},
			{
				Name:  "worker",
				Usage: "Run the event worker",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runWorker(ctx)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}

// runServer starts the HTTP server with graceful shutdown support.
func runServer(ctx context.Context) error {
	// Load configuration
	cfg := config.Load()

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

// runMigrations executes database migrations based on the configured driver.
func runMigrations() error {
	cfg := config.Load()

	// Create container just for logger
	container := app.NewContainer(cfg)
	logger := container.Logger()

	logger.Info("running database migrations",
		slog.String("driver", cfg.DBDriver),
	)

	// Determine migration path based on driver
	migrationsPath := "file://migrations/postgresql"
	if cfg.DBDriver == "mysql" {
		migrationsPath = "file://migrations/mysql"
	}

	m, err := migrate.New(migrationsPath, cfg.DBConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer closeMigrate(m, logger)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("migrations completed successfully")
	return nil
}

// runWorker starts the outbox event processor with graceful shutdown support.
func runWorker(ctx context.Context) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("starting outbox event processor", slog.String("version", "1.0.0"))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get outbox use case from container (this initializes all dependencies)
	outboxUseCase, err := container.OutboxUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize outbox use case: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start outbox event processor
	return outboxUseCase.Start(ctx)
}
