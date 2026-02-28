package commands

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoMocks "github.com/allisson/secrets/internal/crypto/usecase/mocks"
)

func TestRunRewrapDeks(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	masterKeyChain := cryptoDomain.NewMasterKeyChain("test-master-key")
	kekID := uuid.New()
	kekIDStr := kekID.String()

	t.Run("success", func(t *testing.T) {
		mockKekUseCase := &cryptoMocks.MockKekUseCase{}
		mockDekUseCase := &cryptoMocks.MockDekUseCase{}
		kekChain := cryptoDomain.NewKekChain(nil)

		mockKekUseCase.On("Unwrap", ctx, masterKeyChain).Return(kekChain, nil)
		mockDekUseCase.On("Rewrap", ctx, kekChain, kekID, 100).Return(10, nil).Once()
		mockDekUseCase.On("Rewrap", ctx, kekChain, kekID, 100).Return(0, nil).Once()

		err := RunRewrapDeks(ctx, masterKeyChain, mockKekUseCase, mockDekUseCase, logger, kekIDStr, 100)
		require.NoError(t, err)

		mockKekUseCase.AssertExpectations(t)
		mockDekUseCase.AssertExpectations(t)
	})

	t.Run("invalid-kek-id", func(t *testing.T) {
		err := RunRewrapDeks(ctx, masterKeyChain, nil, nil, logger, "invalid", 100)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid kek-id")
	})
}
