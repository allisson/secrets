// Package usecase implements tokenization business logic.
//
// Coordinates token generation, encryption, and lifecycle management with configurable
// deterministic behavior. Uses TxManager for transactional consistency.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationService "github.com/allisson/secrets/internal/tokenization/service"
)

// tokenizationUseCase implements TokenizationUseCase for managing tokenization operations.
type tokenizationUseCase struct {
	txManager        database.TxManager
	tokenizationRepo TokenizationKeyRepository
	tokenRepo        TokenRepository
	dekRepo          DekRepository
	aeadManager      cryptoService.AEADManager
	keyManager       cryptoService.KeyManager
	hashService      HashService
	kekChain         *cryptoDomain.KekChain
}

// getKek retrieves a KEK from the chain by its ID.
func (t *tokenizationUseCase) getKek(kekID uuid.UUID) (*cryptoDomain.Kek, error) {
	kek, ok := t.kekChain.Get(kekID)
	if !ok {
		return nil, cryptoDomain.ErrKekNotFound
	}
	return kek, nil
}

// Tokenize generates a token for the given plaintext value using the latest version of the named key.
// In deterministic mode, returns the existing token if the value has been tokenized before.
// Metadata is optional display data (e.g., last 4 digits) stored unencrypted.
func (t *tokenizationUseCase) Tokenize(
	ctx context.Context,
	keyName string,
	plaintext []byte,
	metadata map[string]any,
	expiresAt *time.Time,
) (*tokenizationDomain.Token, error) {
	// Get latest tokenization key version
	tokenizationKey, err := t.tokenizationRepo.GetByName(ctx, keyName)
	if err != nil {
		return nil, err
	}

	// In deterministic mode, check if token already exists for this value
	if tokenizationKey.IsDeterministic {
		valueHash := t.hashService.Hash(plaintext)
		existingToken, err := t.tokenRepo.GetByValueHash(ctx, tokenizationKey.ID, valueHash)
		if err != nil && !apperrors.Is(err, tokenizationDomain.ErrTokenNotFound) {
			return nil, err
		}
		if existingToken != nil {
			// Return existing valid token
			if existingToken.IsValid() {
				return existingToken, nil
			}
			// Existing token is expired or revoked - proceed to create new token
		}
	}

	// Get DEK by tokenization key's DekID
	dek, err := t.dekRepo.Get(ctx, tokenizationKey.DekID)
	if err != nil {
		return nil, err
	}

	// Get KEK for decrypting DEK
	kek, err := t.getKek(dek.KekID)
	if err != nil {
		return nil, err
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}
	defer cryptoDomain.Zero(dekKey)

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Encrypt plaintext
	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to encrypt plaintext")
	}

	// Generate token using appropriate generator
	generator, err := tokenizationService.NewTokenGenerator(tokenizationKey.FormatType)
	if err != nil {
		return nil, err
	}

	// For format-preserving tokens, use plaintext length as hint
	tokenLength := len(plaintext)
	tokenValue, err := generator.Generate(tokenLength)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to generate token")
	}

	// Create token record
	token := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: tokenizationKey.ID,
		Token:             tokenValue,
		ValueHash:         nil,
		Ciphertext:        ciphertext,
		Nonce:             nonce,
		Metadata:          metadata,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         expiresAt,
		RevokedAt:         nil,
	}

	// In deterministic mode, store value hash for lookup
	if tokenizationKey.IsDeterministic {
		valueHash := t.hashService.Hash(plaintext)
		token.ValueHash = &valueHash
	}

	// Persist token
	if err := t.tokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	return token, nil
}

// Detokenize retrieves the original plaintext value for a given token.
// Returns ErrTokenNotFound if token doesn't exist, ErrTokenExpired if expired, ErrTokenRevoked if revoked.
// Security Note: Callers MUST zero the returned plaintext after use: cryptoDomain.Zero(plaintext).
func (t *tokenizationUseCase) Detokenize(
	ctx context.Context,
	token string,
) (plaintext []byte, metadata map[string]any, err error) {
	// Get token record
	tokenRecord, err := t.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	// Validate token is not expired
	if tokenRecord.IsExpired() {
		return nil, nil, tokenizationDomain.ErrTokenExpired
	}

	// Validate token is not revoked
	if tokenRecord.IsRevoked() {
		return nil, nil, tokenizationDomain.ErrTokenRevoked
	}

	// Get tokenization key to retrieve its DekID
	tokenizationKey, err := t.tokenizationRepo.Get(ctx, tokenRecord.TokenizationKeyID)
	if err != nil {
		return nil, nil, err
	}

	// Get DEK
	dek, err := t.dekRepo.Get(ctx, tokenizationKey.DekID)
	if err != nil {
		return nil, nil, err
	}

	// Get KEK for decrypting DEK
	kek, err := t.getKek(dek.KekID)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, nil, err
	}
	defer cryptoDomain.Zero(dekKey)

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt ciphertext with nonce
	plaintext, err = cipher.Decrypt(tokenRecord.Ciphertext, tokenRecord.Nonce, nil)
	if err != nil {
		return nil, nil, cryptoDomain.ErrDecryptionFailed
	}

	return plaintext, tokenRecord.Metadata, nil
}

// Validate checks if a token exists and is valid (not expired or revoked).
func (t *tokenizationUseCase) Validate(ctx context.Context, token string) (bool, error) {
	// Get token record
	tokenRecord, err := t.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		if apperrors.Is(err, tokenizationDomain.ErrTokenNotFound) {
			return false, nil
		}
		return false, err
	}

	// Check if token is valid
	return tokenRecord.IsValid(), nil
}

// Revoke marks a token as revoked, preventing further detokenization.
func (t *tokenizationUseCase) Revoke(ctx context.Context, token string) error {
	// Verify token exists first
	_, err := t.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	// Revoke the token
	return t.tokenRepo.Revoke(ctx, token)
}

// CleanupExpired deletes tokens that expired more than the specified number of days ago.
// Returns the number of deleted tokens. Use dryRun=true to preview count without deletion.
func (t *tokenizationUseCase) CleanupExpired(ctx context.Context, days int, dryRun bool) (int64, error) {
	if days < 0 {
		return 0, apperrors.New("days must be non-negative")
	}

	// Calculate the cutoff timestamp (days ago from now in UTC)
	cutoff := time.Now().UTC().AddDate(0, 0, -days)

	if dryRun {
		// In dry run mode, count expired tokens without deleting
		return t.tokenRepo.CountExpired(ctx, cutoff)
	}

	// Delete expired tokens
	return t.tokenRepo.DeleteExpired(ctx, cutoff)
}

// NewTokenizationUseCase creates a new TokenizationUseCase with injected dependencies.
func NewTokenizationUseCase(
	txManager database.TxManager,
	tokenizationRepo TokenizationKeyRepository,
	tokenRepo TokenRepository,
	dekRepo DekRepository,
	aeadManager cryptoService.AEADManager,
	keyManager cryptoService.KeyManager,
	hashService HashService,
	kekChain *cryptoDomain.KekChain,
) TokenizationUseCase {
	return &tokenizationUseCase{
		txManager:        txManager,
		tokenizationRepo: tokenizationRepo,
		tokenRepo:        tokenRepo,
		dekRepo:          dekRepo,
		aeadManager:      aeadManager,
		keyManager:       keyManager,
		hashService:      hashService,
		kekChain:         kekChain,
	}
}
