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
	tokenizationKey := &tokenizationDomain.TokenizationKey{
		ID:              keyID,
		Name:            name,
		Version:         version,
		FormatType:      formatType,
		IsDeterministic: isDeterministic,
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

// Rotate creates a new version of an existing tokenization key.
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

		// Get active KEK from chain
		activeKek, err := getKek(t.kekChain, t.kekChain.ActiveKekID())
		if err != nil {
			return apperrors.Wrap(err, "failed to get active KEK")
		}

		// Create new DEK encrypted with active KEK
		dek, err := t.keyManager.CreateDek(activeKek, alg)
		if err != nil {
			return apperrors.Wrap(err, "failed to create DEK")
		}

		// Persist new DEK
		if err := t.dekRepo.Create(txCtx, &dek); err != nil {
			return apperrors.Wrap(err, "failed to persist DEK")
		}

		// Create new tokenization key with incremented version
		keyID, err := uuid.NewV7()
		if err != nil {
			return apperrors.Wrap(err, "failed to generate UUID for tokenization key")
		}
		newKey = &tokenizationDomain.TokenizationKey{
			ID:              keyID,
			Name:            name,
			Version:         currentKey.Version + 1,
			FormatType:      formatType,
			IsDeterministic: isDeterministic,
			DekID:           dek.ID,
			CreatedAt:       time.Now().UTC(),
		}

		// Validate tokenization key fields
		if err := newKey.Validate(); err != nil {
			return apperrors.Wrap(err, "tokenization key validation failed")
		}

		// Persist new tokenization key
		if err := t.tokenizationKeyRepo.Create(txCtx, newKey); err != nil {
			return apperrors.Wrap(err, "failed to persist rotated tokenization key")
		}
		return nil
	})

	if err != nil {
		return nil, apperrors.Wrap(err, "failed to rotate tokenization key")
	}

	return newKey, nil
}

// Delete soft-deletes a tokenization key by setting its deleted_at timestamp.
func (t *tokenizationKeyUseCase) Delete(ctx context.Context, keyID uuid.UUID) error {
	err := t.tokenizationKeyRepo.Delete(ctx, keyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete tokenization key")
	}
	return nil
}

// List retrieves tokenization keys ordered by name ascending with pagination.
func (t *tokenizationKeyUseCase) List(
	ctx context.Context,
	offset, limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	keys, err := t.tokenizationKeyRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list tokenization keys")
	}
	return keys, nil
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
