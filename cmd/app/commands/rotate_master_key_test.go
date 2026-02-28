package commands

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

type mockKMSKeeper struct {
	mock.Mock
}

func (m *mockKMSKeeper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	args := m.Called(ctx, ciphertext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockKMSKeeper) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	args := m.Called(ctx, plaintext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockKMSKeeper) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockKMSService struct {
	mock.Mock
}

func (m *mockKMSService) OpenKeeper(ctx context.Context, keyURI string) (cryptoDomain.KMSKeeper, error) {
	args := m.Called(ctx, keyURI)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(cryptoDomain.KMSKeeper), args.Error(1)
}

func TestRunRotateMasterKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	kmsProvider := "localsecrets"
	kmsKeyURI := "base64key://YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="
	existingMasterKeys := "old-key:YWJjZGVmZ2hpamtsbW5vcA=="
	existingActiveKeyID := "old-key"

	t.Run("success", func(t *testing.T) {
		mockKMSService := &mockKMSService{}
		mockKeeper := &mockKMSKeeper{}

		mockKMSService.On("OpenKeeper", ctx, kmsKeyURI).Return(mockKeeper, nil)
		mockKeeper.On("Encrypt", ctx, mock.AnythingOfType("[]uint8")).Return([]byte("encrypted-key"), nil)
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
		require.Contains(t, out.String(), "KMS_PROVIDER=\"localsecrets\"")
		require.Contains(
			t,
			out.String(),
			"MASTER_KEYS=\"old-key:YWJjZGVmZ2hpamtsbW5vcA==,new-key:ZW5jcnlwdGVkLWtleQ==\"",
		)
		require.Contains(t, out.String(), "ACTIVE_MASTER_KEY_ID=\"new-key\"")

		mockKMSService.AssertExpectations(t)
		mockKeeper.AssertExpectations(t)
	})

	t.Run("kms-open-error", func(t *testing.T) {
		mockKMSService := &mockKMSService{}
		mockKMSService.On("OpenKeeper", ctx, kmsKeyURI).Return(nil, errors.New("kms error"))

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

		require.Error(t, err)
		require.Contains(t, err.Error(), "kms error")
	})

	t.Run("missing-kms-params", func(t *testing.T) {
		mockKMSService := &mockKMSService{}
		err := RunRotateMasterKey(ctx, mockKMSService, logger, &bytes.Buffer{}, "new-key", "", "", "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "KMS_PROVIDER and KMS_KEY_URI are required")
	})
}
