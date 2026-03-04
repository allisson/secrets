package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunServer_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Call RunServer which should fail (e.g. database connection refused)
	// rather than blocking indefinitely in unit tests.
	err := RunServer(ctx, "v1.0.0")
	require.Error(t, err)
}
