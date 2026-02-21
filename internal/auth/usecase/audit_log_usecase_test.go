package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// mockAuditLogRepository is a mock implementation of AuditLogRepository for testing.
type mockAuditLogRepository struct {
	mock.Mock
}

func (m *mockAuditLogRepository) Create(ctx context.Context, auditLog *authDomain.AuditLog) error {
	args := m.Called(ctx, auditLog)
	return args.Error(0)
}

func (m *mockAuditLogRepository) Get(ctx context.Context, id uuid.UUID) (*authDomain.AuditLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.AuditLog), args.Error(1)
}

func (m *mockAuditLogRepository) List(
	ctx context.Context,
	offset, limit int,
	createdAtFrom, createdAtTo *time.Time,
) ([]*authDomain.AuditLog, error) {
	args := m.Called(ctx, offset, limit, createdAtFrom, createdAtTo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authDomain.AuditLog), args.Error(1)
}

func (m *mockAuditLogRepository) DeleteOlderThan(
	ctx context.Context,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	args := m.Called(ctx, olderThan, dryRun)
	return args.Get(0).(int64), args.Error(1)
}

func TestAuditLogUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateAuditLogWithAllFields", func(t *testing.T) {
		// Setup mocks
		mockRepo := &mockAuditLogRepository{}

		// Test data
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		capability := authDomain.ReadCapability
		path := "/api/v1/secrets/mykey"
		metadata := map[string]any{
			"user_agent": "Mozilla/5.0",
			"ip_address": "192.168.1.100",
			"method":     "GET",
		}

		// Capture the audit log passed to repository
		var capturedAuditLog *authDomain.AuditLog
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).
			Run(func(args mock.Arguments) {
				capturedAuditLog = args.Get(1).(*authDomain.AuditLog)
			}).
			Return(nil).
			Once()

		// Create use case
		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		// Execute
		err := useCase.Create(ctx, requestID, clientID, capability, path, metadata)

		// Assert
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)

		// Verify captured audit log fields
		assert.NotEqual(t, uuid.Nil, capturedAuditLog.ID, "audit log ID should not be nil")
		assert.Equal(t, requestID, capturedAuditLog.RequestID, "request ID should match")
		assert.Equal(t, clientID, capturedAuditLog.ClientID, "client ID should match")
		assert.Equal(t, capability, capturedAuditLog.Capability, "capability should match")
		assert.Equal(t, path, capturedAuditLog.Path, "path should match")
		assert.Equal(t, metadata, capturedAuditLog.Metadata, "metadata should match")
		assert.False(t, capturedAuditLog.CreatedAt.IsZero(), "created_at should be set")
	})

	t.Run("Success_CreateAuditLogWithNilMetadata", func(t *testing.T) {
		// Setup mocks
		mockRepo := &mockAuditLogRepository{}

		// Test data
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		capability := authDomain.WriteCapability
		path := "/api/v1/secrets/mykey"

		// Capture the audit log passed to repository
		var capturedAuditLog *authDomain.AuditLog
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).
			Run(func(args mock.Arguments) {
				capturedAuditLog = args.Get(1).(*authDomain.AuditLog)
			}).
			Return(nil).
			Once()

		// Create use case
		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		// Execute with nil metadata
		err := useCase.Create(ctx, requestID, clientID, capability, path, nil)

		// Assert
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)

		// Verify metadata is nil
		assert.Nil(t, capturedAuditLog.Metadata, "metadata should be nil")
		assert.NotEqual(t, uuid.Nil, capturedAuditLog.ID, "audit log ID should not be nil")
		assert.Equal(t, requestID, capturedAuditLog.RequestID, "request ID should match")
	})

	t.Run("Success_CreateMultipleAuditLogs", func(t *testing.T) {
		// Setup mocks
		mockRepo := &mockAuditLogRepository{}

		// Test data
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		capability := authDomain.DeleteCapability
		path := "/api/v1/secrets/mykey"

		// Capture audit log IDs
		var capturedIDs []uuid.UUID
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).
			Run(func(args mock.Arguments) {
				auditLog := args.Get(1).(*authDomain.AuditLog)
				capturedIDs = append(capturedIDs, auditLog.ID)
			}).
			Return(nil).
			Times(3)

		// Create use case
		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		// Execute multiple times
		for i := 0; i < 3; i++ {
			err := useCase.Create(ctx, requestID, clientID, capability, path, nil)
			assert.NoError(t, err)
		}

		// Assert
		mockRepo.AssertExpectations(t)

		// Verify all IDs are unique and non-nil
		assert.Len(t, capturedIDs, 3, "should have captured 3 audit log IDs")
		for i, id := range capturedIDs {
			assert.NotEqual(t, uuid.Nil, id, "audit log ID %d should not be nil", i)
		}

		// Check uniqueness
		uniqueIDs := make(map[uuid.UUID]bool)
		for _, id := range capturedIDs {
			uniqueIDs[id] = true
		}
		assert.Len(t, uniqueIDs, 3, "all audit log IDs should be unique")
	})

	t.Run("Success_CreateAuditLogsWithDifferentCapabilities", func(t *testing.T) {
		// Setup mocks
		mockRepo := &mockAuditLogRepository{}

		// Test data
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		path := "/api/v1/keys/kek"

		capabilities := []authDomain.Capability{
			authDomain.ReadCapability,
			authDomain.WriteCapability,
			authDomain.DeleteCapability,
			authDomain.EncryptCapability,
			authDomain.DecryptCapability,
			authDomain.RotateCapability,
		}

		// Capture capabilities
		var capturedCapabilities []authDomain.Capability
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).
			Run(func(args mock.Arguments) {
				auditLog := args.Get(1).(*authDomain.AuditLog)
				capturedCapabilities = append(capturedCapabilities, auditLog.Capability)
			}).
			Return(nil).
			Times(len(capabilities))

		// Create use case
		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		// Execute for each capability
		for _, cap := range capabilities {
			err := useCase.Create(ctx, requestID, clientID, cap, path, nil)
			assert.NoError(t, err)
		}

		// Assert
		mockRepo.AssertExpectations(t)

		// Verify all capabilities were captured correctly
		assert.Equal(t, capabilities, capturedCapabilities, "all capabilities should be captured")
	})

	t.Run("Error_RepositoryCreateFailure", func(t *testing.T) {
		// Setup mocks
		mockRepo := &mockAuditLogRepository{}

		// Test data
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		capability := authDomain.ReadCapability
		path := "/api/v1/secrets/mykey"
		metadata := map[string]any{"key": "value"}

		// Setup repository to return error
		repositoryErr := errors.New("database connection failed")
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).
			Return(repositoryErr).
			Once()

		// Create use case
		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		// Execute
		err := useCase.Create(ctx, requestID, clientID, capability, path, metadata)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create audit log", "error should be wrapped")
		assert.Contains(t, err.Error(), "database connection failed", "original error should be included")
		mockRepo.AssertExpectations(t)
	})
}

