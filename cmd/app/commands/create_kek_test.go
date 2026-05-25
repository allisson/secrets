package commands

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/keyring"
	keyringMocks "github.com/allisson/secrets/internal/keyring/mocks"
)

func TestRunCreateKek(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	masterKeyChain := keyring.NewMasterKeyChain("key1")

	t.Run("success", func(t *testing.T) {
		mockUseCase := &keyringMocks.MockKekUseCase{}
		mockUseCase.On("Create", ctx, masterKeyChain, keyring.AESGCM).Return(nil)

		err := RunCreateKek(ctx, mockUseCase, masterKeyChain, logger, "aes-gcm")

		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-algorithm", func(t *testing.T) {
		mockUseCase := &keyringMocks.MockKekUseCase{}
		err := RunCreateKek(ctx, mockUseCase, masterKeyChain, logger, "invalid")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid algorithm")
	})
}
