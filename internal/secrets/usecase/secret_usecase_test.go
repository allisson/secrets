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
	ctx := context.Background()

	t.Run("Success_CreateNewSecret", func(t *testing.T) {
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

		path := "/app/api-key"
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
			Return(nil, apperrors.ErrNotFound).
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

		path := "/app/api-key"
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
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create empty KEK chain (no KEKs available)
		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		path := "/app/api-key"
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
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.True(t, errors.Is(err, cryptoDomain.ErrKekNotFound))
	})

	t.Run("Error_SecretRepoGetByPathFails", func(t *testing.T) {
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

		path := "/app/api-key"
		value := []byte("secret-value")
		expectedError := errors.New("database error")

		// Setup expectations
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
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_CreateDekFails", func(t *testing.T) {
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

		path := "/app/api-key"
		value := []byte("secret-value")
		expectedError := errors.New("failed to create dek")

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(mock.Anything, path).
			Return(nil, apperrors.ErrNotFound).
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
		)
		secret, err := uc.CreateOrUpdate(ctx, path, value)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secret)
		assert.Equal(t, expectedError, err)
	})
}

// TestSecretUseCase_Get tests the Get method of secretUseCase.
func TestSecretUseCase_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_GetAndDecryptSecret", func(t *testing.T) {
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

		path := "/app/api-key"
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
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, path, result.Path)
		assert.Equal(t, plaintext, result.Plaintext)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
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

		path := "/app/nonexistent"

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(nil, apperrors.ErrNotFound).
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
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("Error_DekNotFound", func(t *testing.T) {
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

		path := "/app/api-key"
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
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrDekNotFound))
	})

	t.Run("Error_KekNotFound", func(t *testing.T) {
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

		path := "/app/api-key"
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
		)
		result, err := uc.Get(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, cryptoDomain.ErrKekNotFound))
	})

	t.Run("Error_DecryptionFailed", func(t *testing.T) {
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

		path := "/app/api-key"
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
	ctx := context.Background()

	t.Run("Success_DeleteSecret", func(t *testing.T) {
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

		path := "/app/api-key"
		secretID := uuid.Must(uuid.NewV7())

		secret := &secretsDomain.Secret{
			ID:         secretID,
			Path:       path,
			Version:    1,
			DekID:      uuid.Must(uuid.NewV7()),
			Ciphertext: []byte("encrypted-secret"),
			Nonce:      []byte("secret-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(secret, nil).
			Once()

		mockSecretRepo.EXPECT().
			Delete(ctx, secretID).
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
		)
		err := uc.Delete(ctx, path)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
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

		path := "/app/nonexistent"

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(nil, apperrors.ErrNotFound).
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
		)
		err := uc.Delete(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("Error_DeleteFails", func(t *testing.T) {
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

		path := "/app/api-key"
		secretID := uuid.Must(uuid.NewV7())
		expectedError := errors.New("database error")

		secret := &secretsDomain.Secret{
			ID:         secretID,
			Path:       path,
			Version:    1,
			DekID:      uuid.Must(uuid.NewV7()),
			Ciphertext: []byte("encrypted-secret"),
			Nonce:      []byte("secret-nonce"),
			CreatedAt:  time.Now().UTC(),
		}

		// Setup expectations
		mockSecretRepo.EXPECT().
			GetByPath(ctx, path).
			Return(secret, nil).
			Once()

		mockSecretRepo.EXPECT().
			Delete(ctx, secretID).
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
		)
		err := uc.Delete(ctx, path)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

// TestSecretUseCase_List tests the List method of secretUseCase.
func TestSecretUseCase_List(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_ListSecrets", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		expectedSecrets := []*secretsDomain.Secret{
			{
				ID:      uuid.Must(uuid.NewV7()),
				Path:    "sec-1",
				Version: 1,
			},
			{
				ID:      uuid.Must(uuid.NewV7()),
				Path:    "sec-2",
				Version: 2,
			},
		}

		mockSecretRepo.EXPECT().
			List(ctx, 0, 10).
			Return(expectedSecrets, nil).
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
		)

		secrets, err := uc.List(ctx, 0, 10)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, secrets, 2)
		assert.Equal(t, "sec-1", secrets[0].Path)
		assert.Equal(t, uint(1), secrets[0].Version)
		assert.Equal(t, "sec-2", secrets[1].Path)
		assert.Equal(t, uint(2), secrets[1].Version)
	})

	t.Run("Error_RepositoryFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockDekRepo := secretsUsecaseMocks.NewMockDekRepository(t)
		mockSecretRepo := secretsUsecaseMocks.NewMockSecretRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		kekChain := createKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		expectedErr := errors.New("db error")

		mockSecretRepo.EXPECT().
			List(ctx, 0, 10).
			Return(nil, expectedErr).
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
		)

		secrets, err := uc.List(ctx, 0, 10)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, secrets)
		assert.Equal(t, expectedErr, err)
	})
}

// createKekChain is a helper function to create a KEK chain for testing.
func createKekChain(keks []*cryptoDomain.Kek) *cryptoDomain.KekChain {
	if len(keks) == 0 {
		// Create a dummy KEK chain with a nil active ID
		return &cryptoDomain.KekChain{}
	}
	return cryptoDomain.NewKekChain(keks)
}
