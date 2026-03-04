package commands

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunMigrations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("invalid-driver", func(t *testing.T) {
		err := RunMigrations(logger, "invalid", "postgres://localhost")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create migrate instance")
	})

	t.Run("invalid-connection-string", func(t *testing.T) {
		err := RunMigrations(logger, "postgres", "invalid-connection-string")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create migrate instance")
	})
}

func TestRunMigrationsDown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("invalid-driver", func(t *testing.T) {
		err := RunMigrationsDown(logger, "invalid", "postgres://localhost", 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create migrate instance")
	})

	t.Run("invalid-connection-string", func(t *testing.T) {
		err := RunMigrationsDown(logger, "postgres", "invalid-connection-string", 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create migrate instance")
	})

	t.Run("zero-steps", func(t *testing.T) {
		// Zero steps should still attempt to create the migrate instance and return ErrNoChange
		err := RunMigrationsDown(logger, "postgres", "postgres://localhost/testdb", 0)
		require.Error(t, err)
		// Will fail at migrate instance creation since we don't have a real DB, which is expected
		require.Contains(t, err.Error(), "failed to create migrate instance")
	})
}
