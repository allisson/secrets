package commands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// Manual mocks for KMS since they might not be generated in all environments
type MockKMSService struct {
	mock.Mock
}

func (m *MockKMSService) OpenKeeper(ctx context.Context, uri string) (cryptoDomain.KMSKeeper, error) {
	args := m.Called(ctx, uri)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(cryptoDomain.KMSKeeper), args.Error(1)
}

type MockKMSKeeper struct {
	mock.Mock
}

func (m *MockKMSKeeper) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	args := m.Called(ctx, plaintext)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKMSKeeper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	args := m.Called(ctx, ciphertext)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKMSKeeper) Close() error {
	return m.Called().Error(0)
}

func TestRunCreateMasterKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		mockService := &MockKMSService{}
		mockKeeper := &MockKMSKeeper{}

		mockService.On("OpenKeeper", ctx, "base64key://...").Return(mockKeeper, nil)
		mockKeeper.On("Encrypt", ctx, mock.AnythingOfType("[]uint8")).Return([]byte("encrypted"), nil)
		mockKeeper.On("Close").Return(nil)

		var out bytes.Buffer
		err := RunCreateMasterKey(
			ctx,
			mockService,
			logger,
			&out,
			"test-key",
			"localsecrets",
			"base64key://...",
		)
		require.NoError(t, err)
		require.Contains(t, out.String(), "MASTER_KEYS=\"test-key:")

		mockService.AssertExpectations(t)
		mockKeeper.AssertExpectations(t)
	})

	t.Run("missing-parameters", func(t *testing.T) {
		err := RunCreateMasterKey(ctx, nil, logger, nil, "", "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "required")
	})

	t.Run("kms-error", func(t *testing.T) {
		mockService := &MockKMSService{}
		mockService.On("OpenKeeper", ctx, "invalid").Return(nil, errors.New("kms error"))

		err := RunCreateMasterKey(
			ctx,
			mockService,
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
