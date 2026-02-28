package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/allisson/secrets/internal/metrics"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	secretsUsecaseMocks "github.com/allisson/secrets/internal/secrets/usecase/mocks"
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

var _ metrics.BusinessMetrics = (*mockBusinessMetrics)(nil)

// TestNewSecretUseCaseWithMetrics tests the metrics decorator constructor.
func TestNewSecretUseCaseWithMetrics(t *testing.T) {
	t.Parallel()

	mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
	mockMetrics := &mockBusinessMetrics{}

	decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)

	assert.NotNil(t, decorator)
	assert.Implements(t, (*SecretUseCase)(nil), decorator)
}

// TestMetricsDecorator_CreateOrUpdate tests the CreateOrUpdate method with metrics.
func TestMetricsDecorator_CreateOrUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_RecordsSuccessMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/api-key"
		value := []byte("secret-value")
		expectedSecret := &secretsDomain.Secret{
			ID:        uuid.Must(uuid.NewV7()),
			Path:      path,
			Version:   1,
			CreatedAt: time.Now().UTC(),
		}

		// Setup expectations
		mockUseCase.EXPECT().
			CreateOrUpdate(ctx, path, value).
			Return(expectedSecret, nil).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_create", "success").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_create", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedSecret, result)
	})

	t.Run("Error_RecordsErrorMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/api-key"
		value := []byte("secret-value")
		expectedError := errors.New("database error")

		// Setup expectations
		mockUseCase.EXPECT().
			CreateOrUpdate(ctx, path, value).
			Return(nil, expectedError).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_create", "error").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_create", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

// TestMetricsDecorator_Get tests the Get method with metrics.
func TestMetricsDecorator_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_RecordsSuccessMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/api-key"
		expectedSecret := &secretsDomain.Secret{
			ID:        uuid.Must(uuid.NewV7()),
			Path:      path,
			Version:   1,
			Plaintext: []byte("decrypted-value"),
			CreatedAt: time.Now().UTC(),
		}

		// Setup expectations
		mockUseCase.EXPECT().
			Get(ctx, path).
			Return(expectedSecret, nil).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_get", "success").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_get", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.Get(ctx, path)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedSecret, result)
	})

	t.Run("Error_RecordsErrorMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/nonexistent"
		expectedError := secretsDomain.ErrSecretNotFound

		// Setup expectations
		mockUseCase.EXPECT().
			Get(ctx, path).
			Return(nil, expectedError).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_get", "error").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_get", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

// TestMetricsDecorator_GetByVersion tests the GetByVersion method with metrics.
func TestMetricsDecorator_GetByVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_RecordsSuccessMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/api-key"
		version := uint(2)
		expectedSecret := &secretsDomain.Secret{
			ID:        uuid.Must(uuid.NewV7()),
			Path:      path,
			Version:   version,
			Plaintext: []byte("decrypted-value"),
			CreatedAt: time.Now().UTC(),
		}

		// Setup expectations
		mockUseCase.EXPECT().
			GetByVersion(ctx, path, version).
			Return(expectedSecret, nil).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_get_version", "success").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_get_version", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.GetByVersion(ctx, path, version)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedSecret, result)
	})

	t.Run("Error_RecordsErrorMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/api-key"
		version := uint(999)
		expectedError := secretsDomain.ErrSecretNotFound

		// Setup expectations
		mockUseCase.EXPECT().
			GetByVersion(ctx, path, version).
			Return(nil, expectedError).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_get_version", "error").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_get_version", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.GetByVersion(ctx, path, version)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

// TestMetricsDecorator_Delete tests the Delete method with metrics.
func TestMetricsDecorator_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_RecordsSuccessMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/api-key"

		// Setup expectations
		mockUseCase.EXPECT().
			Delete(ctx, path).
			Return(nil).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_delete", "success").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_delete", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		err := decorator.Delete(ctx, path)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_RecordsErrorMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		path := "/app/nonexistent"
		expectedError := secretsDomain.ErrSecretNotFound

		// Setup expectations
		mockUseCase.EXPECT().
			Delete(ctx, path).
			Return(expectedError).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_delete", "error").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_delete", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		err := decorator.Delete(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

// TestMetricsDecorator_List tests the List method with metrics.
func TestMetricsDecorator_List(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_RecordsSuccessMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		offset := 0
		limit := 10
		expectedSecrets := []*secretsDomain.Secret{
			{
				ID:        uuid.Must(uuid.NewV7()),
				Path:      "/app/key1",
				Version:   1,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        uuid.Must(uuid.NewV7()),
				Path:      "/app/key2",
				Version:   1,
				CreatedAt: time.Now().UTC(),
			},
		}

		// Setup expectations
		mockUseCase.EXPECT().
			List(ctx, offset, limit).
			Return(expectedSecrets, nil).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_list", "success").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_list", mock.AnythingOfType("time.Duration"), "success").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.List(ctx, offset, limit)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedSecrets, result)
		assert.Len(t, result, 2)
	})

	t.Run("Error_RecordsErrorMetrics", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockUseCase := secretsUsecaseMocks.NewMockSecretUseCase(t)
		mockMetrics := &mockBusinessMetrics{}

		offset := 0
		limit := 10
		expectedError := errors.New("database error")

		// Setup expectations
		mockUseCase.EXPECT().
			List(ctx, offset, limit).
			Return(nil, expectedError).
			Once()

		mockMetrics.On("RecordOperation", ctx, "secrets", "secret_list", "error").
			Return().
			Once()

		mockMetrics.On("RecordDuration", ctx, "secrets", "secret_list", mock.AnythingOfType("time.Duration"), "error").
			Return().
			Once()

		// Execute
		decorator := NewSecretUseCaseWithMetrics(mockUseCase, mockMetrics)
		result, err := decorator.List(ctx, offset, limit)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}
