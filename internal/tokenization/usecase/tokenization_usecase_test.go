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
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationMocks "github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

// TestTokenizationUseCase_Tokenize tests the Tokenize method.
func TestTokenizationUseCase_Tokenize(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_NonDeterministicMode", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dekID := uuid.Must(uuid.NewV7())
		tokenizationKeyID := uuid.Must(uuid.NewV7())

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              tokenizationKeyID,
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			Version:         1,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		dekKey := make([]byte, 32)
		plaintext := []byte("test-value")
		ciphertext := []byte("encrypted-value")
		nonce := []byte("test-nonce")
		metadata := map[string]any{"last4": "alue"}
		expiresAt := time.Now().UTC().Add(24 * time.Hour)

		// Create mock cipher
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, activeKek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Encrypt(plaintext, mock.Anything).
			Return(ciphertext, nonce, nil).
			Once()

		mockTokenRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(token *tokenizationDomain.Token) bool {
				return token.TokenizationKeyID == tokenizationKeyID &&
					len(token.Token) > 0 &&
					token.ValueHash == nil &&
					string(token.Ciphertext) == string(ciphertext) &&
					string(token.Nonce) == string(nonce) &&
					token.ExpiresAt.Equal(expiresAt)
			})).
			Return(nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", plaintext, metadata, &expiresAt)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, tokenizationKeyID, token.TokenizationKeyID)
		assert.NotEmpty(t, token.Token)
		assert.Nil(t, token.ValueHash)
		assert.Equal(t, ciphertext, token.Ciphertext)
		assert.Equal(t, nonce, token.Nonce)
		assert.Equal(t, metadata, token.Metadata)
		assert.Equal(t, expiresAt, *token.ExpiresAt)
	})

	t.Run("Success_DeterministicMode_NewToken", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dekID := uuid.Must(uuid.NewV7())
		tokenizationKeyID := uuid.Must(uuid.NewV7())

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              tokenizationKeyID,
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatLuhnPreserving,
			IsDeterministic: true,
			Version:         1,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		dekKey := make([]byte, 32)
		plaintext := []byte("4111111111111111")
		valueHash := "hash-of-plaintext"
		ciphertext := []byte("encrypted-value")
		nonce := []byte("test-nonce")

		// Create mock cipher
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockHashService.EXPECT().
			Hash(plaintext).
			Return(valueHash).
			Once()

		mockTokenRepo.EXPECT().
			GetByValueHash(ctx, tokenizationKeyID, valueHash).
			Return(nil, tokenizationDomain.ErrTokenNotFound).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, activeKek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Encrypt(plaintext, mock.Anything).
			Return(ciphertext, nonce, nil).
			Once()

		mockHashService.EXPECT().
			Hash(plaintext).
			Return(valueHash).
			Once()

		mockTokenRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(token *tokenizationDomain.Token) bool {
				return token.TokenizationKeyID == tokenizationKeyID &&
					len(token.Token) > 0 &&
					token.ValueHash != nil &&
					*token.ValueHash == valueHash &&
					string(token.Ciphertext) == string(ciphertext) &&
					string(token.Nonce) == string(nonce)
			})).
			Return(nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", plaintext, nil, nil)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, tokenizationKeyID, token.TokenizationKeyID)
		assert.NotEmpty(t, token.Token)
		assert.NotNil(t, token.ValueHash)
		assert.Equal(t, valueHash, *token.ValueHash)
		assert.Equal(t, ciphertext, token.Ciphertext)
		assert.Equal(t, nonce, token.Nonce)
	})

	t.Run("Success_DeterministicMode_ExistingValidToken", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		tokenizationKeyID := uuid.Must(uuid.NewV7())
		plaintext := []byte("test-value")
		valueHash := "hash-of-plaintext"
		existingTokenValue := "existing-token-123"

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              tokenizationKeyID,
			DekID:           uuid.Must(uuid.NewV7()),
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: true,
			Version:         1,
		}

		existingToken := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: tokenizationKeyID,
			Token:             existingTokenValue,
			ValueHash:         &valueHash,
			Ciphertext:        []byte("existing-ciphertext"),
			Nonce:             []byte("existing-nonce"),
			CreatedAt:         time.Now().UTC().Add(-1 * time.Hour),
			ExpiresAt:         nil, // No expiration
			RevokedAt:         nil, // Not revoked
		}

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockHashService.EXPECT().
			Hash(plaintext).
			Return(valueHash).
			Once()

		mockTokenRepo.EXPECT().
			GetByValueHash(ctx, tokenizationKeyID, valueHash).
			Return(existingToken, nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", plaintext, nil, nil)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, existingToken.ID, token.ID)
		assert.Equal(t, existingTokenValue, token.Token)
		assert.Equal(t, valueHash, *token.ValueHash)
	})

	t.Run("Success_DeterministicMode_ExpiredTokenCreatesNew", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dekID := uuid.Must(uuid.NewV7())
		tokenizationKeyID := uuid.Must(uuid.NewV7())
		plaintext := []byte("test-value")
		valueHash := "hash-of-plaintext"
		expiredTime := time.Now().UTC().Add(-1 * time.Hour)

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              tokenizationKeyID,
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: true,
			Version:         1,
		}

		expiredToken := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: tokenizationKeyID,
			Token:             "expired-token",
			ValueHash:         &valueHash,
			Ciphertext:        []byte("old-ciphertext"),
			Nonce:             []byte("old-nonce"),
			CreatedAt:         time.Now().UTC().Add(-2 * time.Hour),
			ExpiresAt:         &expiredTime, // Expired
			RevokedAt:         nil,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		dekKey := make([]byte, 32)
		ciphertext := []byte("new-encrypted-value")
		nonce := []byte("new-nonce")

		// Create mock cipher
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockHashService.EXPECT().
			Hash(plaintext).
			Return(valueHash).
			Once()

		mockTokenRepo.EXPECT().
			GetByValueHash(ctx, tokenizationKeyID, valueHash).
			Return(expiredToken, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, activeKek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Encrypt(plaintext, mock.Anything).
			Return(ciphertext, nonce, nil).
			Once()

		mockHashService.EXPECT().
			Hash(plaintext).
			Return(valueHash).
			Once()

		mockTokenRepo.EXPECT().
			Create(ctx, mock.MatchedBy(func(token *tokenizationDomain.Token) bool {
				return token.TokenizationKeyID == tokenizationKeyID &&
					len(token.Token) > 0 &&
					token.ValueHash != nil &&
					*token.ValueHash == valueHash &&
					string(token.Ciphertext) == string(ciphertext)
			})).
			Return(nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", plaintext, nil, nil)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEqual(t, expiredToken.ID, token.ID) // Should be a new token
		assert.NotEqual(t, "expired-token", token.Token)
	})

	t.Run("Error_TokenizationKeyNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "nonexistent-key").
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "nonexistent-key", []byte("test"), nil, nil)

		// Assert
		assert.Nil(t, token)
		assert.Equal(t, tokenizationDomain.ErrTokenizationKeyNotFound, err)
	})

	t.Run("Error_DekNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		dekID := uuid.Must(uuid.NewV7())
		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			Version:         1,
		}

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(nil, cryptoDomain.ErrDekNotFound).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", []byte("test"), nil, nil)

		// Assert
		assert.Nil(t, token)
		assert.Equal(t, cryptoDomain.ErrDekNotFound, err)
	})

	t.Run("Error_KekNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		dekID := uuid.Must(uuid.NewV7())
		nonexistentKekID := uuid.Must(uuid.NewV7()) // KEK not in chain

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			Version:         1,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        nonexistentKekID, // References KEK not in chain
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", []byte("test"), nil, nil)

		// Assert
		assert.Nil(t, token)
		assert.Equal(t, cryptoDomain.ErrKekNotFound, err)
	})

	t.Run("Error_EncryptionFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dekID := uuid.Must(uuid.NewV7())

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			Version:         1,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("nonce"),
		}

		dekKey := make([]byte, 32)
		plaintext := []byte("test-value")

		mockCipher := cryptoServiceMocks.NewMockAEAD(t)
		encryptionError := errors.New("encryption failed")

		// Setup expectations
		mockTokenizationKeyRepo.EXPECT().
			GetByName(ctx, "test-key").
			Return(tokenizationKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, activeKek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Encrypt(plaintext, mock.Anything).
			Return(nil, nil, encryptionError).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		token, err := uc.Tokenize(ctx, "test-key", plaintext, nil, nil)

		// Assert
		assert.Nil(t, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encrypt plaintext")
	})
}

