package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	authMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func TestRunCleanAuditLogs(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	days := 30

	t.Run("text-output", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		mockUseCase.On("DeleteOlderThan", ctx, days, false).Return(int64(100), nil)

		var out bytes.Buffer
		err := RunCleanAuditLogs(ctx, mockUseCase, logger, &out, days, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 100 audit log(s)")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("json-output", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		mockUseCase.On("DeleteOlderThan", ctx, days, true).Return(int64(50), nil)

		var out bytes.Buffer
		err := RunCleanAuditLogs(ctx, mockUseCase, logger, &out, days, true, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"count": 50`)
		require.Contains(t, out.String(), `"dry_run": true`)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-days", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		err := RunCleanAuditLogs(ctx, mockUseCase, logger, &bytes.Buffer{}, -1, false, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "days must be a positive number")
	})
}
