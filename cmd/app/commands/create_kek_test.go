package commands

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoMocks "github.com/allisson/secrets/internal/crypto/usecase/mocks"
)

func TestRunCreateKek(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	masterKeyChain := cryptoDomain.NewMasterKeyChain("test-master-key")

	t.Run("success", func(t *testing.T) {
		mockUseCase := &cryptoMocks.MockKekUseCase{}
		mockUseCase.On("Create", ctx, masterKeyChain, cryptoDomain.AESGCM).Return(nil)

		err := RunCreateKek(ctx, mockUseCase, masterKeyChain, logger, "aes-gcm")
		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-algorithm", func(t *testing.T) {
		mockUseCase := &cryptoMocks.MockKekUseCase{}
		err := RunCreateKek(ctx, mockUseCase, masterKeyChain, logger, "invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid algorithm")
	})
}
