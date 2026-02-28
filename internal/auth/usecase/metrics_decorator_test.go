package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/auth/usecase"
	usecaseMocks "github.com/allisson/secrets/internal/auth/usecase/mocks"
)

// mockBusinessMetrics is a local mock for metrics.BusinessMetrics to avoid dependency issues.
type mockBusinessMetrics struct {
	mock.Mock
}

func (m *mockBusinessMetrics) RecordOperation(ctx context.Context, domain, operation, status string) {
	m.Called(ctx, domain, operation, status)
}

func (m *mockBusinessMetrics) RecordDuration(
	ctx context.Context,
	domain, operation string,
	duration time.Duration,
	status string,
) {
	m.Called(ctx, domain, operation, duration, status)
}

func TestClientUseCaseWithMetrics(t *testing.T) {
	mockNext := &usecaseMocks.MockClientUseCase{}
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewClientUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	clientID := uuid.New()

	t.Run("Create success", func(t *testing.T) {
		input := &authDomain.CreateClientInput{Name: "test"}
		output := &authDomain.CreateClientOutput{ID: clientID}

		mockNext.On("Create", ctx, input).Return(output, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "auth", "client_create", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "auth", "client_create", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		res, err := uc.Create(ctx, input)
		assert.NoError(t, err)
		assert.Equal(t, output, res)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("Create error", func(t *testing.T) {
		input := &authDomain.CreateClientInput{Name: "test"}
		expectedErr := errors.New("error")

		mockNext.On("Create", ctx, input).Return(nil, expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "auth", "client_create", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "auth", "client_create", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		res, err := uc.Create(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, res)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestTokenUseCaseWithMetrics(t *testing.T) {
	mockNext := &usecaseMocks.MockTokenUseCase{}
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTokenUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()

	t.Run("Issue success", func(t *testing.T) {
		input := &authDomain.IssueTokenInput{ClientID: uuid.New()}
		output := &authDomain.IssueTokenOutput{PlainToken: "token"}

		mockNext.On("Issue", ctx, input).Return(output, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "auth", "token_issue", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "auth", "token_issue", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		res, err := uc.Issue(ctx, input)
		assert.NoError(t, err)
		assert.Equal(t, output, res)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestAuditLogUseCaseWithMetrics(t *testing.T) {
	mockNext := &usecaseMocks.MockAuditLogUseCase{}
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewAuditLogUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()

	t.Run("Create success", func(t *testing.T) {
		requestID := uuid.New()
		clientID := uuid.New()
		mockNext.On("Create", ctx, requestID, clientID, authDomain.ReadCapability, "/test", mock.Anything).
			Return(nil).
			Once()
		mockMetrics.On("RecordOperation", ctx, "auth", "audit_log_create", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "auth", "audit_log_create", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		err := uc.Create(ctx, requestID, clientID, authDomain.ReadCapability, "/test", nil)
		assert.NoError(t, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}
