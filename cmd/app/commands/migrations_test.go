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
