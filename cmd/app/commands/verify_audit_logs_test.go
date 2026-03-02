package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	authMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

func TestRunVerifyAuditLogs(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	startDate := "2023-01-01"
	endDate := "2023-01-02"

	t.Run("text-output-pass", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		mockUseCase.On("VerifyBatch", ctx, mock.Anything, mock.Anything).
			Return(&authUseCase.VerificationReport{
				TotalChecked: 10,
				ValidCount:   10,
			}, nil)

		var out bytes.Buffer
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &out, startDate, endDate, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Status: PASSED")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("text-output-fail", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		invalidLogs := []uuid.UUID{uuid.New(), uuid.New()}
		mockUseCase.On("VerifyBatch", ctx, mock.Anything, mock.Anything).
			Return(&authUseCase.VerificationReport{
				TotalChecked: 10,
				InvalidCount: 2,
				InvalidLogs:  invalidLogs,
			}, nil)

		var out bytes.Buffer
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &out, startDate, endDate, "text")

		require.Error(t, err)
		require.Contains(t, out.String(), "Status: FAILED")
		require.Contains(t, out.String(), invalidLogs[0].String())
		mockUseCase.AssertExpectations(t)
	})

	t.Run("json-output", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		mockUseCase.On("VerifyBatch", ctx, mock.Anything, mock.Anything).
			Return(&authUseCase.VerificationReport{
				TotalChecked: 5,
				ValidCount:   5,
			}, nil)

		var out bytes.Buffer
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &out, startDate, endDate, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"total_checked": 5`)
		require.Contains(t, out.String(), `"passed": true`)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-date", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &bytes.Buffer{}, "invalid", endDate, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid start date")
	})

	t.Run("invalid-range", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &bytes.Buffer{}, endDate, startDate, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "end date must be after start date")
	})
}
