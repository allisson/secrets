package commands

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationMocks "github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

func TestRunRotateTokenizationKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	name := "test-token-key"

	t.Run("success", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationKeyUseCase{}
		mockUseCase.On("Rotate", ctx, name, tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
			Return(&tokenizationDomain.TokenizationKey{
				ID: uuid.New(),
			}, nil)

		err := RunRotateTokenizationKey(ctx, mockUseCase, logger, name, "uuid", false, "aes-gcm")

		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-format", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationKeyUseCase{}
		err := RunRotateTokenizationKey(ctx, mockUseCase, logger, name, "invalid", false, "aes-gcm")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid format type")
	})
}
