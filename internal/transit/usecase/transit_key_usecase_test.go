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
	serviceMocks "github.com/allisson/secrets/internal/crypto/service/mocks"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	usecaseMocks "github.com/allisson/secrets/internal/transit/usecase/mocks"
)

// Helper function to create a test KEK chain
func createTestKekChain(activeKekID uuid.UUID, kek *cryptoDomain.Kek) *cryptoDomain.KekChain {
	keks := []*cryptoDomain.Kek{kek}
	return cryptoDomain.NewKekChain(keks)
}

// Helper function to create a test KEK
func createTestKek() *cryptoDomain.Kek {
	return &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "test-master-key",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek"),
		Key:          make([]byte, 32),
		Nonce:        []byte("nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
}

// Helper function to create a test DEK
func createTestDek(kekID uuid.UUID) *cryptoDomain.Dek {
	return &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek"),
		Nonce:        []byte("nonce"),
		CreatedAt:    time.Now().UTC(),
	}
}

// Helper function to create a test transit key
func createTestTransitKey(name string, version uint, dekID uuid.UUID) *transitDomain.TransitKey {
	return &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Version:   version,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
}

// TestTransitKeyUseCase_Create tests the Create method of transitKeyUseCase.
func TestTransitKeyUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateTransitKeyWithAESGCM", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		expectedDek := createTestDek(kek.ID)

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(*expectedDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(dek *cryptoDomain.Dek) bool {
				return dek.ID == expectedDek.ID && dek.KekID == expectedDek.KekID
			})).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(tk *transitDomain.TransitKey) bool {
				return tk.Name == "test-key" && tk.Version == 1 && tk.DekID == expectedDek.ID
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Create(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, transitKey)
		assert.Equal(t, "test-key", transitKey.Name)
		assert.Equal(t, uint(1), transitKey.Version)
		assert.Equal(t, expectedDek.ID, transitKey.DekID)
	})

	t.Run("Success_CreateTransitKeyWithChaCha20", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		expectedDek := createTestDek(kek.ID)
		expectedDek.Algorithm = cryptoDomain.ChaCha20

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.ChaCha20).
			Return(*expectedDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Create(ctx, "test-key", cryptoDomain.ChaCha20)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, transitKey)
		assert.Equal(t, uint(1), transitKey.Version)
	})

	t.Run("Error_TransitKeyAlreadyExists", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		existingTransitKey := createTestTransitKey("test-key", 1, uuid.Must(uuid.NewV7()))

		// Setup expectations - GetByNameAndVersion should return existing key
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(existingTransitKey, nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Create(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, transitKey)
		assert.True(t, apperrors.Is(err, transitDomain.ErrTransitKeyAlreadyExists))
		assert.True(t, apperrors.Is(err, apperrors.ErrConflict))
	})

	t.Run("Error_DekCreationFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		expectedError := errors.New("dek creation failed")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(cryptoDomain.Dek{}, expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Create(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, transitKey)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_DekPersistenceFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		expectedDek := createTestDek(kek.ID)
		expectedError := errors.New("database error")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(*expectedDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Create(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, transitKey)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_TransitKeyPersistenceFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		expectedDek := createTestDek(kek.ID)
		expectedError := errors.New("database error")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(*expectedDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Create(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, transitKey)
		assert.Equal(t, expectedError, err)
	})
}

