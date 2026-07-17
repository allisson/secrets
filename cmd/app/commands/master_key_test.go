package commands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/keyring"
)

func TestRunCreateMasterKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		var out bytes.Buffer
		err := RunCreateMasterKey(
			ctx,
			keyring.NewFakeKMSService(),
			logger,
			&out,
			"test-key",
			"localsecrets",
			"base64key://...",
		)
		require.NoError(t, err)
		require.Contains(t, out.String(), "MASTER_KEYS=\"test-key:")
	})

	t.Run("missing-parameters", func(t *testing.T) {
		err := RunCreateMasterKey(ctx, nil, logger, nil, "", "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "required")
	})

	t.Run("kms-error", func(t *testing.T) {
		svc := &keyring.FakeKMSService{FailOpen: errors.New("kms error")}
		err := RunCreateMasterKey(
			ctx,
			svc,
			logger,
			&bytes.Buffer{},
			"test-key",
			"localsecrets",
			"invalid",
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to open KMS keeper")
	})
}
