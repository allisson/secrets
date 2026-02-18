package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationMocks "github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

func TestNewTokenizationKeyUseCaseWithMetrics(t *testing.T) {
	mockUseCase := tokenizationMocks.NewMockTokenizationKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}

	decorator := NewTokenizationKeyUseCaseWithMetrics(mockUseCase, mockMetrics)

	assert.NotNil(t, decorator)
	assert.IsType(t, &tokenizationKeyUseCaseWithMetrics{}, decorator)
}

func TestTokenizationKeyUseCaseWithMetrics_Create(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*tokenizationMocks.MockTokenizationKeyUseCase, *mockBusinessMetrics)
		keyName         string
		formatType      tokenizationDomain.FormatType
		isDeterministic bool
		algorithm       cryptoDomain.Algorithm
		expectedKey     *tokenizationDomain.TokenizationKey
		expectedErr     error
		expectedStatus  string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				key := &tokenizationDomain.TokenizationKey{
					ID:              uuid.New(),
					Name:            "test-key",
					Version:         1,
					FormatType:      tokenizationDomain.FormatUUID,
					IsDeterministic: false,
					CreatedAt:       time.Now().UTC(),
				}
				mockUseCase.EXPECT().
					Create(mock.Anything, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
					Return(key, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_create", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_create", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			keyName:         "test-key",
			formatType:      tokenizationDomain.FormatUUID,
			isDeterministic: false,
			algorithm:       cryptoDomain.AESGCM,
			expectedKey: &tokenizationDomain.TokenizationKey{
				ID:              uuid.UUID{},
				Name:            "test-key",
				Version:         1,
				FormatType:      tokenizationDomain.FormatUUID,
				IsDeterministic: false,
			},
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Success_DeterministicKey_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				key := &tokenizationDomain.TokenizationKey{
					ID:              uuid.New(),
					Name:            "deterministic-key",
					Version:         1,
					FormatType:      tokenizationDomain.FormatAlphanumeric,
					IsDeterministic: true,
					CreatedAt:       time.Now().UTC(),
				}
				mockUseCase.EXPECT().
					Create(mock.Anything, "deterministic-key", tokenizationDomain.FormatAlphanumeric, true, cryptoDomain.ChaCha20).
					Return(key, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_create", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_create", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			keyName:         "deterministic-key",
			formatType:      tokenizationDomain.FormatAlphanumeric,
			isDeterministic: true,
			algorithm:       cryptoDomain.ChaCha20,
			expectedKey: &tokenizationDomain.TokenizationKey{
				ID:              uuid.UUID{},
				Name:            "deterministic-key",
				Version:         1,
				FormatType:      tokenizationDomain.FormatAlphanumeric,
				IsDeterministic: true,
			},
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Create(mock.Anything, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
					Return(nil, errors.New("key creation failed")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_create", "error").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_create", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			keyName:         "test-key",
			formatType:      tokenizationDomain.FormatUUID,
			isDeterministic: false,
			algorithm:       cryptoDomain.AESGCM,
			expectedKey:     nil,
			expectedErr:     errors.New("key creation failed"),
			expectedStatus:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationKeyUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationKeyUseCaseWithMetrics(mockUseCase, mockMetrics)

			key, err := decorator.Create(
				context.Background(),
				tt.keyName,
				tt.formatType,
				tt.isDeterministic,
				tt.algorithm,
			)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
				assert.Equal(t, tt.expectedKey.Name, key.Name)
				assert.Equal(t, tt.expectedKey.FormatType, key.FormatType)
				assert.Equal(t, tt.expectedKey.IsDeterministic, key.IsDeterministic)
			}

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestTokenizationKeyUseCaseWithMetrics_Rotate(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*tokenizationMocks.MockTokenizationKeyUseCase, *mockBusinessMetrics)
		keyName         string
		formatType      tokenizationDomain.FormatType
		isDeterministic bool
		algorithm       cryptoDomain.Algorithm
		expectedKey     *tokenizationDomain.TokenizationKey
		expectedErr     error
		expectedStatus  string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				key := &tokenizationDomain.TokenizationKey{
					ID:              uuid.New(),
					Name:            "test-key",
					Version:         2,
					FormatType:      tokenizationDomain.FormatUUID,
					IsDeterministic: false,
					CreatedAt:       time.Now().UTC(),
				}
				mockUseCase.EXPECT().
					Rotate(mock.Anything, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
					Return(key, nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_rotate", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_rotate", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			keyName:         "test-key",
			formatType:      tokenizationDomain.FormatUUID,
			isDeterministic: false,
			algorithm:       cryptoDomain.AESGCM,
			expectedKey: &tokenizationDomain.TokenizationKey{
				ID:              uuid.UUID{},
				Name:            "test-key",
				Version:         2,
				FormatType:      tokenizationDomain.FormatUUID,
				IsDeterministic: false,
			},
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				mockUseCase.EXPECT().
					Rotate(mock.Anything, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
					Return(nil, errors.New("rotation failed")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_rotate", "error").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_rotate", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			keyName:         "test-key",
			formatType:      tokenizationDomain.FormatUUID,
			isDeterministic: false,
			algorithm:       cryptoDomain.AESGCM,
			expectedKey:     nil,
			expectedErr:     errors.New("rotation failed"),
			expectedStatus:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationKeyUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			tt.setupMocks(mockUseCase, mockMetrics)

			decorator := NewTokenizationKeyUseCaseWithMetrics(mockUseCase, mockMetrics)

			key, err := decorator.Rotate(
				context.Background(),
				tt.keyName,
				tt.formatType,
				tt.isDeterministic,
				tt.algorithm,
			)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
				assert.Equal(t, tt.expectedKey.Name, key.Name)
				assert.Equal(t, tt.expectedKey.Version, key.Version)
			}

			mockMetrics.AssertExpectations(t)
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestTokenizationKeyUseCaseWithMetrics_Delete(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*tokenizationMocks.MockTokenizationKeyUseCase, *mockBusinessMetrics)
		keyID          uuid.UUID
		expectedErr    error
		expectedStatus string
	}{
		{
			name: "Success_RecordsSuccessMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				keyID := uuid.New()
				mockUseCase.EXPECT().
					Delete(mock.Anything, keyID).
					Return(nil).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_delete", "success").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_delete", mock.AnythingOfType("time.Duration"), "success").
					Once()
			},
			keyID:          uuid.New(),
			expectedErr:    nil,
			expectedStatus: "success",
		},
		{
			name: "Error_RecordsErrorMetrics",
			setupMocks: func(mockUseCase *tokenizationMocks.MockTokenizationKeyUseCase, mockMetrics *mockBusinessMetrics) {
				keyID := uuid.New()
				mockUseCase.EXPECT().
					Delete(mock.Anything, keyID).
					Return(errors.New("key not found")).
					Once()
				mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_delete", "error").
					Once()
				mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_delete", mock.AnythingOfType("time.Duration"), "error").
					Once()
			},
			keyID:          uuid.New(),
			expectedErr:    errors.New("key not found"),
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUseCase := tokenizationMocks.NewMockTokenizationKeyUseCase(t)
			mockMetrics := &mockBusinessMetrics{}
			// Generate a fresh UUID for each test to pass to mock setup
			testKeyID := uuid.New()
			mockUseCase.EXPECT().
				Delete(mock.Anything, testKeyID).
				Return(tt.expectedErr).
				Once()
			mockMetrics.On("RecordOperation", mock.Anything, "tokenization", "tokenization_key_delete", tt.expectedStatus).
				Once()
			mockMetrics.On("RecordDuration", mock.Anything, "tokenization", "tokenization_key_delete", mock.AnythingOfType("time.Duration"), tt.expectedStatus).
				Once()

			decorator := NewTokenizationKeyUseCaseWithMetrics(mockUseCase, mockMetrics)

			err := decorator.Delete(context.Background(), testKeyID)

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
