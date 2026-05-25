// Package usecase implements tokenization business logic.
//
// Coordinates token generation, encryption, and lifecycle management with configurable
// deterministic behavior. Uses TxManager for transactional consistency and Keyring for
// envelope encryption.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationService "github.com/allisson/secrets/internal/tokenization/service"
)

// validateTokenLength checks if the plaintext length is valid for the token format type.
func validateTokenLength(formatType tokenizationDomain.FormatType, length int) error {
	if formatType == tokenizationDomain.FormatUUID {
		return nil
	}
	if formatType == tokenizationDomain.FormatLuhnPreserving &&
		length < tokenizationDomain.MinLuhnTokenLength {
		return tokenizationDomain.ErrTokenLengthInvalid
	}
	if length > tokenizationDomain.MaxTokenLength {
		return tokenizationDomain.ErrTokenLengthInvalid
	}
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
	hashService      HashService
	keyring          keyring.Keyring
}

// Tokenize generates a token for the given plaintext value using the latest version of the named key.
func (t *tokenizationUseCase) Tokenize(
	ctx context.Context,
	keyName string,
	plaintext []byte,
	metadata map[string]any,
	expiresAt *time.Time,
) (result *tokenizationDomain.Token, err error) {
	if len(plaintext) == 0 {
		return nil, tokenizationDomain.ErrPlaintextEmpty
	}
	if len(plaintext) > tokenizationDomain.MaxPlaintextSize {
		return nil, tokenizationDomain.ErrPlaintextTooLarge
	}

	tokenizationKey, err := t.tokenizationRepo.GetByName(ctx, keyName)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get tokenization key by name")
	}

	// In deterministic mode, look up an existing token before encrypting.
	if tokenizationKey.IsDeterministic {
		valueHash := t.hashService.Hash(plaintext, tokenizationKey.Salt)
		existingToken, err := t.tokenRepo.GetByValueHash(ctx, tokenizationKey.ID, valueHash)
		if err != nil && !apperrors.Is(err, tokenizationDomain.ErrTokenNotFound) {
			return nil, apperrors.Wrap(err, "failed to check existing token in deterministic mode")
		}
		if existingToken != nil && existingToken.IsValid() {
			return existingToken, nil
		}
	}

	handle := keyring.DekHandle{DekID: tokenizationKey.DekID}
	ciphertext, nonce, err := t.keyring.EncryptWith(ctx, handle, plaintext, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to encrypt plaintext")
	}

	generator, err := tokenizationService.NewTokenGenerator(tokenizationKey.FormatType)
	if err != nil {
		return nil, err
	}

	tokenLength := len(plaintext)
	if err := validateTokenLength(tokenizationKey.FormatType, tokenLength); err != nil {
		return nil, err
	}

	tokenValue, err := generator.Generate(tokenLength)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to generate token")
	}

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

	if tokenizationKey.IsDeterministic {
		valueHash := t.hashService.Hash(plaintext, tokenizationKey.Salt)
		token.ValueHash = &valueHash
	}

	if err := t.tokenRepo.Create(ctx, token); err != nil {
		// Race: another goroutine inserted the deterministic token between the
		// existence check and the insert. Re-read and return that one.
		if tokenizationKey.IsDeterministic && apperrors.Is(err, apperrors.ErrConflict) {
			valueHash := t.hashService.Hash(plaintext, tokenizationKey.Salt)
			existingToken, queryErr := t.tokenRepo.GetByValueHash(ctx, tokenizationKey.ID, valueHash)
			if queryErr != nil {
				return nil, apperrors.Wrap(err, "failed to create token")
			}
			if !existingToken.IsValid() {
				return nil, apperrors.Wrap(err, "concurrently created token is invalid or expired")
			}
			return existingToken, nil
		}
		return nil, apperrors.Wrap(err, "failed to create token")
	}

	return token, nil
}

