package commands

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsPath = "file://migrations"

// RunMigrations executes all pending PostgreSQL database migrations.
// Returns nil if no migrations apply. Logs migration progress and success.
func RunMigrations(logger *slog.Logger, dbConnectionString string) error {
	logger.Info("running database migrations")

	m, err := migrate.New(migrationsPath, dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer CloseMigrate(m, logger)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("migrations completed successfully")
	return nil
}

// RunMigrationsDown rolls back PostgreSQL database migrations.
// Returns nil if no migrations rollback. Logs migration progress and success.
func RunMigrationsDown(logger *slog.Logger, dbConnectionString string, steps int) error {
	logger.Info("rolling back database migrations",
		slog.Int("steps", steps),
	)

	m, err := migrate.New(migrationsPath, dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer CloseMigrate(m, logger)

	if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	logger.Info("migrations rolled back successfully")
	return nil
}
