package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRunRotateMasterKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	kmsProvider := "localsecrets"
	kmsKeyURI := "base64key://YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="
	existingMasterKeys := "old-key:ciphertext"
	existingActiveKeyID := "old-key"

	t.Run("success", func(t *testing.T) {
		mockKMSService := &MockKMSService{}
		mockKeeper := &MockKMSKeeper{}

		mockKMSService.On("OpenKeeper", ctx, kmsKeyURI).Return(mockKeeper, nil)
		mockKeeper.On("Encrypt", ctx, mock.Anything).Return([]byte("new-ciphertext"), nil)
		mockKeeper.On("Close").Return(nil)

		var out bytes.Buffer
		err := RunRotateMasterKey(
			ctx,
			mockKMSService,
			logger,
			&out,
			"new-key",
			kmsProvider,
			kmsKeyURI,
			existingMasterKeys,
			existingActiveKeyID,
		)

		require.NoError(t, err)
		require.Contains(t, out.String(), "MASTER_KEYS=\"old-key:ciphertext,new-key:bmV3LWNpcGhlcnRleHQ=\"")
		require.Contains(t, out.String(), "ACTIVE_MASTER_KEY_ID=\"new-key\"")
		mockKMSService.AssertExpectations(t)
		mockKeeper.AssertExpectations(t)
	})

	t.Run("missing-kms-params", func(t *testing.T) {
		mockKMSService := &MockKMSService{}
		err := RunRotateMasterKey(ctx, mockKMSService, logger, &bytes.Buffer{}, "new-key", "", "", "", "")

		require.Error(t, err)
		require.Contains(t, err.Error(), "KMS_PROVIDER and KMS_KEY_URI are required")
	})

	t.Run("missing-existing-keys", func(t *testing.T) {
		mockKMSService := &MockKMSService{}
		err := RunRotateMasterKey(
			ctx,
			mockKMSService,
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
		mockKMSService := &MockKMSService{}
		err := RunRotateMasterKey(
			ctx,
			mockKMSService,
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