// TestTransitKeyUseCase_Rotate tests the Rotate method of transitKeyUseCase.
func TestTransitKeyUseCase_Rotate(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RotateToNewVersion", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		existingDek := createTestDek(kek.ID)
		currentKey := createTestTransitKey("test-key", 1, existingDek.ID)
		newDek := createTestDek(kek.ID)

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			GetByName(mock.Anything, "test-key").
			Return(currentKey, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(*newDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(dek *cryptoDomain.Dek) bool {
				return dek.ID == newDek.ID
			})).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(tk *transitDomain.TransitKey) bool {
				return tk.Name == "test-key" && tk.Version == 2 && tk.DekID == newDek.ID
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Rotate(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, transitKey)
		assert.Equal(t, "test-key", transitKey.Name)
		assert.Equal(t, uint(2), transitKey.Version)
		assert.Equal(t, newDek.ID, transitKey.DekID)
	})

	t.Run("Success_RotateCreatesFirstKeyIfNoneExist", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		newDek := createTestDek(kek.ID)

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			GetByName(mock.Anything, "test-key").
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockTransitRepo.EXPECT().
			GetByNameAndVersion(mock.Anything, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(*newDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(tk *transitDomain.TransitKey) bool {
				return tk.Name == "test-key" && tk.Version == 1
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Rotate(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, transitKey)
		assert.Equal(t, uint(1), transitKey.Version)
	})

	t.Run("Success_RotateWithDifferentAlgorithm", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		existingDek := createTestDek(kek.ID)
		currentKey := createTestTransitKey("test-key", 1, existingDek.ID)
		newDek := createTestDek(kek.ID)
		newDek.Algorithm = cryptoDomain.ChaCha20

		// Setup expectations
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			GetByName(mock.Anything, "test-key").
			Return(currentKey, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.ChaCha20).
			Return(*newDek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		mockTransitRepo.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Rotate(ctx, "test-key", cryptoDomain.ChaCha20)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, transitKey)
		assert.Equal(t, uint(2), transitKey.Version)
	})

	t.Run("Error_GetByNameFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		expectedError := errors.New("database error")

		// Setup expectations
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(expectedError).
			Once()

		mockTransitRepo.EXPECT().
			GetByName(mock.Anything, "test-key").
			Return(nil, expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Rotate(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, transitKey)
	})

	t.Run("Error_DekCreationFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		existingDek := createTestDek(kek.ID)
		currentKey := createTestTransitKey("test-key", 1, existingDek.ID)
		expectedError := errors.New("dek creation failed")

		// Setup expectations
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				_ = fn(ctx)
			}).
			Return(expectedError).
			Once()

		mockTransitRepo.EXPECT().
			GetByName(mock.Anything, "test-key").
			Return(currentKey, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(kek, cryptoDomain.AESGCM).
			Return(cryptoDomain.Dek{}, expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		transitKey, err := uc.Rotate(ctx, "test-key", cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, transitKey)
	})
}

// TestTransitKeyUseCase_Delete tests the Delete method of transitKeyUseCase.
func TestTransitKeyUseCase_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_SoftDeleteTransitKey", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		transitKeyID := uuid.Must(uuid.NewV7())

		// Setup expectations
		mockTransitRepo.EXPECT().
			Delete(ctx, transitKeyID).
			Return(nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		err := uc.Delete(ctx, transitKeyID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_DeleteFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		transitKeyID := uuid.Must(uuid.NewV7())
		expectedError := errors.New("database error")

		// Setup expectations
		mockTransitRepo.EXPECT().
			Delete(ctx, transitKeyID).
			Return(expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		err := uc.Delete(ctx, transitKeyID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

// TestTransitKeyUseCase_Encrypt tests the Encrypt method of transitKeyUseCase.
func TestTransitKeyUseCase_Encrypt(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_EncryptPlaintext", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockCipher := serviceMocks.NewMockAEAD(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		plaintext := []byte("sensitive data")
		dekKey := make([]byte, 32)
		ciphertext := []byte("encrypted-data")
		nonce := []byte("random-nonce")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAeadManager.EXPECT().
			CreateCipher(dekKey, dek.Algorithm).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			NonceSize().
			Return(12).
			Maybe()

		mockCipher.EXPECT().
			Encrypt(plaintext, mock.Anything).
			Return(ciphertext, nonce, nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		blob, err := uc.Encrypt(ctx, "test-key", plaintext)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, blob)
		assert.Equal(t, uint(1), blob.Version)
		assert.NotNil(t, blob.Ciphertext)
		assert.Nil(t, blob.Plaintext)
	})

	t.Run("Error_TransitKeyNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		plaintext := []byte("sensitive data")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		blob, err := uc.Encrypt(ctx, "test-key", plaintext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, blob)
		assert.True(t, apperrors.Is(err, transitDomain.ErrTransitKeyNotFound))
	})

	t.Run("Error_DekNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		plaintext := []byte("sensitive data")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(nil, cryptoDomain.ErrDekNotFound).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		blob, err := uc.Encrypt(ctx, "test-key", plaintext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, blob)
		assert.True(t, apperrors.Is(err, cryptoDomain.ErrDekNotFound))
	})

	t.Run("Error_DecryptionFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		plaintext := []byte("sensitive data")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(nil, cryptoDomain.ErrDecryptionFailed).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		blob, err := uc.Encrypt(ctx, "test-key", plaintext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, blob)
		assert.True(t, apperrors.Is(err, cryptoDomain.ErrDecryptionFailed))
	})

	t.Run("Error_EncryptionFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockCipher := serviceMocks.NewMockAEAD(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		plaintext := []byte("sensitive data")
		dekKey := make([]byte, 32)
		expectedError := errors.New("encryption failed")

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAeadManager.EXPECT().
			CreateCipher(dekKey, dek.Algorithm).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			NonceSize().
			Return(12).
			Maybe()

		mockCipher.EXPECT().
			Encrypt(plaintext, mock.Anything).
			Return(nil, nil, expectedError).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		blob, err := uc.Encrypt(ctx, "test-key", plaintext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, blob)
	})
}

