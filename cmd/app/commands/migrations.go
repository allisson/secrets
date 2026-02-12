package commands

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunMigrations executes database migrations based on the configured driver.
// Determines migration path from DBDriver (postgresql or mysql) and applies all pending
// migrations. Returns nil if no migrations to apply. Logs migration progress and success.
func RunMigrations() error {
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
