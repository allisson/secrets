package commands

import (
	"bytes"
	"context"
	"encoding/json"
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
	startDate := "2025-01-01"
	endDate := "2025-01-02"

	report := &authUseCase.VerificationReport{
		TotalChecked: 10,
		SignedCount:  10,
		ValidCount:   10,
	}

	t.Run("success-text", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		mockUseCase.On("VerifyBatch", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return(report, nil)

		var out bytes.Buffer
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &out, startDate, endDate, "text")
		require.NoError(t, err)
		require.Contains(t, out.String(), "Audit Log Integrity Verification")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("success-json", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		mockUseCase.On("VerifyBatch", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return(report, nil)

		var out bytes.Buffer
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &out, startDate, endDate, "json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(out.Bytes(), &result)
		require.NoError(t, err)
		require.Equal(t, float64(10), result["total_checked"])
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-dates", func(t *testing.T) {
		err := RunVerifyAuditLogs(ctx, nil, logger, nil, "invalid", endDate, "text")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid start date")
	})

	t.Run("integrity-failure", func(t *testing.T) {
		mockUseCase := &authMocks.MockAuditLogUseCase{}
		failureReport := &authUseCase.VerificationReport{
			TotalChecked: 10,
			InvalidCount: 2,
			InvalidLogs:  []uuid.UUID{uuid.New(), uuid.New()},
		}
		mockUseCase.On("VerifyBatch", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return(failureReport, nil)

		var out bytes.Buffer
		err := RunVerifyAuditLogs(ctx, mockUseCase, logger, &out, startDate, endDate, "text")
		require.Error(t, err)
		require.Contains(t, err.Error(), "integrity check failed")
		require.Contains(t, out.String(), "WARNING: 2 log(s) failed integrity check!")
	})
}