func TestAuditLogUseCase_DeleteOlderThan(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_DeleteOlderThan", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		days := 90
		dryRun := false
		expectedCount := int64(150)

		// The use case should calculate cutoff date and call repository
		mockRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time"), dryRun).
			Run(func(args mock.Arguments) {
				cutoffDate := args.Get(1).(time.Time)
				// Verify cutoff date is approximately 90 days ago
				now := time.Now().UTC()
				expectedCutoff := now.AddDate(0, 0, -90)
				// Allow 1 second tolerance for test execution time
				timeDiff := cutoffDate.Sub(expectedCutoff)
				assert.True(t, timeDiff >= -1*time.Second && timeDiff <= 1*time.Second,
					"cutoff date should be approximately 90 days ago")
			}).
			Return(expectedCount, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		count, err := useCase.DeleteOlderThan(ctx, days, dryRun)

		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_DeleteOlderThanWithZeroResults", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		days := 30
		dryRun := false
		expectedCount := int64(0)

		mockRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(expectedCount, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		count, err := useCase.DeleteOlderThan(ctx, days, dryRun)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_DryRunMode", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		days := 90
		dryRun := true
		expectedCount := int64(250)

		// In dry-run mode, repository uses COUNT query instead of DELETE
		mockRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(expectedCount, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		count, err := useCase.DeleteOlderThan(ctx, days, dryRun)

		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_DeleteOlderThanWithVariousDays", func(t *testing.T) {
		testCases := []struct {
			name          string
			days          int
			dryRun        bool
			expectedCount int64
		}{
			{"1 day", 1, false, int64(5)},
			{"7 days", 7, false, int64(25)},
			{"30 days", 30, false, int64(120)},
			{"365 days", 365, false, int64(1500)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockRepo := &mockAuditLogRepository{}

				mockRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time"), tc.dryRun).
					Return(tc.expectedCount, nil).
					Once()

				useCase := NewAuditLogUseCase(mockRepo, nil, nil)

				count, err := useCase.DeleteOlderThan(ctx, tc.days, tc.dryRun)

				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, count)
				mockRepo.AssertExpectations(t)
			})
		}
	})

	t.Run("Error_RepositoryFailure", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		days := 60
		dryRun := false
		repositoryErr := errors.New("database connection failed")

		mockRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(int64(0), repositoryErr).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		count, err := useCase.DeleteOlderThan(ctx, days, dryRun)

		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "failed to delete old audit logs")
		assert.Contains(t, err.Error(), "database connection failed")
		mockRepo.AssertExpectations(t)
	})
}

