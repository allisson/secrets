package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gocloud.dev/secrets"

	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	serviceMocks "github.com/allisson/secrets/internal/crypto/service/mocks"
	usecaseMocks "github.com/allisson/secrets/internal/crypto/usecase/mocks"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
)

// TestKekUseCase_Create tests the Create method of kekUseCase.
func TestKekUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateKekWithAESGCM", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedKek := cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		// Setup expectations
		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.AESGCM).
			Return(expectedKek, nil).
			Once()

		mockKekRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(kek *cryptoDomain.Kek) bool {
				return kek.ID == expectedKek.ID &&
					kek.MasterKeyID == expectedKek.MasterKeyID &&
					kek.Algorithm == expectedKek.Algorithm
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Success_CreateKekWithChaCha20", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedKek := cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.ChaCha20,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		// Setup expectations
		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.ChaCha20).
			Return(expectedKek, nil).
			Once()

		mockKekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Create(ctx, masterKeyChain, cryptoDomain.ChaCha20)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_KeyManagerCreateKekFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedErr := cryptoDomain.ErrUnsupportedAlgorithm

		// Setup expectations
		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.AESGCM).
			Return(cryptoDomain.Kek{}, expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Error_RepositoryCreateFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedKek := cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		expectedErr := errors.New("database error")

		// Setup expectations
		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.AESGCM).
			Return(expectedKek, nil).
			Once()

		mockKekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

// TestKekUseCase_Rotate tests the Rotate method of kekUseCase.
func TestKekUseCase_Rotate(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RotateKek", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		currentKek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("old-encrypted-kek"),
			Nonce:        []byte("old-nonce"),
			Version:      1,
		}

		newKek := cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.ChaCha20,
			EncryptedKey: []byte("new-encrypted-kek"),
			Nonce:        []byte("new-nonce"),
			Version:      2,
		}

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockKekRepo.EXPECT().
			List(mock.Anything).
			Return([]*cryptoDomain.Kek{currentKek}, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.ChaCha20).
			Return(newKek, nil).
			Once()

		mockKekRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(kek *cryptoDomain.Kek) bool {
				return kek.Version == 2
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Rotate(ctx, masterKeyChain, cryptoDomain.ChaCha20)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Success_CreateFirstKekWhenNoneExist", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedKek := cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		// List returns empty slice (no KEKs exist)
		mockKekRepo.EXPECT().
			List(mock.Anything).
			Return([]*cryptoDomain.Kek{}, nil).
			Once()

		// Should create first KEK with version 1
		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.AESGCM).
			Return(expectedKek, nil).
			Once()

		mockKekRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(kek *cryptoDomain.Kek) bool {
				return kek.ID == expectedKek.ID &&
					kek.MasterKeyID == expectedKek.MasterKeyID &&
					kek.Algorithm == expectedKek.Algorithm &&
					kek.Version == 1
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Rotate(ctx, masterKeyChain, cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_ListKeksFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedErr := errors.New("database error")

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(expectedErr).
			Once()

		mockKekRepo.EXPECT().
			List(mock.Anything).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Rotate(ctx, masterKeyChain, cryptoDomain.ChaCha20)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Error_CreateNewKekFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		currentKek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("old-encrypted-kek"),
			Nonce:        []byte("old-nonce"),
			Version:      1,
		}

		expectedErr := cryptoDomain.ErrUnsupportedAlgorithm

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(expectedErr).
			Once()

		mockKekRepo.EXPECT().
			List(mock.Anything).
			Return([]*cryptoDomain.Kek{currentKek}, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.ChaCha20).
			Return(cryptoDomain.Kek{}, expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Rotate(ctx, masterKeyChain, cryptoDomain.ChaCha20)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Error_CreateNewKekInRepositoryFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		currentKek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("old-encrypted-kek"),
			Nonce:        []byte("old-nonce"),
			Version:      1,
		}

		newKek := cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.ChaCha20,
			EncryptedKey: []byte("new-encrypted-kek"),
			Nonce:        []byte("new-nonce"),
			Version:      2,
		}

		expectedErr := errors.New("database error")

		// Setup expectations for transaction
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(expectedErr).
			Once()

		mockKekRepo.EXPECT().
			List(mock.Anything).
			Return([]*cryptoDomain.Kek{currentKek}, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateKek(masterKey, cryptoDomain.ChaCha20).
			Return(newKek, nil).
			Once()

		mockKekRepo.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		err := uc.Rotate(ctx, masterKeyChain, cryptoDomain.ChaCha20)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

// TestKekUseCase_Unwrap tests the Unwrap method of kekUseCase.
func TestKekUseCase_Unwrap(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_UnwrapSingleKek", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		kek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		decryptedKey := make([]byte, 32)

		// Setup expectations
		mockKekRepo.EXPECT().
			List(ctx).
			Return([]*cryptoDomain.Kek{kek}, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptKek(kek, masterKey).
			Return(decryptedKey, nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		kekChain, err := uc.Unwrap(ctx, masterKeyChain)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, kekChain)
		assert.Equal(t, kek.ID, kekChain.ActiveKekID())

		// Verify the KEK in the chain has the decrypted key
		retrievedKek, found := kekChain.Get(kek.ID)
		assert.True(t, found)
		assert.Equal(t, decryptedKey, retrievedKek.Key)
	})

	t.Run("Error_NoKekFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		// Setup expectations
		mockKekRepo.EXPECT().
			List(ctx).
			Return([]*cryptoDomain.Kek{}, nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		kekChain, err := uc.Unwrap(ctx, masterKeyChain)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, kekChain)
		assert.ErrorIs(t, err, cryptoDomain.ErrKekNotFound)
	})

	t.Run("Success_UnwrapMultipleKeks", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		kek1 := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek-1"),
			Nonce:        []byte("nonce-1"),
			Version:      2,
		}

		kek2 := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.ChaCha20,
			EncryptedKey: []byte("encrypted-kek-2"),
			Nonce:        []byte("nonce-2"),
			Version:      1,
		}

		decryptedKey1 := make([]byte, 32)
		decryptedKey2 := make([]byte, 32)

		// Setup expectations
		mockKekRepo.EXPECT().
			List(ctx).
			Return([]*cryptoDomain.Kek{kek1, kek2}, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptKek(kek1, masterKey).
			Return(decryptedKey1, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptKek(kek2, masterKey).
			Return(decryptedKey2, nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		kekChain, err := uc.Unwrap(ctx, masterKeyChain)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, kekChain)
		assert.Equal(t, kek1.ID, kekChain.ActiveKekID())

		// Verify both KEKs are in the chain
		retrievedKek1, found1 := kekChain.Get(kek1.ID)
		assert.True(t, found1)
		assert.Equal(t, decryptedKey1, retrievedKek1.Key)

		retrievedKek2, found2 := kekChain.Get(kek2.ID)
		assert.True(t, found2)
		assert.Equal(t, decryptedKey2, retrievedKek2.Key)
	})

	t.Run("Error_ListKeksFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		expectedErr := errors.New("database error")

		// Setup expectations
		mockKekRepo.EXPECT().
			List(ctx).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		kekChain, err := uc.Unwrap(ctx, masterKeyChain)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, kekChain)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Error_MasterKeyNotFoundForKek", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		// KEK with a different master key ID that doesn't exist in the chain
		kek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  "non-existent-master-key",
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		// Setup expectations
		mockKekRepo.EXPECT().
			List(ctx).
			Return([]*cryptoDomain.Kek{kek}, nil).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		kekChain, err := uc.Unwrap(ctx, masterKeyChain)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, kekChain)
		assert.Contains(t, err.Error(), "master key not found")
	})

	t.Run("Error_DecryptKekFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockKekRepo := usecaseMocks.NewMockKekRepository(t)
		mockKeyManager := serviceMocks.NewMockKeyManager(t)

		// Create test data
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}
		masterKeyChain := createMasterKeyChain(masterKeyID, masterKey)
		defer masterKeyChain.Close()

		kek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  masterKeyID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-kek"),
			Nonce:        []byte("nonce"),
			Version:      1,
		}

		expectedErr := cryptoDomain.ErrDecryptionFailed

		// Setup expectations
		mockKekRepo.EXPECT().
			List(ctx).
			Return([]*cryptoDomain.Kek{kek}, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptKek(kek, masterKey).
			Return(nil, expectedErr).
			Once()

		// Execute
		uc := NewKekUseCase(mockTxManager, mockKekRepo, mockKeyManager)
		kekChain, err := uc.Unwrap(ctx, masterKeyChain)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, kekChain)
		assert.ErrorIs(t, err, expectedErr)
	})
}

