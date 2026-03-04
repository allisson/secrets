package commands

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	transitMocks "github.com/allisson/secrets/internal/transit/usecase/mocks"
)

func TestRunPurgeTransitKeys(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	days := 30

	t.Run("text-output", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		mockUseCase.On("PurgeDeleted", ctx, days, false).Return(int64(100), nil)

		var out bytes.Buffer
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &out, days, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 100 transit key(s) older than 30 day(s)")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("text-output-dry-run", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		mockUseCase.On("PurgeDeleted", ctx, days, true).Return(int64(75), nil)

		var out bytes.Buffer
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &out, days, true, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Dry-run mode: Would delete 75 transit key(s) older than 30 day(s)")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("json-output", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		mockUseCase.On("PurgeDeleted", ctx, days, true).Return(int64(50), nil)

		var out bytes.Buffer
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &out, days, true, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"count": 50`)
		require.Contains(t, out.String(), `"days": 30`)
		require.Contains(t, out.String(), `"dry_run": true`)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("json-output-no-dry-run", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		mockUseCase.On("PurgeDeleted", ctx, days, false).Return(int64(25), nil)

		var out bytes.Buffer
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &out, days, false, "json")

		require.NoError(t, err)
		require.Contains(t, out.String(), `"count": 25`)
		require.Contains(t, out.String(), `"days": 30`)
		require.Contains(t, out.String(), `"dry_run": false`)
		mockUseCase.AssertExpectations(t)
	})

	t.Run("invalid-days-negative", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &bytes.Buffer{}, -1, false, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "days must be a positive number")
	})

	t.Run("zero-days-allowed", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		mockUseCase.On("PurgeDeleted", ctx, 0, false).Return(int64(10), nil)

		var out bytes.Buffer
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &out, 0, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 10 transit key(s) older than 0 day(s)")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("no-keys-to-delete", func(t *testing.T) {
		mockUseCase := &transitMocks.MockTransitKeyUseCase{}
		mockUseCase.On("PurgeDeleted", ctx, days, false).Return(int64(0), nil)

		var out bytes.Buffer
		err := RunPurgeTransitKeys(ctx, mockUseCase, logger, &out, days, false, "text")

		require.NoError(t, err)
		require.Contains(t, out.String(), "Successfully deleted 0 transit key(s)")
		mockUseCase.AssertExpectations(t)
	})
}
