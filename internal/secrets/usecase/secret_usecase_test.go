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
	cryptoServiceMocks "github.com/allisson/secrets/internal/crypto/service/mocks"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	secretsUsecaseMocks "github.com/allisson/secrets/internal/secrets/usecase/mocks"
)

// TestSecretUseCase_CreateOrUpdate tests the CreateOrUpdate method of secretUseCase.
func TestSecretUseCase_CreateOrUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_CreateNewSecret", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Create test data
		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		value := []byte("secret-value")

		dekID := uuid.Must(uuid.NewV7())
		dek := cryptoDomain.Dek{
			ID:           dekID,
			KekID:        kekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		ciphertext := []byte("encrypted-secret")
		nonce := []byte("secret-nonce")
		dekKey := make([]byte, 32)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(mock.Anything, path).
			Return(nil, secretsDomain.ErrSecretNotFound).
			Once()

		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, &dek).
			Return(nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(&dek, kek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Encrypt(value, mock.Anything).
			Return(ciphertext, nonce, nil).
			Once()

		mockSecretRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(secret *secretsDomain.Secret) bool {
				return secret.Path == path &&
					secret.Version == 1 &&
					secret.DekID == dekID
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, path, secret.Path)
		assert.Equal(t, uint(1), secret.Version)
		assert.Equal(t, dekID, secret.DekID)
	})

	t.Run("Success_UpdateExistingSecret", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Create test data
		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		value := []byte("new-secret-value")

		existingSecret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    2,
			DekID:      uuid.Must(uuid.NewV7()),
			Ciphertext: []byte("old-encrypted-secret"),
			Nonce:      []byte("old-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		dekID := uuid.Must(uuid.NewV7())
		dek := cryptoDomain.Dek{
			ID:           dekID,
			KekID:        kekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		ciphertext := []byte("new-encrypted-secret")
		nonce := []byte("new-secret-nonce")
		dekKey := make([]byte, 32)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(mock.Anything, path).
			Return(existingSecret, nil).
			Once()

		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, &dek).
			Return(nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(&dek, kek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Encrypt(value, mock.Anything).
			Return(ciphertext, nonce, nil).
			Once()

		mockSecretRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(secret *secretsDomain.Secret) bool {
				return secret.Path == path &&
					secret.Version == 3 && // Incremented version
					secret.DekID == dekID
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, path, secret.Path)
		assert.Equal(t, uint(3), secret.Version)
		assert.Equal(t, dekID, secret.DekID)
	})

	t.Run("Error_ActiveKekNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create empty KEK chain (no KEKs available)
		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		path := "app/api-key"
		value := []byte("secret-value")

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.True(t, errors.Is(err, cryptoDomain.ErrKekNotFound))
	})

	t.Run("Error_SecretRepoGetByPathFails", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		value := []byte("secret-value")
		expectedError := errors.New("database error")

		// Setup expectations
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Once()

		mockSecretRepo.EXPECT().
			GetByPath(mock.Anything, path).
			Return(nil, expectedError).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_CreateDekFails", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		value := []byte("secret-value")
		expectedError := errors.New("failed to create dek")

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(mock.Anything, path).
			Return(nil, secretsDomain.ErrSecretNotFound).
			Once()

		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(expectedError).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(cryptoDomain.Dek{}, expectedError).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_SecretValueTooLarge", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		path := "app/api-key"
		value := make([]byte, 10) // 10 bytes

		// Use a limit of 5 bytes
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			5,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.True(t, errors.Is(err, secretsDomain.ErrSecretValueTooLarge))
	})

	t.Run("Error_InvalidPath", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		path := "/invalid/path"
		value := []byte("secret-value")

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.Equal(t, "invalid secret path format: invalid input", err.Error())
	})
}

// TestSecretUseCase_Get tests the Get method of secretUseCase.
func TestSecretUseCase_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_GetAndDecryptSecret", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Create test data
		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		dekID := uuid.Must(uuid.NewV7())
		ciphertext := []byte("encrypted-secret")
		nonce := []byte("secret-nonce")
		plaintext := []byte("secret-value")

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    1,
			DekID:      dekID,
			Ciphertext: ciphertext,
			Nonce:      nonce,
			CreatedAt:  time.Now().UTC(),
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        kekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		dekKey := make([]byte, 32)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(plaintext, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, path, result.Path)
		assert.Equal(t, plaintext, result.Plaintext)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/nonexistent"

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(nil, secretsDomain.ErrSecretNotFound).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("Error_DekNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		dekID := uuid.Must(uuid.NewV7())

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    1,
			DekID:      dekID,
			Ciphertext: []byte("encrypted-secret"),
			Nonce:      []byte("secret-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(nil, cryptoDomain.ErrDekNotFound).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrDekNotFound))
	})

	t.Run("Error_KekNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		dekID := uuid.Must(uuid.NewV7())
		differentKekID := uuid.Must(uuid.NewV7())

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    1,
			DekID:      dekID,
			Ciphertext: []byte("encrypted-secret"),
			Nonce:      []byte("secret-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        differentKekID, // Different KEK ID not in the chain
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrKekNotFound))
	})

	t.Run("Error_DecryptionFailed", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Create test data
		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		dekID := uuid.Must(uuid.NewV7())
		ciphertext := []byte("encrypted-secret")
		nonce := []byte("secret-nonce")

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    1,
			DekID:      dekID,
			Ciphertext: ciphertext,
			Nonce:      nonce,
			CreatedAt:  time.Now().UTC(),
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        kekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		dekKey := make([]byte, 32)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(nil, errors.New("decryption failed")).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrDecryptionFailed))
	})
}

