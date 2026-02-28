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

// validateTokenLength checks if the plaintext length is valid for the token format type.
func validateTokenLength(formatType tokenizationDomain.FormatType, length int) error {
	// UUID format ignores length parameter
	if formatType == tokenizationDomain.FormatUUID {
		return nil
	}

	// Luhn format requires at least 2 characters
	if formatType == tokenizationDomain.FormatLuhnPreserving &&
		length < tokenizationDomain.MinLuhnTokenLength {
		return tokenizationDomain.ErrTokenLengthInvalid
	}

	// All format-preserving tokens have max length constraint
	if length > tokenizationDomain.MaxTokenLength {
		return tokenizationDomain.ErrTokenLengthInvalid
	}

	// Minimum length is 1 for numeric/alphanumeric
	if length < 1 {
		return tokenizationDomain.ErrTokenLengthInvalid
	}

	return nil
}

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

// Tokenize generates a token for the given plaintext value using the latest version of the named key.
// In deterministic mode, returns the existing token if the value has been tokenized before.
// Metadata is optional display data (e.g., last 4 digits) stored unencrypted.
//
// Rate Limiting: Production systems should implement rate limiting on this method to prevent abuse.
// Recommended: 100 requests per minute per user/API key for standard use cases.
// Adjust based on your specific security requirements and usage patterns.
func (t *tokenizationUseCase) Tokenize(
	ctx context.Context,
	keyName string,
	plaintext []byte,
	metadata map[string]any,
	expiresAt *time.Time,
) (*tokenizationDomain.Token, error) {
	// Validate plaintext size
	if len(plaintext) == 0 {
		return nil, tokenizationDomain.ErrPlaintextEmpty
	}
	if len(plaintext) > tokenizationDomain.MaxPlaintextSize {
		return nil, tokenizationDomain.ErrPlaintextTooLarge
	}

	// Get latest tokenization key version
	tokenizationKey, err := t.tokenizationRepo.GetByName(ctx, keyName)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get tokenization key by name")
	}

	// In deterministic mode, check if token already exists for this value
	if tokenizationKey.IsDeterministic {
		valueHash := t.hashService.Hash(plaintext)
		existingToken, err := t.tokenRepo.GetByValueHash(ctx, tokenizationKey.ID, valueHash)
		if err != nil && !apperrors.Is(err, tokenizationDomain.ErrTokenNotFound) {
			return nil, apperrors.Wrap(err, "failed to check existing token in deterministic mode")
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
		return nil, apperrors.Wrap(err, "failed to get DEK")
	}

	// Get KEK for decrypting DEK
	kek, err := getKek(t.kekChain, dek.KekID)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get KEK")
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to decrypt DEK")
	}
	defer cryptoDomain.Zero(dekKey)

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to create cipher")
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

	// Validate token length matches format requirements
	if err := validateTokenLength(tokenizationKey.FormatType, tokenLength); err != nil {
		return nil, err
	}

	tokenValue, err := generator.Generate(tokenLength)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to generate token")
	}

	// Create token record
	tokenID, err := uuid.NewV7()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to generate UUID for token")
	}
	token := &tokenizationDomain.Token{
		ID:                tokenID,
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
		// In deterministic mode, handle race condition where another goroutine
		// created the same token between our check and insert
		if tokenizationKey.IsDeterministic && apperrors.Is(err, apperrors.ErrConflict) {
			// Race detected: another concurrent request inserted this token
			// Query again to get the token that was inserted
			valueHash := t.hashService.Hash(plaintext)
			existingToken, queryErr := t.tokenRepo.GetByValueHash(ctx, tokenizationKey.ID, valueHash)
			if queryErr != nil {
				// If query fails, return original create error
				return nil, apperrors.Wrap(err, "failed to create token")
			}
			// Return the token created by the concurrent request
			return existingToken, nil
		}
		return nil, apperrors.Wrap(err, "failed to create token")
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
		return nil, nil, apperrors.Wrap(err, "failed to get token")
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
		return nil, nil, apperrors.Wrap(err, "failed to get tokenization key")
	}

	// Get DEK
	dek, err := t.dekRepo.Get(ctx, tokenizationKey.DekID)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "failed to get DEK")
	}

	// Get KEK for decrypting DEK
	kek, err := getKek(t.kekChain, dek.KekID)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "failed to get KEK")
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "failed to decrypt DEK")
	}
	defer cryptoDomain.Zero(dekKey)

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "failed to create cipher")
	}

	// Decrypt ciphertext with nonce
	plaintext, err = cipher.Decrypt(tokenRecord.Ciphertext, tokenRecord.Nonce, nil)
	if err != nil {
		return nil, nil, apperrors.Wrap(
			cryptoDomain.ErrDecryptionFailed,
			"failed to decrypt token ciphertext",
		)
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
		return false, apperrors.Wrap(err, "failed to validate token")
	}

	// Check if token is valid
	return tokenRecord.IsValid(), nil
}

// Revoke marks a token as revoked, preventing further detokenization.
func (t *tokenizationUseCase) Revoke(ctx context.Context, token string) error {
	// Verify token exists first
	_, err := t.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		return apperrors.Wrap(err, "failed to get token for revocation")
	}

	// Revoke the token
	err = t.tokenRepo.Revoke(ctx, token)
	if err != nil {
		return apperrors.Wrap(err, "failed to revoke token")
	}
	return nil
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