// createMasterKeyChain is a helper function to create a MasterKeyChain for testing.
// It uses the localsecrets KMS provider to create a properly encrypted master key chain.
func createMasterKeyChain(activeID string, masterKey *cryptoDomain.MasterKey) *cryptoDomain.MasterKeyChain {
	ctx := context.Background()

	// Generate a random KMS key for localsecrets provider
	kmsKey := make([]byte, 32)
	if _, err := rand.Read(kmsKey); err != nil {
		panic("failed to generate KMS key: " + err.Error())
	}
	kmsKeyURI := "base64key://" + base64.URLEncoding.EncodeToString(kmsKey)

	// Open KMS keeper
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		panic("failed to open KMS keeper: " + err.Error())
	}
	defer func() {
		_ = keeperInterface.Close()
	}()

	// Type assert to get Encrypt method
	keeper, ok := keeperInterface.(*secrets.Keeper)
	if !ok {
		panic("keeper should be *secrets.Keeper")
	}

	// Encrypt master key with KMS
	ciphertext, err := keeper.Encrypt(ctx, masterKey.Key)
	if err != nil {
		panic("failed to encrypt master key with KMS: " + err.Error())
	}

	// Encode ciphertext to base64
	encodedCiphertext := base64.StdEncoding.EncodeToString(ciphertext)

	// Set environment variables
	if err := os.Setenv("MASTER_KEYS", masterKey.ID+":"+encodedCiphertext); err != nil {
		panic("failed to set MASTER_KEYS env: " + err.Error())
	}
	if err := os.Setenv("ACTIVE_MASTER_KEY_ID", activeID); err != nil {
		panic("failed to set ACTIVE_MASTER_KEY_ID env: " + err.Error())
	}
	if err := os.Setenv("KMS_PROVIDER", "localsecrets"); err != nil {
		panic("failed to set KMS_PROVIDER env: " + err.Error())
	}
	if err := os.Setenv("KMS_KEY_URI", kmsKeyURI); err != nil {
		panic("failed to set KMS_KEY_URI env: " + err.Error())
	}

	// Load master key chain using KMS
	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   kmsKeyURI,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mkc, err := cryptoDomain.LoadMasterKeyChain(ctx, cfg, kmsService, logger)
	if err != nil {
		panic("failed to load master key chain: " + err.Error())
	}

	return mkc
}
