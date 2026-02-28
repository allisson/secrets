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

	t.Run("success", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationKeyUseCase{}
		expectedKey := &tokenizationDomain.TokenizationKey{
			ID:              uuid.New(),
			Name:            "test-token",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: true,
			Version:         2,
		}
		mockUseCase.On("Rotate", ctx, "test-token", tokenizationDomain.FormatUUID, true, cryptoDomain.AESGCM).
			Return(expectedKey, nil)

		err := RunRotateTokenizationKey(ctx, mockUseCase, logger, "test-token", "uuid", true, "aes-gcm")
		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})
}
