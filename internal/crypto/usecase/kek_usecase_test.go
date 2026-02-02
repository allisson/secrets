package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
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

	t.Run("Error_MasterKeyNotFound", func(t *testing.T) {
		// Create a master key
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}

		// Set up environment with keys, but activeID points to non-existent key
		encodedKey := base64.StdEncoding.EncodeToString(masterKey.Key)
		err := os.Setenv("MASTER_KEYS", masterKeyID+":"+encodedKey)
		assert.NoError(t, err)
		err = os.Setenv("ACTIVE_MASTER_KEY_ID", "non-existent-key")
		assert.NoError(t, err)

		// This should fail because active master key doesn't exist in the chain
		masterKeyChain, err := cryptoDomain.LoadMasterKeyChainFromEnv()

		// Assert - should fail during chain creation
		assert.Error(t, err)
		assert.Nil(t, masterKeyChain)
		assert.Contains(t, err.Error(), "active master key not found")
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

	t.Run("Error_MasterKeyNotFound", func(t *testing.T) {
		// Create a master key
		masterKeyID := "test-master-key"
		masterKey := &cryptoDomain.MasterKey{
			ID:  masterKeyID,
			Key: make([]byte, 32),
		}

		// Set up environment with keys, but activeID points to non-existent key
		encodedKey := base64.StdEncoding.EncodeToString(masterKey.Key)
		err := os.Setenv("MASTER_KEYS", masterKeyID+":"+encodedKey)
		assert.NoError(t, err)
		err = os.Setenv("ACTIVE_MASTER_KEY_ID", "non-existent-key")
		assert.NoError(t, err)

		// This should fail because active master key doesn't exist in the chain
		masterKeyChain, err := cryptoDomain.LoadMasterKeyChainFromEnv()

		// Assert - should fail during chain creation
		assert.Error(t, err)
		assert.Nil(t, masterKeyChain)
		assert.Contains(t, err.Error(), "active master key not found")
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
// It uses environment variables to create a real MasterKeyChain instance.
func createMasterKeyChain(activeID string, masterKey *cryptoDomain.MasterKey) *cryptoDomain.MasterKeyChain {
	// Encode the master key to base64
	encodedKey := base64.StdEncoding.EncodeToString(masterKey.Key)

	// Set environment variables
	err := os.Setenv("MASTER_KEYS", masterKey.ID+":"+encodedKey)
	if err != nil {
		panic("failed to set MASTER_KEYS env var: " + err.Error())
	}
	err = os.Setenv("ACTIVE_MASTER_KEY_ID", activeID)
	if err != nil {
		panic("failed to set ACTIVE_MASTER_KEY_ID env var: " + err.Error())
	}

	// Load the master key chain from environment
	mkc, err := cryptoDomain.LoadMasterKeyChainFromEnv()
	if err != nil {
		panic("failed to create master key chain for testing: " + err.Error())
	}

	return mkc
}
