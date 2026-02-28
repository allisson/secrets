package commands

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoMocks "github.com/allisson/secrets/internal/crypto/usecase/mocks"
)

func TestRunRotateKek(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	masterKeyChain := cryptoDomain.NewMasterKeyChain("key1")

	t.Run("success-aes-gcm", func(t *testing.T) {
		mockUseCase := &cryptoMocks.MockKekUseCase{}
		mockUseCase.On("Rotate", ctx, masterKeyChain, cryptoDomain.AESGCM).Return(nil)

		err := RunRotateKek(ctx, mockUseCase, masterKeyChain, logger, "aes-gcm")

		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("success-chacha20", func(t *testing.T) {
		mockUseCase := &cryptoMocks.MockKekUseCase{}
		mockUseCase.On("Rotate", ctx, masterKeyChain, cryptoDomain.ChaCha20).Return(nil)

		err := RunRotateKek(ctx, mockUseCase, masterKeyChain, logger, "chacha20-poly1305")

		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-algorithm", func(t *testing.T) {
		mockUseCase := &cryptoMocks.MockKekUseCase{}
		err := RunRotateKek(ctx, mockUseCase, masterKeyChain, logger, "invalid")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid algorithm")
	})
}