// TokenizeBatch generates tokens for multiple plaintext values, wrapped in a transaction.
func (t *tokenizationUseCase) TokenizeBatch(
	ctx context.Context,
	keyName string,
	plaintexts [][]byte,
	metadatas []map[string]any,
	expiresAt *time.Time,
) (result []*tokenizationDomain.Token, err error) {
	var tokens []*tokenizationDomain.Token
	err = t.txManager.WithTx(ctx, func(ctx context.Context) error {
		for i, plaintext := range plaintexts {
			var metadata map[string]any
			if i < len(metadatas) {
				metadata = metadatas[i]
			}
			token, err := t.Tokenize(ctx, keyName, plaintext, metadata, expiresAt)
			if err != nil {
				return err
			}
			tokens = append(tokens, token)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// Detokenize retrieves the original plaintext value for a given token.
// Security: callers MUST zero the returned plaintext after use.
func (t *tokenizationUseCase) Detokenize(
	ctx context.Context,
	token string,
) (plaintext []byte, metadata map[string]any, err error) {
	tokenRecord, err := t.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "failed to get token")
	}

	if tokenRecord.IsExpired() {
		return nil, nil, tokenizationDomain.ErrTokenExpired
	}
	if tokenRecord.IsRevoked() {
		return nil, nil, tokenizationDomain.ErrTokenRevoked
	}

	tokenizationKey, err := t.tokenizationRepo.Get(ctx, tokenRecord.TokenizationKeyID)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "failed to get tokenization key")
	}

	handle := keyring.DekHandle{DekID: tokenizationKey.DekID}
	plaintext, err = t.keyring.DecryptWith(ctx, handle, tokenRecord.Ciphertext, tokenRecord.Nonce, nil)
	if err != nil {
		return nil, nil, apperrors.Wrap(
			keyring.ErrDecryptionFailed,
			"failed to decrypt token ciphertext",
		)
	}

	return plaintext, tokenRecord.Metadata, nil
}

// DetokenizeBatch retrieves original plaintext values for multiple tokens, wrapped in a transaction.
func (t *tokenizationUseCase) DetokenizeBatch(
	ctx context.Context,
	tokens []string,
) (plaintexts [][]byte, metadatas []map[string]any, err error) {
	err = t.txManager.WithTx(ctx, func(ctx context.Context) error {
		for _, token := range tokens {
			plaintext, metadata, err := t.Detokenize(ctx, token)
			if err != nil {
				return err
			}
			plaintexts = append(plaintexts, plaintext)
			metadatas = append(metadatas, metadata)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return plaintexts, metadatas, nil
}

// Validate checks if a token exists and is valid (not expired or revoked).
func (t *tokenizationUseCase) Validate(ctx context.Context, token string) (valid bool, err error) {
	tokenRecord, err := t.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		if apperrors.Is(err, tokenizationDomain.ErrTokenNotFound) {
			return false, nil
		}
		return false, apperrors.Wrap(err, "failed to validate token")
	}
	return tokenRecord.IsValid(), nil
}

// Revoke marks a token as revoked, preventing further detokenization.
func (t *tokenizationUseCase) Revoke(ctx context.Context, token string) (err error) {
	if _, err = t.tokenRepo.GetByToken(ctx, token); err != nil {
		return apperrors.Wrap(err, "failed to get token for revocation")
	}
	if err = t.tokenRepo.Revoke(ctx, token); err != nil {
		return apperrors.Wrap(err, "failed to revoke token")
	}
	return nil
}

// CleanupExpired deletes tokens that expired more than the specified number of days ago.
func (t *tokenizationUseCase) CleanupExpired(
	ctx context.Context,
	days int,
	dryRun bool,
) (count int64, err error) {
	if days < 0 {
		return 0, apperrors.New("days must be non-negative")
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	if dryRun {
		count, err = t.tokenRepo.CountExpired(ctx, cutoff)
		return
	}
	count, err = t.tokenRepo.DeleteExpired(ctx, cutoff)
	return
}

// NewTokenizationUseCase creates a new TokenizationUseCase backed by a Keyring.
func NewTokenizationUseCase(
	txManager database.TxManager,
	tokenizationRepo TokenizationKeyRepository,
	tokenRepo TokenRepository,
	hashService HashService,
	kr keyring.Keyring,
) TokenizationUseCase {
	return &tokenizationUseCase{
		txManager:        txManager,
		tokenizationRepo: tokenizationRepo,
		tokenRepo:        tokenRepo,
		hashService:      hashService,
		keyring:          kr,
	}
}
