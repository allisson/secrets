package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoServiceMocks "github.com/allisson/secrets/internal/crypto/service/mocks"
	databaseMocks "github.com/allisson/secrets/internal/database/mocks"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationMocks "github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

// createKekChain creates a test KEK chain for tokenization key tests.
func createKekChain(masterKey *cryptoDomain.MasterKey) *cryptoDomain.KekChain {
	// Create a test KEK with plaintext key populated
	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  masterKey.ID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: make([]byte, 32),
		Key:          make([]byte, 32), // Plaintext KEK key for testing
		Nonce:        make([]byte, 12),
		Version:      1,
	}

	// Create KEK chain with the test KEK (newest first)
	kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{kek})

	return kekChain
}

// getActiveKek is a helper to get the active KEK from a chain.
func getActiveKek(kekChain *cryptoDomain.KekChain) *cryptoDomain.Kek {
	activeID := kekChain.ActiveKekID()
	kek, ok := kekChain.Get(activeID)
	if !ok {
		panic("active KEK not found in chain")
	}
	return kek
}

// TestTokenizationKeyUseCase_Create tests the Create method.
func TestTokenizationKeyUseCase_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_CreateKeyWithUUIDFormat", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dek := cryptoDomain.Dek{
			ID:           uuid.Must(uuid.NewV7()),
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(activeKek, cryptoDomain.AESGCM).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(d *cryptoDomain.Dek) bool {
				return d.ID == dek.ID && d.KekID == dek.KekID
			})).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(key *tokenizationDomain.TokenizationKey) bool {
				return key.Name == "test-key" &&
					key.FormatType == tokenizationDomain.FormatUUID &&
					key.Version == 1 &&
					key.IsDeterministic == false &&
					key.DekID == dek.ID
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Create(ctx, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "test-key", key.Name)
		assert.Equal(t, tokenizationDomain.FormatUUID, key.FormatType)
		assert.Equal(t, uint(1), key.Version)
		assert.False(t, key.IsDeterministic)
	})

	t.Run("Success_CreateKeyWithLuhnPreservingDeterministic", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dek := cryptoDomain.Dek{
			ID:           uuid.Must(uuid.NewV7()),
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.ChaCha20,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByNameAndVersion(ctx, "payment-cards", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(activeKek, cryptoDomain.ChaCha20).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(key *tokenizationDomain.TokenizationKey) bool {
				return key.Name == "payment-cards" &&
					key.FormatType == tokenizationDomain.FormatLuhnPreserving &&
					key.Version == 1 &&
					key.IsDeterministic == true
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Create(
			ctx,
			"payment-cards",
			tokenizationDomain.FormatLuhnPreserving,
			true,
			cryptoDomain.ChaCha20,
		)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "payment-cards", key.Name)
		assert.Equal(t, tokenizationDomain.FormatLuhnPreserving, key.FormatType)
		assert.True(t, key.IsDeterministic)
	})

	t.Run("Error_KeyManagerCreateDekFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		expectedError := errors.New("key manager error")

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(mock.Anything, mock.Anything).
			Return(cryptoDomain.Dek{}, expectedError).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Create(ctx, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_DekRepositoryCreateFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dek := cryptoDomain.Dek{
			ID:           uuid.Must(uuid.NewV7()),
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		expectedError := errors.New("database error")

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(mock.Anything, mock.Anything).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(expectedError).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Create(ctx, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Error_TokenizationKeyRepositoryCreateFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dek := cryptoDomain.Dek{
			ID:           uuid.Must(uuid.NewV7()),
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		expectedError := errors.New("key already exists")

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(mock.Anything, mock.Anything).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Create(ctx, mock.Anything).
			Return(expectedError).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Create(ctx, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Equal(t, expectedError, err)
	})
}

// TestTokenizationKeyUseCase_Rotate tests the Rotate method.
func TestTokenizationKeyUseCase_Rotate(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RotateKey", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		existingKey := &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatNumeric,
			Version:         1,
			IsDeterministic: true,
			DekID:           uuid.Must(uuid.NewV7()),
		}

		activeKek := getActiveKek(kekChain)
		dek := cryptoDomain.Dek{
			ID:           uuid.Must(uuid.NewV7()),
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		// Setup expectations
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			GetByName(mock.Anything, "test-key").
			Return(existingKey, nil).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(activeKek, cryptoDomain.AESGCM).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(key *tokenizationDomain.TokenizationKey) bool {
				return key.Name == "test-key" &&
					key.FormatType == tokenizationDomain.FormatNumeric &&
					key.Version == 2 && // Version incremented
					key.IsDeterministic == true &&
					key.DekID == dek.ID
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Rotate(ctx, "test-key", tokenizationDomain.FormatNumeric, true, cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "test-key", key.Name)
		assert.Equal(t, uint(2), key.Version)
	})

	t.Run("Success_CreateFirstKeyWhenNoneExist", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dek := cryptoDomain.Dek{
			ID:           uuid.Must(uuid.NewV7()),
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		// Setup expectations
		mockTxManager.EXPECT().
			WithTx(ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(ctx context.Context, fn func(context.Context) error) {
				// Execute the transaction function
				_ = fn(ctx)
			}).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			GetByName(mock.Anything, "new-key").
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		// Expectations for Create() call within transaction
		mockTokenizationKeyRepo.EXPECT().
			GetByNameAndVersion(mock.Anything, "new-key", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		mockKeyManager.EXPECT().
			CreateDek(activeKek, cryptoDomain.AESGCM).
			Return(dek, nil).
			Once()

		mockDekRepo.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(key *tokenizationDomain.TokenizationKey) bool {
				return key.Name == "new-key" &&
					key.FormatType == tokenizationDomain.FormatUUID &&
					key.Version == 1
			})).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		key, err := uc.Rotate(ctx, "new-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "new-key", key.Name)
		assert.Equal(t, uint(1), key.Version)
	})
}

// TestTokenizationKeyUseCase_Delete tests the Delete method.
func TestTokenizationKeyUseCase_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_DeleteKey", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		keyID := uuid.Must(uuid.NewV7())

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			Delete(ctx, keyID).
			Return(nil).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		err := uc.Delete(ctx, keyID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_DeleteFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		keyID := uuid.Must(uuid.NewV7())
		expectedError := errors.New("database error")

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			Delete(ctx, keyID).
			Return(expectedError).
			Once()

		// Execute
		uc := NewTokenizationKeyUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockDekRepo,
			mockKeyManager,
			kekChain,
		)
		err := uc.Delete(ctx, keyID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}
