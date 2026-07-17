package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/keyring"
)

func TestRunRotateMasterKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	kmsProvider := "localsecrets"
	kmsKeyURI := "base64key://YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="
	existingMasterKeys := "old-key:ciphertext"
	existingActiveKeyID := "old-key"

	t.Run("success", func(t *testing.T) {
		var out bytes.Buffer
		err := RunRotateMasterKey(
			ctx,
			keyring.NewFakeKMSService(),
			logger,
			&out,
			"new-key",
			kmsProvider,
			kmsKeyURI,
			existingMasterKeys,
			existingActiveKeyID,
		)

		require.NoError(t, err)
		// The freshly generated master key is random, so the appended ciphertext
		// is non-deterministic; assert the structure preserves the existing keys
		// and appends the new one as active.
		require.Contains(t, out.String(), "MASTER_KEYS=\"old-key:ciphertext,new-key:")
		require.Contains(t, out.String(), "ACTIVE_MASTER_KEY_ID=\"new-key\"")
	})

	t.Run("missing-kms-params", func(t *testing.T) {
		err := RunRotateMasterKey(
			ctx,
			keyring.NewFakeKMSService(),
			logger,
			&bytes.Buffer{},
			"new-key",
			"",
			"",
			"",
			"",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "KMS_PROVIDER and KMS_KEY_URI are required")
	})

	t.Run("missing-existing-keys", func(t *testing.T) {
		err := RunRotateMasterKey(
			ctx,
			keyring.NewFakeKMSService(),
			logger,
			&bytes.Buffer{},
			"new-key",
			kmsProvider,
			kmsKeyURI,
			"",
			"",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "MASTER_KEYS is not set")
	})

	t.Run("invalid-active-key-id", func(t *testing.T) {
		err := RunRotateMasterKey(
			ctx,
			keyring.NewFakeKMSService(),
			logger,
			&bytes.Buffer{},
			"new-key",
			kmsProvider,
			kmsKeyURI,
			existingMasterKeys,
			"invalid-key",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "not found in MASTER_KEYS")
	})
}
