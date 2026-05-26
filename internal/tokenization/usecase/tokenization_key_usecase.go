package usecase

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// tokenizationKeyUseCase implements TokenizationKeyUseCase for managing tokenization keys.
type tokenizationKeyUseCase struct {
	txManager           database.TxManager
	tokenizationKeyRepo tokenizationDomain.TokenizationKeyRepository
	keyring             keyring.Keyring
}

// createTokenizationKey is a helper that creates a tokenization key within an existing
// transaction context. It does NOT create its own transaction; the caller manages it.
func (t *tokenizationKeyUseCase) createTokenizationKey(
	ctx context.Context,
	name string,
	version uint,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg keyring.Algorithm,
) (*tokenizationDomain.TokenizationKey, error) {
	handle, err := t.keyring.AllocateDek(ctx, alg)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to allocate DEK")
	}

	keyID, err := uuid.NewV7()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to generate UUID for tokenization key")
	}

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
		DekID:           handle.DekID,
		CreatedAt:       time.Now().UTC(),
	}

	if err := tokenizationKey.Validate(); err != nil {
		return nil, apperrors.Wrap(err, "tokenization key validation failed")
	}

	if err := t.tokenizationKeyRepo.Create(ctx, tokenizationKey); err != nil {
		return nil, apperrors.Wrap(err, "failed to persist tokenization key")
	}

	return tokenizationKey, nil
}

// Create generates and persists a new tokenization key with version 1.
func (t *tokenizationKeyUseCase) Create(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg keyring.Algorithm,
) (result *tokenizationDomain.TokenizationKey, err error) {
	if err = formatType.Validate(); err != nil {
		return nil, tokenizationDomain.ErrInvalidFormatType
	}

	existingKey, err := t.tokenizationKeyRepo.GetByNameAndVersion(ctx, name, 1)
	if err != nil && !apperrors.Is(err, tokenizationDomain.ErrTokenizationKeyNotFound) {
		return nil, apperrors.Wrap(err, "failed to check for existing tokenization key")
	}
	if existingKey != nil {
		return nil, tokenizationDomain.ErrTokenizationKeyAlreadyExists
	}

	var tokenizationKey *tokenizationDomain.TokenizationKey
	err = t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		tokenizationKey, err = t.createTokenizationKey(txCtx, name, 1, formatType, isDeterministic, alg)
		return err
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to create tokenization key")
	}

	return tokenizationKey, nil
}

// Rotate creates a new version of an existing tokenization key.
// If the key doesn't exist, it creates the first version.
func (t *tokenizationKeyUseCase) Rotate(
	ctx context.Context,
	name string,
	formatType tokenizationDomain.FormatType,
	isDeterministic bool,
	alg keyring.Algorithm,
) (result *tokenizationDomain.TokenizationKey, err error) {
	if err = formatType.Validate(); err != nil {
		return nil, tokenizationDomain.ErrInvalidFormatType
	}

	var newKey *tokenizationDomain.TokenizationKey
	err = t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		currentKey, err := t.tokenizationKeyRepo.GetByName(txCtx, name)
		if err != nil {
			if apperrors.Is(err, tokenizationDomain.ErrTokenizationKeyNotFound) {
				newKey, err = t.createTokenizationKey(txCtx, name, 1, formatType, isDeterministic, alg)
				return err
			}
			return apperrors.Wrap(err, "failed to get current tokenization key")
		}
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

// Delete soft deletes a tokenization key and all its versions by name.
func (t *tokenizationKeyUseCase) Delete(ctx context.Context, name string) (err error) {
	if err = t.tokenizationKeyRepo.Delete(ctx, name); err != nil {
		return apperrors.Wrap(err, "failed to delete tokenization key")
	}
	return nil
}

// GetByName retrieves a single tokenization key by its name.
func (t *tokenizationKeyUseCase) GetByName(
	ctx context.Context,
	name string,
) (result *tokenizationDomain.TokenizationKey, err error) {
	key, err := t.tokenizationKeyRepo.GetByName(ctx, name)
	if err != nil {
		if apperrors.Is(err, tokenizationDomain.ErrTokenizationKeyNotFound) {
			return nil, err
		}
		return nil, apperrors.Wrap(err, "failed to get tokenization key")
	}
	return key, nil
}

// ListCursor retrieves tokenization keys ordered by name ascending.
func (t *tokenizationKeyUseCase) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) (result []*tokenizationDomain.TokenizationKey, err error) {
	keys, err := t.tokenizationKeyRepo.ListCursor(ctx, afterName, limit)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list tokenization keys")
	}
	return keys, nil
}

// PurgeDeleted permanently removes soft-deleted tokenization keys and their associated tokens.
func (t *tokenizationKeyUseCase) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (count int64, err error) {
	if olderThanDays < 0 {
		return 0, apperrors.New("olderThanDays must be a positive number")
	}

	olderThan := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	count, err = t.tokenizationKeyRepo.HardDelete(ctx, olderThan, dryRun)
	return
}

// NewTokenizationKeyUseCase creates a new tokenization key use case instance.
func NewTokenizationKeyUseCase(
	txManager database.TxManager,
	tokenizationKeyRepo tokenizationDomain.TokenizationKeyRepository,
	kr keyring.Keyring,
) TokenizationKeyUseCase {
	return &tokenizationKeyUseCase{
		txManager:           txManager,
		tokenizationKeyRepo: tokenizationKeyRepo,
		keyring:             kr,
	}
}