// TestTransitKeyUseCase_Decrypt tests the Decrypt method of transitKeyUseCase.
func TestTransitKeyUseCase_Decrypt(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_DecryptCiphertext", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockCipher := serviceMocks.NewMockAEAD(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		plaintext := []byte("sensitive data")
		dekKey := make([]byte, 32)

		// Create a valid encrypted blob string
		nonce := []byte("012345678901") // 12 bytes
		ciphertext := []byte("encrypted-data-with-tag")
		//nolint:gocritic // intentionally creating new slice for test data
		encryptedData := append(nonce, ciphertext...)
		blob := transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: encryptedData,
		}
		ciphertextStr := blob.String()

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAeadManager.EXPECT().
			CreateCipher(dekKey, dek.Algorithm).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			NonceSize().
			Return(12).
			Maybe()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(plaintext, nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		resultBlob, err := uc.Decrypt(ctx, "test-key", ciphertextStr)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, resultBlob)
		assert.Equal(t, uint(1), resultBlob.Version)
		assert.Equal(t, plaintext, resultBlob.Plaintext)
		assert.Nil(t, resultBlob.Ciphertext)
	})

	t.Run("Success_DecryptWithOldVersion", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockCipher := serviceMocks.NewMockAEAD(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 5, dek.ID)
		plaintext := []byte("old data")
		dekKey := make([]byte, 32)

		// Create a valid encrypted blob string with version 5
		nonce := []byte("012345678901")
		ciphertext := []byte("old-encrypted-data")
		//nolint:gocritic // intentionally creating new slice for test data
		encryptedData := append(nonce, ciphertext...)
		blob := transitDomain.EncryptedBlob{
			Version:    5,
			Ciphertext: encryptedData,
		}
		ciphertextStr := blob.String()

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(5)).
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAeadManager.EXPECT().
			CreateCipher(dekKey, dek.Algorithm).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			NonceSize().
			Return(12).
			Maybe()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(plaintext, nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		resultBlob, err := uc.Decrypt(ctx, "test-key", ciphertextStr)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, resultBlob)
		assert.Equal(t, uint(5), resultBlob.Version)
		assert.Equal(t, plaintext, resultBlob.Plaintext)
	})

	t.Run("Error_InvalidBlobFormat", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		invalidCiphertext := "invalid-blob-format"

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		blob, err := uc.Decrypt(ctx, "test-key", invalidCiphertext)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, blob)
		assert.True(t, apperrors.Is(err, transitDomain.ErrInvalidBlobFormat))
	})

	t.Run("Error_TransitKeyNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		// Create a valid encrypted blob string
		blob := transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: []byte("data"),
		}
		ciphertextStr := blob.String()

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		resultBlob, err := uc.Decrypt(ctx, "test-key", ciphertextStr)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultBlob)
		assert.True(t, apperrors.Is(err, transitDomain.ErrTransitKeyNotFound))
	})

	t.Run("Error_CiphertextTooShort", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockCipher := serviceMocks.NewMockAEAD(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		dekKey := make([]byte, 32)

		// Create a blob with data shorter than nonce size (12 bytes)
		blob := transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: []byte("short"), // Only 5 bytes
		}
		ciphertextStr := blob.String()

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAeadManager.EXPECT().
			CreateCipher(dekKey, dek.Algorithm).
			Return(mockCipher, nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		resultBlob, err := uc.Decrypt(ctx, "test-key", ciphertextStr)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultBlob)
		assert.True(t, apperrors.Is(err, cryptoDomain.ErrDecryptionFailed))
	})

	t.Run("Error_DecryptionFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockCipher := serviceMocks.NewMockAEAD(t)

		// Create test data
		kek := createTestKek()
		kekChain := createTestKekChain(kek.ID, kek)
		defer kekChain.Close()

		dek := createTestDek(kek.ID)
		transitKey := createTestTransitKey("test-key", 1, dek.ID)
		dekKey := make([]byte, 32)

		// Create a valid encrypted blob string
		nonce := []byte("012345678901")
		ciphertext := []byte("encrypted-data")
		//nolint:gocritic // intentionally creating new slice for test data
		encryptedData := append(nonce, ciphertext...)
		blob := transitDomain.EncryptedBlob{
			Version:    1,
			Ciphertext: encryptedData,
		}
		ciphertextStr := blob.String()

		// Setup expectations
		mockTransitRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(transitKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dek.ID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, kek).
			Return(dekKey, nil).
			Once()

		mockAeadManager.EXPECT().
			CreateCipher(dekKey, dek.Algorithm).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			NonceSize().
			Return(12).
			Maybe()

		mockCipher.EXPECT().
			Decrypt(ciphertext, nonce, mock.Anything).
			Return(nil, errors.New("authentication failed")).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager, mockTransitRepo, mockDekRepo, mockKeyManager, mockAeadManager, kekChain,
		)
		resultBlob, err := uc.Decrypt(ctx, "test-key", ciphertextStr)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultBlob)
		assert.True(t, apperrors.Is(err, cryptoDomain.ErrDecryptionFailed))
	})
}