func TestAuditLogUseCase_List(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_ListWithoutFilters", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		expectedAuditLogs := []*authDomain.AuditLog{
			{
				ID:         uuid.Must(uuid.NewV7()),
				RequestID:  uuid.Must(uuid.NewV7()),
				ClientID:   uuid.Must(uuid.NewV7()),
				Capability: authDomain.ReadCapability,
				Path:       "/secrets/test",
				Metadata:   nil,
				CreatedAt:  time.Now().UTC(),
			},
		}

		mockRepo.On("List", ctx, 0, 50, (*time.Time)(nil), (*time.Time)(nil)).
			Return(expectedAuditLogs, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 0, 50, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedAuditLogs, auditLogs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_ListWithCreatedAtFromFilter", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		now := time.Now().UTC()
		createdAtFrom := now.Add(-1 * time.Hour)

		expectedAuditLogs := []*authDomain.AuditLog{
			{
				ID:         uuid.Must(uuid.NewV7()),
				RequestID:  uuid.Must(uuid.NewV7()),
				ClientID:   uuid.Must(uuid.NewV7()),
				Capability: authDomain.WriteCapability,
				Path:       "/secrets/recent",
				Metadata:   nil,
				CreatedAt:  now,
			},
		}

		mockRepo.On("List", ctx, 0, 50, &createdAtFrom, (*time.Time)(nil)).
			Return(expectedAuditLogs, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 0, 50, &createdAtFrom, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedAuditLogs, auditLogs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_ListWithCreatedAtToFilter", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		now := time.Now().UTC()
		createdAtTo := now.Add(-1 * time.Hour)

		expectedAuditLogs := []*authDomain.AuditLog{
			{
				ID:         uuid.Must(uuid.NewV7()),
				RequestID:  uuid.Must(uuid.NewV7()),
				ClientID:   uuid.Must(uuid.NewV7()),
				Capability: authDomain.DeleteCapability,
				Path:       "/secrets/old",
				Metadata:   nil,
				CreatedAt:  now.Add(-2 * time.Hour),
			},
		}

		mockRepo.On("List", ctx, 0, 50, (*time.Time)(nil), &createdAtTo).
			Return(expectedAuditLogs, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 0, 50, nil, &createdAtTo)

		assert.NoError(t, err)
		assert.Equal(t, expectedAuditLogs, auditLogs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_ListWithBothFilters", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		now := time.Now().UTC()
		createdAtFrom := now.Add(-3 * time.Hour)
		createdAtTo := now.Add(-1 * time.Hour)

		expectedAuditLogs := []*authDomain.AuditLog{
			{
				ID:         uuid.Must(uuid.NewV7()),
				RequestID:  uuid.Must(uuid.NewV7()),
				ClientID:   uuid.Must(uuid.NewV7()),
				Capability: authDomain.EncryptCapability,
				Path:       "/secrets/range",
				Metadata:   nil,
				CreatedAt:  now.Add(-2 * time.Hour),
			},
		}

		mockRepo.On("List", ctx, 0, 50, &createdAtFrom, &createdAtTo).
			Return(expectedAuditLogs, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 0, 50, &createdAtFrom, &createdAtTo)

		assert.NoError(t, err)
		assert.Equal(t, expectedAuditLogs, auditLogs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_ListEmptyResult", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		expectedAuditLogs := []*authDomain.AuditLog{}

		mockRepo.On("List", ctx, 0, 50, (*time.Time)(nil), (*time.Time)(nil)).
			Return(expectedAuditLogs, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 0, 50, nil, nil)

		assert.NoError(t, err)
		assert.Len(t, auditLogs, 0)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_ListWithPagination", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		expectedAuditLogs := []*authDomain.AuditLog{
			{
				ID:         uuid.Must(uuid.NewV7()),
				RequestID:  uuid.Must(uuid.NewV7()),
				ClientID:   uuid.Must(uuid.NewV7()),
				Capability: authDomain.ReadCapability,
				Path:       "/secrets/page2",
				Metadata:   nil,
				CreatedAt:  time.Now().UTC(),
			},
		}

		mockRepo.On("List", ctx, 10, 25, (*time.Time)(nil), (*time.Time)(nil)).
			Return(expectedAuditLogs, nil).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 10, 25, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedAuditLogs, auditLogs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error_RepositoryFailure", func(t *testing.T) {
		mockRepo := &mockAuditLogRepository{}

		repositoryErr := errors.New("database connection failed")
		mockRepo.On("List", ctx, 0, 50, (*time.Time)(nil), (*time.Time)(nil)).
			Return(nil, repositoryErr).
			Once()

		useCase := NewAuditLogUseCase(mockRepo, nil, nil)

		auditLogs, err := useCase.List(ctx, 0, 50, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, auditLogs)
		assert.Contains(t, err.Error(), "failed to list audit logs")
		assert.Contains(t, err.Error(), "database connection failed")
		mockRepo.AssertExpectations(t)
	})
}
