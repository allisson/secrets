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

// getKek retrieves a KEK from the chain by its ID.
func (t *tokenizationKeyUseCase) getKek(kekID uuid.UUID) (*cryptoDomain.Kek, error) {
	kek, ok := t.kekChain.Get(kekID)
	if !ok {
		return nil, cryptoDomain.ErrKekNotFound
	}
	return kek, nil
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
		return nil, err
	}
	if existingKey != nil {
		return nil, tokenizationDomain.ErrTokenizationKeyAlreadyExists
	}

	// Get active KEK from chain
	activeKek, err := t.getKek(t.kekChain.ActiveKekID())
	if err != nil {
		return nil, err
	}

	// Create DEK encrypted with active KEK
	dek, err := t.keyManager.CreateDek(activeKek, alg)
	if err != nil {
		return nil, err
	}

	// Persist DEK to database
	if err := t.dekRepo.Create(ctx, &dek); err != nil {
		return nil, err
	}

	// Create tokenization key with version 1
	tokenizationKey := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            name,
		Version:         1,
		FormatType:      formatType,
		IsDeterministic: isDeterministic,
		DekID:           dek.ID,
		CreatedAt:       time.Now().UTC(),
	}

	// Persist tokenization key
	if err := t.tokenizationKeyRepo.Create(ctx, tokenizationKey); err != nil {
		return nil, err
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
				newKey, err = t.Create(txCtx, name, formatType, isDeterministic, alg)
				return err
			}
			return err
		}

		// Get active KEK from chain
		activeKek, err := t.getKek(t.kekChain.ActiveKekID())
		if err != nil {
			return err
		}

		// Create new DEK encrypted with active KEK
		dek, err := t.keyManager.CreateDek(activeKek, alg)
		if err != nil {
			return err
		}

		// Persist new DEK
		if err := t.dekRepo.Create(txCtx, &dek); err != nil {
			return err
		}

		// Create new tokenization key with incremented version
		newKey = &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            name,
			Version:         currentKey.Version + 1,
			FormatType:      formatType,
			IsDeterministic: isDeterministic,
			DekID:           dek.ID,
			CreatedAt:       time.Now().UTC(),
		}

		// Persist new tokenization key
		return t.tokenizationKeyRepo.Create(txCtx, newKey)
	})

	if err != nil {
		return nil, err
	}

	return newKey, nil
}

// Delete soft-deletes a tokenization key by setting its deleted_at timestamp.
func (t *tokenizationKeyUseCase) Delete(ctx context.Context, keyID uuid.UUID) error {
	return t.tokenizationKeyRepo.Delete(ctx, keyID)
}

// List retrieves tokenization keys ordered by name ascending with pagination.
func (t *tokenizationKeyUseCase) List(
	ctx context.Context,
	offset, limit int,
) ([]*tokenizationDomain.TokenizationKey, error) {
	return t.tokenizationKeyRepo.List(ctx, offset, limit)
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
