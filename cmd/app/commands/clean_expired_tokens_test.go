package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	tokenizationMocks "github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

func TestRunCleanExpiredTokens(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	days := 30

	t.Run("text-output", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationUseCase{}
		mockUseCase.On("CleanupExpired", ctx, days, false).Return(int64(10), nil)

		var out bytes.Buffer
		err := RunCleanExpiredTokens(ctx, mockUseCase, logger, &out, days, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 10 expired token(s)")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("json-output", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationUseCase{}
		mockUseCase.On("CleanupExpired", ctx, days, true).Return(int64(5), nil)

		var out bytes.Buffer
		err := RunCleanExpiredTokens(ctx, mockUseCase, logger, &out, days, true, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"count": 5`)
		require.Contains(t, out.String(), `"dry_run": true`)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-days", func(t *testing.T) {
		mockUseCase := &tokenizationMocks.MockTokenizationUseCase{}
		err := RunCleanExpiredTokens(ctx, mockUseCase, logger, &bytes.Buffer{}, -1, false, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "days must be a positive number")
	})
}
