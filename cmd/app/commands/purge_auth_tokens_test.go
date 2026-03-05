package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func TestRunPurgeAuthTokens(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	days := 30

	t.Run("text-output", func(t *testing.T) {
		mockUseCase := usecaseMocks.NewMockTokenUseCase(t)
		mockUseCase.EXPECT().PurgeExpiredAndRevoked(ctx, days).Return(int64(100), nil).Once()

		var out bytes.Buffer
		err := RunPurgeAuthTokens(ctx, mockUseCase, logger, &out, days, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully purged 100 expired/revoked authentication token(s)")
	})

	t.Run("json-output", func(t *testing.T) {
		mockUseCase := usecaseMocks.NewMockTokenUseCase(t)
		mockUseCase.EXPECT().PurgeExpiredAndRevoked(ctx, days).Return(int64(50), nil).Once()

		var out bytes.Buffer
		err := RunPurgeAuthTokens(ctx, mockUseCase, logger, &out, days, false, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"count": 50`)
		require.Contains(t, out.String(), `"dry_run": false`)
	})

	t.Run("invalid-days", func(t *testing.T) {
		mockUseCase := usecaseMocks.NewMockTokenUseCase(t)
		err := RunPurgeAuthTokens(ctx, mockUseCase, logger, &bytes.Buffer{}, -1, false, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "days must be a non-negative number")
	})
}