// TestTokenizationUseCase_Detokenize tests the Detokenize method.
func TestTokenizationUseCase_Detokenize(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_DetokenizeValid", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		// Create test data
		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dekID := uuid.Must(uuid.NewV7())
		tokenizationKeyID := uuid.Must(uuid.NewV7())
		tokenValue := "test-token-123"
		plaintext := []byte("original-value")
		ciphertext := []byte("encrypted-value")
		nonce := []byte("test-nonce")
		metadata := map[string]any{"last4": "alue"}

		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: tokenizationKeyID,
			Token:             tokenValue,
			Ciphertext:        ciphertext,
			Nonce:             nonce,
			Metadata:          metadata,
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              tokenizationKeyID,
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			Version:         1,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
		}

		dekKey := make([]byte, 32)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Get(ctx, tokenizationKeyID).
			Return(tokenizationKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, activeKek).
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

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		resultPlaintext, resultMetadata, err := uc.Detokenize(ctx, tokenValue)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, plaintext, resultPlaintext)
		assert.Equal(t, metadata, resultMetadata)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, "nonexistent-token").
			Return(nil, tokenizationDomain.ErrTokenNotFound).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		plaintext, metadata, err := uc.Detokenize(ctx, "nonexistent-token")

		// Assert
		assert.Nil(t, plaintext)
		assert.Nil(t, metadata)
		assert.Equal(t, tokenizationDomain.ErrTokenNotFound, err)
	})

	t.Run("Error_TokenExpired", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		expiredTime := time.Now().UTC().Add(-1 * time.Hour)
		tokenValue := "expired-token"

		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             tokenValue,
			Ciphertext:        []byte("ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC().Add(-2 * time.Hour),
			ExpiresAt:         &expiredTime,
			RevokedAt:         nil,
		}

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		plaintext, metadata, err := uc.Detokenize(ctx, tokenValue)

		// Assert
		assert.Nil(t, plaintext)
		assert.Nil(t, metadata)
		assert.Equal(t, tokenizationDomain.ErrTokenExpired, err)
	})

	t.Run("Error_TokenRevoked", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		revokedTime := time.Now().UTC().Add(-30 * time.Minute)
		tokenValue := "revoked-token"

		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             tokenValue,
			Ciphertext:        []byte("ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC().Add(-1 * time.Hour),
			ExpiresAt:         nil,
			RevokedAt:         &revokedTime,
		}

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		plaintext, metadata, err := uc.Detokenize(ctx, tokenValue)

		// Assert
		assert.Nil(t, plaintext)
		assert.Nil(t, metadata)
		assert.Equal(t, tokenizationDomain.ErrTokenRevoked, err)
	})

	t.Run("Error_DecryptionFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		activeKek := getActiveKek(kekChain)
		dekID := uuid.Must(uuid.NewV7())
		tokenizationKeyID := uuid.Must(uuid.NewV7())
		tokenValue := "test-token"

		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: tokenizationKeyID,
			Token:             tokenValue,
			Ciphertext:        []byte("corrupted-ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		tokenizationKey := &tokenizationDomain.TokenizationKey{
			ID:              tokenizationKeyID,
			DekID:           dekID,
			Name:            "test-key",
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			Version:         1,
		}

		dek := &cryptoDomain.Dek{
			ID:           dekID,
			KekID:        activeKek.ID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-dek"),
			Nonce:        []byte("dek-nonce"),
		}

		dekKey := make([]byte, 32)
		mockCipher := cryptoServiceMocks.NewMockAEAD(t)
		decryptionError := errors.New("decryption failed")

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		mockTokenizationKeyRepo.EXPECT().
			Get(ctx, tokenizationKeyID).
			Return(tokenizationKey, nil).
			Once()

		mockDekRepo.EXPECT().
			Get(ctx, dekID).
			Return(dek, nil).
			Once()

		mockKeyManager.EXPECT().
			DecryptDek(dek, activeKek).
			Return(dekKey, nil).
			Once()

		mockAEADManager.EXPECT().
			CreateCipher(dekKey, cryptoDomain.AESGCM).
			Return(mockCipher, nil).
			Once()

		mockCipher.EXPECT().
			Decrypt(tokenRecord.Ciphertext, tokenRecord.Nonce, mock.Anything).
			Return(nil, decryptionError).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		plaintext, metadata, err := uc.Detokenize(ctx, tokenValue)

		// Assert
		assert.Nil(t, plaintext)
		assert.Nil(t, metadata)
		assert.Equal(t, cryptoDomain.ErrDecryptionFailed, err)
	})
}

