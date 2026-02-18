package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationMocks "github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

// mockBusinessMetrics is a mock implementation of metrics.BusinessMetrics for testing.
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

func TestNewTokenizationUseCaseWithMetrics(t *testing.T) {
	mockUseCase := tokenizationMocks.NewMockTokenizationUseCase(t)
	mockMetrics := &mockBusinessMetrics{}

	decorator := NewTokenizationUseCaseWithMetrics(mockUseCase, mockMetrics)

	assert.NotNil(t, decorator)
	assert.IsType(t, &tokenizationUseCaseWithMetrics{}, decorator)
}

func TestTokenizationUseCaseWithMetrics_Tokenize(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*tokenizationMocks.MockTokenizationUseCase, *mockBusinessMetrics)
		keyName        string
		plaintext      []byte
		metadata       map[string]any
		expiresAt      *time.Time
		expectedToken  *tokenizationDomain.Token
		expectedErr    error
		expectedStatus string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				hash := "hash"
				token := &tokenizationDomain.Token{
					ID:        uuid.New(),
					Token:     "test-token",
					ValueHash: &hash,
				}
				mockUseCase.EXPECT().
					Tokenize(mock.Anything, "test-key", []byte("plaintext"), mock.Anything, mock.Anything).
					Return(token, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenize", "success").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenize", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			keyName:   "test-key",
			plaintext: []byte("plaintext"),
			metadata:  map[string]any{"key": "value"},
			expiresAt: nil,
			expectedToken: &tokenizationDomain.Token{
				ID:        uuid.UUID{},
				Token:     "test-token",
				ValueHash: nil,
			},
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Tokenize(mock.Anything, "test-key", []byte("plaintext"), mock.Anything, mock.Anything).
					Return(nil, errors.New("tokenization failed")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenize", "error").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenize", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			keyName:        "test-key",
			plaintext:      []byte("plaintext"),
			metadata:       map[string]any{"key": "value"},
			expiresAt:      nil,
			expectedToken:  nil,
			expectedErr:    errors.New("tokenization failed"),
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationUseCaseWithMetrics(mockUseCase, mockMetrics)

			token, err := decorator.Tokenize(
				context.Background(),
				tt.keyName,
				tt.plaintext,
				tt.metadata,
				tt.expiresAt,
			)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
			}

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestTokenizationUseCaseWithMetrics_Detokenize(t *testing.T) {
	tests := []struct {
		name              string
		setupMocks        func(*tokenizationMocks.MockTokenizationUseCase, *mockBusinessMetrics)
		token             string
		expectedPlaintext []byte
		expectedMetadata  map[string]any
		expectedErr       error
		expectedStatus    string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Detokenize(mock.Anything, "test-token").
					Return([]byte("plaintext"), map[string]any{"key": "value"}, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "detokenize", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "detokenize", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			token:             "test-token",
			expectedPlaintext: []byte("plaintext"),
			expectedMetadata:  map[string]any{"key": "value"},
			expectedErr:       nil,
			expectedStatus:    "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Detokenize(mock.Anything, "invalid-token").
					Return(nil, nil, errors.New("token not found")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "detokenize", "error").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "detokenize", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			token:             "invalid-token",
			expectedPlaintext: nil,
			expectedMetadata:  nil,
			expectedErr:       errors.New("token not found"),
			expectedStatus:    "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationUseCaseWithMetrics(mockUseCase, mockMetrics)

			plaintext, metadata, err := decorator.Detokenize(context.Background(), tt.token)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Nil(t, plaintext)
				assert.Nil(t, metadata)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPlaintext, plaintext)
				assert.Equal(t, tt.expectedMetadata, metadata)
			}

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestTokenizationUseCaseWithMetrics_Validate(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*tokenizationMocks.MockTokenizationUseCase, *mockBusinessMetrics)
		token          string
		expectedValid  bool
		expectedErr    error
		expectedStatus string
	}{
		{
			name: "Success_ValidToken_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Validate(mock.Anything, "valid-token").
					Return(true, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "validate", "success").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "validate", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			token:          "valid-token",
			expectedValid:  true,
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Success_InvalidToken_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Validate(mock.Anything, "invalid-token").
					Return(false, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "validate", "success").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "validate", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			token:          "invalid-token",
			expectedValid:  false,
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Validate(mock.Anything, "error-token").
					Return(false, errors.New("validation error")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "validate", "error").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "validate", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			token:          "error-token",
			expectedValid:  false,
			expectedErr:    errors.New("validation error"),
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationUseCaseWithMetrics(mockUseCase, mockMetrics)

			valid, err := decorator.Validate(context.Background(), tt.token)

			if tt.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedValid, valid)

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestTokenizationUseCaseWithMetrics_Revoke(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*tokenizationMocks.MockTokenizationUseCase, *mockBusinessMetrics)
		token          string
		expectedErr    error
		expectedStatus string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Revoke(mock.Anything, "test-token").
					Return(nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "revoke", "success").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "revoke", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			token:          "test-token",
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Revoke(mock.Anything, "invalid-token").
					Return(errors.New("token not found")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "revoke", "error").Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "revoke", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			token:          "invalid-token",
			expectedErr:    errors.New("token not found"),
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationUseCaseWithMetrics(mockUseCase, mockMetrics)

			err := decorator.Revoke(context.Background(), tt.token)

			if tt.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestTokenizationUseCaseWithMetrics_CleanupExpired(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*tokenizationMocks.MockTokenizationUseCase, *mockBusinessMetrics)
		days           int
		dryRun         bool
		expectedCount  int64
		expectedErr    error
		expectedStatus string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					CleanupExpired(mock.Anything, 30, false).
					Return(int64(10), nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "cleanup_expired", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "cleanup_expired", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			days:           30,
			dryRun:         false,
			expectedCount:  10,
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Success_DryRun_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					CleanupExpired(mock.Anything, 7, true).
					Return(int64(5), nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "cleanup_expired", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "cleanup_expired", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			days:           7,
			dryRun:         true,
			expectedCount:  5,
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					CleanupExpired(mock.Anything, 30, false).
					Return(int64(0), errors.New("cleanup failed")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "cleanup_expired", "error").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "cleanup_expired", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			days:           30,
			dryRun:         false,
			expectedCount:  0,
			expectedErr:    errors.New("cleanup failed"),
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationUseCaseWithMetrics(mockUseCase, mockMetrics)

			count, err := decorator.CleanupExpired(context.Background(), tt.days, tt.dryRun)

			if tt.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCount, count)

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}