// TestSecretUseCase_Delete tests the Delete method of secretUseCase.
func TestSecretUseCase_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_DeleteSecret", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"

		// Setup expectations
		mockSecretRepo.EXPECT().
			Delete(ctx, path).
			Return(nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		err := uc.Delete(ctx, path)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/nonexistent"

		// Setup expectations
		mockSecretRepo.EXPECT().
			Delete(ctx, path).
			Return(secretsDomain.ErrSecretNotFound).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		err := uc.Delete(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("Error_DeleteFails", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		expectedError := errors.New("database error")

		// Setup expectations
		mockSecretRepo.EXPECT().
			Delete(ctx, path).
			Return(expectedError).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		err := uc.Delete(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

// TestSecretUseCase_GetByVersion tests the GetByVersion method of secretUseCase.
func TestSecretUseCase_GetByVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_GetSpecificVersion", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Create test data
		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		version := uint(2)
		dekID := uuid.Must(uuid.NewV7())
		ciphertext := []byte("encrypted-secret")
		nonce := []byte("secret-nonce")
		plaintext := []byte("secret-value")

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      dekID,
			Ciphertext: ciphertext,
			Nonce:      nonce,
			CreatedAt:  time.Now().UTC(),
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        kekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		dekKey := make([]byte, 32)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPathAndVersion(ctx, path, version).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(plaintext, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.GetByVersion(ctx, path, version)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, path, result.Path)
		assert.Equal(t, version, result.Version)
		assert.Equal(t, plaintext, result.Plaintext)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/nonexistent"
		version := uint(1)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPathAndVersion(ctx, path, version).
			Return(nil, secretsDomain.ErrSecretNotFound).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.GetByVersion(ctx, path, version)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("Error_DecryptionFailed", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		version := uint(1)
		dekID := uuid.Must(uuid.NewV7())
		ciphertext := []byte("encrypted-secret")
		nonce := []byte("secret-nonce")

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      dekID,
			Ciphertext: ciphertext,
			Nonce:      nonce,
			CreatedAt:  time.Now().UTC(),
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        kekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		dekKey := make([]byte, 32)

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPathAndVersion(ctx, path, version).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(nil, errors.New("decryption failed")).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.GetByVersion(ctx, path, version)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrDecryptionFailed))
	})

	t.Run("Error_DekNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		version := uint(1)
		dekID := uuid.Must(uuid.NewV7())

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      dekID,
			Ciphertext: []byte("encrypted-secret"),
			Nonce:      []byte("secret-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPathAndVersion(ctx, path, version).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(nil, cryptoDomain.ErrDekNotFound).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.GetByVersion(ctx, path, version)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("Error_KekNotFound", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekID := uuid.Must(uuid.NewV7())
		kek := &cryptoDomain.Kek{
			ID:           kekID,
			MasterKeyID:  "test-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			Key:          make([]byte, 32),
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("kek-nonce"),
			Version:      1,
			CreatedAt:    time.Now().UTC(),
		}
		kekChain := createKekChain([]*cryptoDomain.Kek{kek})
		defer kekChain.Close()

		path := "app/api-key"
		version := uint(1)
		dekID := uuid.Must(uuid.NewV7())
		differentKekID := uuid.Must(uuid.NewV7()) // Different KEK ID

		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      dekID,
			Ciphertext: []byte("encrypted-secret"),
			Nonce:      []byte("secret-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        differentKekID, // KEK not in chain
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
			CreatedAt:    time.Now().UTC(),
		}

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPathAndVersion(ctx, path, version).
			Return(secret, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		result, err := uc.GetByVersion(ctx, path, version)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrKekNotFound))
	})
}

// TestSecretUseCase_PurgeDeleted tests the PurgeDeleted method of secretUseCase.
func TestSecretUseCase_PurgeDeleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_PurgeDeletedSecrets", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		olderThanDays := 30
		dryRun := false
		expectedCount := int64(5)

		// Setup expectations
		mockSecretRepo.EXPECT().
			HardDelete(ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(expectedCount, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		count, err := uc.PurgeDeleted(ctx, olderThanDays, dryRun)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("Success_DryRun", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		olderThanDays := 60
		dryRun := true
		expectedCount := int64(10)

		// Setup expectations
		mockSecretRepo.EXPECT().
			HardDelete(ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(expectedCount, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		count, err := uc.PurgeDeleted(ctx, olderThanDays, dryRun)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("Success_NoSecretsToDelete", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		olderThanDays := 90
		dryRun := false
		expectedCount := int64(0)

		// Setup expectations
		mockSecretRepo.EXPECT().
			HardDelete(ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(expectedCount, nil).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		count, err := uc.PurgeDeleted(ctx, olderThanDays, dryRun)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("Error_NegativeDays", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		olderThanDays := -5
		dryRun := false

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		count, err := uc.PurgeDeleted(ctx, olderThanDays, dryRun)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "olderThanDays must be non-negative")
	})

	t.Run("Error_RepositoryFails", func(t *testing.T) {
		t.Parallel()
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		olderThanDays := 30
		dryRun := false
		expectedError := errors.New("database error")

		// Setup expectations
		mockSecretRepo.EXPECT().
			HardDelete(ctx, mock.AnythingOfType("time.Time"), dryRun).
			Return(int64(0), expectedError).
			Once()

		// Execute
		uc := NewSecretUseCase(
			mockTxManager,
			mockDekRepo,
			mockSecretRepo,
			kekChain,
			mockAEADManager,
			mockKeyManager,
			cryptoDomain.AESGCM,
			524288,
		)
		count, err := uc.PurgeDeleted(ctx, olderThanDays, dryRun)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, expectedError, err)
	})
}

// createKekChain is a helper function to create a KEK chain for testing.
func createKekChain(keks []*cryptoDomain.Kek) *cryptoDomain.KekChain {
	return cryptoDomain.NewKekChain(keks)
}
