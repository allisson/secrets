package usecase

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// tokenizationKeyUseCase implements TokenizationKeyUseCase for managing tokenization keys.
type tokenizationKeyUseCase struct {
	txManager           database.TxManager
	tokenizationKeyRepo TokenizationKeyRepository
	dekRepo             DekRepository
	keyManager          cryptoService.KeyManager
	kekChain            *cryptoDomain.KekChain
}

// createTokenizationKey is a helper that creates a tokenization key within an existing transaction context.
// It does NOT create its own transaction - the caller must handle transaction management.
func (t *tokenizationKeyUseCase) createTokenizationKey(
	ctx context.Context,
	name string,
	version uint,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg cryptoDomain.Algorithm,
) (*tokenizationDomain.TokenizationKey, error) {
	// Get active KEK from chain
	activeKek, err := getKek(t.kekChain, t.kekChain.ActiveKekID())
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get active KEK")
	}

	// Create DEK encrypted with active KEK
	dek, err := t.keyManager.CreateDek(activeKek, alg)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to create DEK")
	}

	// Persist DEK to database
	if err := t.dekRepo.Create(ctx, &dek); err != nil {
		return nil, apperrors.Wrap(err, "failed to persist DEK")
	}

	// Create tokenization key
	keyID, err := uuid.NewV7()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to generate UUID for tokenization key")
	}

	// Generate salt for deterministic hashing
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, apperrors.Wrap(err, "failed to generate salt")
	}

	tokenizationKey := &tokenizationDomain.TokenizationKey{
		ID:              keyID,
		Name:            name,
		Version:         version,
		FormatType:      formatType,
		IsDeterministic: isDeterministic,
		Salt:            salt,
		DekID:           dek.ID,
		CreatedAt:       time.Now().UTC(),
	}

	// Validate tokenization key fields
	if err := tokenizationKey.Validate(); err != nil {
		return nil, apperrors.Wrap(err, "tokenization key validation failed")
	}

	// Persist tokenization key
	if err := t.tokenizationKeyRepo.Create(ctx, tokenizationKey); err != nil {
		return nil, apperrors.Wrap(err, "failed to persist tokenization key")
	}

	return tokenizationKey, nil
}

// Create generates and persists a new tokenization key with version 1.
// Returns ErrTokenizationKeyAlreadyExists if a key with the same name already exists.
func (t *tokenizationKeyUseCase) Create(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg cryptoDomain.Algorithm,
) (*tokenizationDomain.TokenizationKey, error) {
	// Validate format type
	if err := formatType.Validate(); err != nil {
		return nil, tokenizationDomain.ErrInvalidFormatType
	}

	// Check if tokenization key with version 1 already exists
	existingKey, err := t.tokenizationKeyRepo.GetByNameAndVersion(ctx, name, 1)
	if err != nil && !apperrors.Is(err, tokenizationDomain.ErrTokenizationKeyNotFound) {
		return nil, apperrors.Wrap(err, "failed to check for existing tokenization key")
	}
	if existingKey != nil {
		return nil, tokenizationDomain.ErrTokenizationKeyAlreadyExists
	}

	var tokenizationKey *tokenizationDomain.TokenizationKey

	// Wrap DEK and tokenization key creation in a transaction
	err = t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		tokenizationKey, err = t.createTokenizationKey(txCtx, name, 1, formatType, isDeterministic, alg)
		return err
	})

	if err != nil {
		return nil, apperrors.Wrap(err, "failed to create tokenization key")
	}

	return tokenizationKey, nil
}

// Rotate creates a new version of an existing tokenization key by incrementing the version number.
// Generates a new DEK for the new version while preserving old versions for detokenization.
// If the key doesn't exist, it creates the first version.
func (t *tokenizationKeyUseCase) Rotate(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg cryptoDomain.Algorithm,
) (*tokenizationDomain.TokenizationKey, error) {
	// Validate format type
	if err := formatType.Validate(); err != nil {
		return nil, tokenizationDomain.ErrInvalidFormatType
	}

	var newKey *tokenizationDomain.TokenizationKey

	err := t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Get latest tokenization key version
		currentKey, err := t.tokenizationKeyRepo.GetByName(txCtx, name)
		if err != nil {
			// If key doesn't exist, create first version
			if apperrors.Is(err, tokenizationDomain.ErrTokenizationKeyNotFound) {
				newKey, err = t.createTokenizationKey(txCtx, name, 1, formatType, isDeterministic, alg)
				return err
			}
			return apperrors.Wrap(err, "failed to get current tokenization key")
		}

		// Create new tokenization key version using helper
		newKey, err = t.createTokenizationKey(
			txCtx,
			name,
			currentKey.Version+1,
			formatType,
			isDeterministic,
			alg,
		)
		return err
	})

	if err != nil {
		return nil, apperrors.Wrap(err, "failed to rotate tokenization key")
	}

	return newKey, nil
}

// Delete soft-deletes a tokenization key and all its versions by setting its deleted_at timestamp.
func (t *tokenizationKeyUseCase) Delete(ctx context.Context, keyID uuid.UUID) error {
	err := t.tokenizationKeyRepo.Delete(ctx, keyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete tokenization key")
	}
	return nil
}

// GetByName retrieves a single tokenization key by its name.
// Returns the latest version for the key. Filters out soft-deleted keys.
func (t *tokenizationKeyUseCase) GetByName(
	ctx context.Context,
	name string,
) (*tokenizationDomain.TokenizationKey, error) {
	key, err := t.tokenizationKeyRepo.GetByName(ctx, name)
	if err != nil {
		if apperrors.Is(err, tokenizationDomain.ErrTokenizationKeyNotFound) {
			return nil, err
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key")
	}
	return key, nil
}

// ListCursor retrieves tokenization keys ordered by name ascending with cursor-based pagination.
// Returns the latest version for each key name.
func (t *tokenizationKeyUseCase) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	keys, err := t.tokenizationKeyRepo.ListCursor(ctx, afterName, limit)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list tokenization keys")
	}
	return keys, nil
}

// PurgeDeleted permanently removes soft-deleted tokenization keys and their associated tokens.
// Only keys deleted longer than olderThanDays ago are affected.
// If dryRun is true, returns the count of items that would be deleted without performing the operation.
func (t *tokenizationKeyUseCase) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (int64, error) {
	if olderThanDays < 0 {
		return 0, apperrors.New("olderThanDays must be a positive number")
	}

	olderThan := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	return t.tokenizationKeyRepo.HardDelete(ctx, olderThan, dryRun)
}

// NewTokenizationKeyUseCase creates a new tokenization key use case instance.
func NewTokenizationKeyUseCase(
	txManager database.TxManager,
	tokenizationKeyRepo TokenizationKeyRepository,
	dekRepo DekRepository,
	keyManager cryptoService.KeyManager,
	kekChain *cryptoDomain.KekChain,
) TokenizationKeyUseCase {
	return &tokenizationKeyUseCase{
		txManager:           txManager,
		tokenizationKeyRepo: tokenizationKeyRepo,
		dekRepo:             dekRepo,
		keyManager:          keyManager,
		kekChain:            kekChain,
	}
}
