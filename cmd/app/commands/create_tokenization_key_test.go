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

func TestRunCreateTokenizationKey(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("success", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationKeyUseCase{}
		expectedKey := &tokenizationDomain.TokenizationKey{
			ID:              uuid.New(),
			Name:            "test-token",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: true,
			Version:         1,
		}
		mockUseCase.On("Create", ctx, "test-token", tokenizationDomain.FormatUUID, true, cryptoDomain.AESGCM).
			Return(expectedKey, nil)

		err := RunCreateTokenizationKey(ctx, mockUseCase, logger, "test-token", "uuid", true, "aes-gcm")
		require.NoError(t, err)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-format", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationKeyUseCase{}
		err := RunCreateTokenizationKey(ctx, mockUseCase, logger, "test", "invalid", true, "aes-gcm")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid format type")
	})
}