// TestTokenizationUseCase_Validate tests the Validate method.
func TestTokenizationUseCase_Validate(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_ValidToken", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		tokenValue := "valid-token"
		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             tokenValue,
			Ciphertext:        []byte("ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		isValid, err := uc.Validate(ctx, tokenValue)

		// Assert
		assert.NoError(t, err)
		assert.True(t, isValid)
	})

	t.Run("Success_ExpiredToken", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		expiredTime := time.Now().UTC().Add(-1 * time.Hour)
		tokenValue := "expired-token"
		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             tokenValue,
			Ciphertext:        []byte("ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC().Add(-2 * time.Hour),
			ExpiresAt:         &expiredTime,
			RevokedAt:         nil,
		}

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		isValid, err := uc.Validate(ctx, tokenValue)

		// Assert
		assert.NoError(t, err)
		assert.False(t, isValid)
	})

	t.Run("Success_TokenNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, "nonexistent-token").
			Return(nil, tokenizationDomain.ErrTokenNotFound).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		isValid, err := uc.Validate(ctx, "nonexistent-token")

		// Assert
		assert.NoError(t, err)
		assert.False(t, isValid)
	})

	t.Run("Error_RepositoryError", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		dbError := errors.New("database error")

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, "test-token").
			Return(nil, dbError).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		isValid, err := uc.Validate(ctx, "test-token")

		// Assert
		assert.False(t, isValid)
		assert.Equal(t, dbError, err)
	})
}