// TestTransitKeyUseCase_List tests the List method of transitKeyUseCase.
func TestTransitKeyUseCase_List(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_ListTransitKeys", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		expectedKeys := []*transitDomain.TransitKey{
			{
				ID:      uuid.Must(uuid.NewV7()),
				Name:    "key-1",
				Version: 1,
			},
			{
				ID:      uuid.Must(uuid.NewV7()),
				Name:    "key-2",
				Version: 2,
			},
		}

		mockTransitRepo.EXPECT().
			List(ctx, 0, 10).
			Return(expectedKeys, nil).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager,
			mockTransitRepo,
			mockDekRepo,
			mockKeyManager,
			mockAeadManager,
			kekChain,
		)

		keys, err := uc.List(ctx, 0, 10)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, keys, 2)
		assert.Equal(t, "key-1", keys[0].Name)
		assert.Equal(t, uint(1), keys[0].Version)
		assert.Equal(t, "key-2", keys[1].Name)
		assert.Equal(t, uint(2), keys[1].Version)
	})

	t.Run("Error_RepositoryFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTransitRepo := usecaseMocks.NewMockTransitKeyRepository(t)
		mockDekRepo := usecaseMocks.NewMockDekRepository(t)
		mockAeadManager := serviceMocks.NewMockAEADManager(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{})
		defer kekChain.Close()

		expectedErr := errors.New("db error")

		mockTransitRepo.EXPECT().
			List(ctx, 0, 10).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewTransitKeyUseCase(
			mockTxManager,
			mockTransitRepo,
			mockDekRepo,
			mockKeyManager,
			mockAeadManager,
			kekChain,
		)

		keys, err := uc.List(ctx, 0, 10)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, keys)
		assert.Equal(t, expectedErr, err)
	})
}
