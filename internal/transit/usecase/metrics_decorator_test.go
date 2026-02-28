package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	"github.com/allisson/secrets/internal/transit/usecase"
	usecaseMocks "github.com/allisson/secrets/internal/transit/usecase/mocks"
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

func TestTransitKeyUseCaseWithMetrics_Create(t *testing.T) {
	mockNext := usecaseMocks.NewMockTransitKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTransitKeyUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	name := "test-key"
	alg := cryptoDomain.AESGCM

	t.Run("Create_Success", func(t *testing.T) {
		// Arrange
		expectedKey := &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   1,
			DekID:     uuid.Must(uuid.NewV7()),
			CreatedAt: time.Now().UTC(),
		}

		mockNext.EXPECT().Create(ctx, name, alg).Return(expectedKey, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_create", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_create", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Act
		result, err := uc.Create(ctx, name, alg)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedKey, result)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("Create_Error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("create failed")

		mockNext.EXPECT().Create(ctx, name, alg).Return(nil, expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_create", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_create", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Act
		result, err := uc.Create(ctx, name, alg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestTransitKeyUseCaseWithMetrics_Rotate(t *testing.T) {
	mockNext := usecaseMocks.NewMockTransitKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTransitKeyUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	name := "test-key"
	alg := cryptoDomain.ChaCha20

	t.Run("Rotate_Success", func(t *testing.T) {
		// Arrange
		expectedKey := &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   2,
			DekID:     uuid.Must(uuid.NewV7()),
			CreatedAt: time.Now().UTC(),
		}

		mockNext.EXPECT().Rotate(ctx, name, alg).Return(expectedKey, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_rotate", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_rotate", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Act
		result, err := uc.Rotate(ctx, name, alg)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedKey, result)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("Rotate_Error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("rotation failed")

		mockNext.EXPECT().Rotate(ctx, name, alg).Return(nil, expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_rotate", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_rotate", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Act
		result, err := uc.Rotate(ctx, name, alg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestTransitKeyUseCaseWithMetrics_Delete(t *testing.T) {
	mockNext := usecaseMocks.NewMockTransitKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTransitKeyUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	transitKeyID := uuid.Must(uuid.NewV7())

	t.Run("Delete_Success", func(t *testing.T) {
		// Arrange
		mockNext.EXPECT().Delete(ctx, transitKeyID).Return(nil).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_delete", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_delete", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Act
		err := uc.Delete(ctx, transitKeyID)

		// Assert
		assert.NoError(t, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("Delete_Error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("deletion failed")

		mockNext.EXPECT().Delete(ctx, transitKeyID).Return(expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_delete", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_delete", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Act
		err := uc.Delete(ctx, transitKeyID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestTransitKeyUseCaseWithMetrics_Encrypt(t *testing.T) {
	mockNext := usecaseMocks.NewMockTransitKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTransitKeyUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	name := "test-key"
	plaintext := []byte("secret data")

	t.Run("Encrypt_Success", func(t *testing.T) {
		// Arrange
		expectedBlob := &transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: []byte("encrypted data"),
		}

		mockNext.EXPECT().Encrypt(ctx, name, plaintext).Return(expectedBlob, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_encrypt", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_encrypt", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Act
		result, err := uc.Encrypt(ctx, name, plaintext)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedBlob, result)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("Encrypt_Error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("encryption failed")

		mockNext.EXPECT().Encrypt(ctx, name, plaintext).Return(nil, expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_encrypt", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_encrypt", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Act
		result, err := uc.Encrypt(ctx, name, plaintext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestTransitKeyUseCaseWithMetrics_Decrypt(t *testing.T) {
	mockNext := usecaseMocks.NewMockTransitKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTransitKeyUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	name := "test-key"
	ciphertext := "1:ZW5jcnlwdGVkIGRhdGE="

	t.Run("Decrypt_Success", func(t *testing.T) {
		// Arrange
		expectedBlob := &transitDomain.EncryptedBlob{
			Version:   1,
			Plaintext: []byte("secret data"),
		}

		mockNext.EXPECT().Decrypt(ctx, name, ciphertext).Return(expectedBlob, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_decrypt", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_decrypt", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Act
		result, err := uc.Decrypt(ctx, name, ciphertext)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedBlob, result)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("Decrypt_Error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("decryption failed")

		mockNext.EXPECT().Decrypt(ctx, name, ciphertext).Return(nil, expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_decrypt", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_decrypt", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Act
		result, err := uc.Decrypt(ctx, name, ciphertext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}

func TestTransitKeyUseCaseWithMetrics_List(t *testing.T) {
	mockNext := usecaseMocks.NewMockTransitKeyUseCase(t)
	mockMetrics := &mockBusinessMetrics{}
	uc := usecase.NewTransitKeyUseCaseWithMetrics(mockNext, mockMetrics)

	ctx := context.Background()
	offset := 0
	limit := 50

	t.Run("List_Success", func(t *testing.T) {
		// Arrange
		expectedKeys := []*transitDomain.TransitKey{
			{
				ID:        uuid.Must(uuid.NewV7()),
				Name:      "key-1",
				Version:   1,
				DekID:     uuid.Must(uuid.NewV7()),
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        uuid.Must(uuid.NewV7()),
				Name:      "key-2",
				Version:   1,
				DekID:     uuid.Must(uuid.NewV7()),
				CreatedAt: time.Now().UTC(),
			},
		}

		mockNext.EXPECT().List(ctx, offset, limit).Return(expectedKeys, nil).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_list", "success").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_list", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Act
		result, err := uc.List(ctx, offset, limit)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedKeys, result)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("List_Error", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("list failed")

		mockNext.EXPECT().List(ctx, offset, limit).Return(nil, expectedErr).Once()
		mockMetrics.On("RecordOperation", ctx, "transit", "transit_key_list", "error").Return().Once()
		mockMetrics.On("RecordDuration", ctx, "transit", "transit_key_list", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Act
		result, err := uc.List(ctx, offset, limit)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockNext.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})
}