// TestTokenizationUseCase_Revoke tests the Revoke method.
func TestTokenizationUseCase_Revoke(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_RevokeToken", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		tokenValue := "token-to-revoke"
		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             tokenValue,
			Ciphertext:        []byte("ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		mockTokenRepo.EXPECT().
			Revoke(ctx, tokenValue).
			Return(nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		err := uc.Revoke(ctx, tokenValue)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Error_TokenNotFound", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, "nonexistent-token").
			Return(nil, tokenizationDomain.ErrTokenNotFound).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		err := uc.Revoke(ctx, "nonexistent-token")

		// Assert
		assert.Equal(t, tokenizationDomain.ErrTokenNotFound, err)
	})

	t.Run("Error_RevokeFails", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		tokenValue := "test-token"
		tokenRecord := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             tokenValue,
			Ciphertext:        []byte("ciphertext"),
			Nonce:             []byte("nonce"),
			CreatedAt:         time.Now().UTC(),
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		dbError := errors.New("database error")

		// Setup expectations
		mockTokenRepo.EXPECT().
			GetByToken(ctx, tokenValue).
			Return(tokenRecord, nil).
			Once()

		mockTokenRepo.EXPECT().
			Revoke(ctx, tokenValue).
			Return(dbError).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		err := uc.Revoke(ctx, tokenValue)

		// Assert
		assert.Equal(t, dbError, err)
	})
}

// TestTokenizationUseCase_CleanupExpired tests the CleanupExpired method.
func TestTokenizationUseCase_CleanupExpired(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_DryRunMode", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Setup expectations
		mockTokenRepo.EXPECT().
			CountExpired(ctx, mock.MatchedBy(func(cutoff time.Time) bool {
				// Verify cutoff is approximately 7 days ago
				expectedCutoff := time.Now().UTC().AddDate(0, 0, -7)
				// Allow 2 second variance for test execution time
				return cutoff.After(expectedCutoff.Add(-2*time.Second)) &&
					cutoff.Before(expectedCutoff.Add(2*time.Second))
			})).
			Return(int64(42), nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		count, err := uc.CleanupExpired(ctx, 7, true)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, int64(42), count)
	})

	t.Run("Success_DeleteMode", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Setup expectations
		mockTokenRepo.EXPECT().
			DeleteExpired(ctx, mock.MatchedBy(func(cutoff time.Time) bool {
				// Verify cutoff is approximately 30 days ago
				expectedCutoff := time.Now().UTC().AddDate(0, 0, -30)
				// Allow 2 second variance for test execution time
				return cutoff.After(expectedCutoff.Add(-2*time.Second)) &&
					cutoff.Before(expectedCutoff.Add(2*time.Second))
			})).
			Return(int64(100), nil).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		count, err := uc.CleanupExpired(ctx, 30, false)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, int64(100), count)
	})

	t.Run("Error_NegativeDays", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		count, err := uc.CleanupExpired(ctx, -1, false)

		// Assert
		assert.Equal(t, int64(0), count)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "days must be non-negative")
	})

	t.Run("Error_RepositoryError", func(t *testing.T) {
		// Setup mocks
		mockTxManager := databaseMocks.NewMockTxManager(t)
		mockTokenizationKeyRepo := tokenizationMocks.NewMockTokenizationKeyRepository(t)
		mockTokenRepo := tokenizationMocks.NewMockTokenRepository(t)
		mockDekRepo := tokenizationMocks.NewMockDekRepository(t)
		mockAEADManager := cryptoServiceMocks.NewMockAEADManager(t)
		mockKeyManager := cryptoServiceMocks.NewMockKeyManager(t)
		mockHashService := tokenizationMocks.NewMockHashService(t)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: make([]byte, 32),
		}
		kekChain := createKekChain(masterKey)
		defer kekChain.Close()

		dbError := errors.New("database error")

		// Setup expectations
		mockTokenRepo.EXPECT().
			DeleteExpired(ctx, mock.AnythingOfType("time.Time")).
			Return(int64(0), dbError).
			Once()

		// Create use case
		uc := NewTokenizationUseCase(
			mockTxManager,
			mockTokenizationKeyRepo,
			mockTokenRepo,
			mockDekRepo,
			mockAEADManager,
			mockKeyManager,
			mockHashService,
			kekChain,
		)

		// Execute
		count, err := uc.CleanupExpired(ctx, 7, false)

		// Assert
		assert.Equal(t, int64(0), count)
		assert.Equal(t, dbError, err)
	})
}
