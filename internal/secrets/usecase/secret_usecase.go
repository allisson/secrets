// Package usecase implements business logic orchestration for secret management.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	"github.com/allisson/secrets/internal/keyring"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// secretUseCase implements the SecretUseCase interface for managing secrets.
type secretUseCase struct {
	txManager            database.TxManager
	keyring              keyring.Keyring
	secretRepo           SecretRepository
	secretValueSizeLimit int
}

// CreateOrUpdate creates a new secret or creates a new version of an existing secret.
func (s *secretUseCase) CreateOrUpdate(
	ctx context.Context,
	path string,
	value []byte,
) (*secretsDomain.Secret, error) {
	if err := validateSecretPath(path); err != nil {
		return nil, err
	}

	if len(value) > s.secretValueSizeLimit {
		return nil, secretsDomain.ErrSecretValueTooLarge
	}

	var newSecret *secretsDomain.Secret
	err := s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Version lookup happens inside the transaction to avoid races.
		var version uint = 1
		existingSecret, err := s.secretRepo.GetByPath(txCtx, path)
		if err != nil && !errors.Is(err, secretsDomain.ErrSecretNotFound) {
			return err
		}
		if existingSecret != nil {
			version = existingSecret.Version + 1
		}

		env, err := s.keyring.Encrypt(txCtx, value)
		if err != nil {
			return err
		}

		newSecret = &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      env.DekID,
			Ciphertext: env.Ciphertext,
			Nonce:      env.Nonce,
			CreatedAt:  time.Now().UTC(),
		}
		return s.secretRepo.Create(txCtx, newSecret)
	})
	if err != nil {
		return nil, err
	}

	return newSecret, nil
}

// Get retrieves and decrypts a secret by its path (latest version).
func (s *secretUseCase) Get(ctx context.Context, path string) (*secretsDomain.Secret, error) {
	secret, err := s.secretRepo.GetByPath(ctx, path)
	if err != nil {
		return nil, err
	}
	return s.decryptSecret(ctx, secret)
}

// GetByVersion retrieves and decrypts a secret by its path and specific version.
func (s *secretUseCase) GetByVersion(
	ctx context.Context,
	path string,
	version uint,
) (*secretsDomain.Secret, error) {
	secret, err := s.secretRepo.GetByPathAndVersion(ctx, path, version)
	if err != nil {
		return nil, err
	}
	return s.decryptSecret(ctx, secret)
}

func (s *secretUseCase) decryptSecret(
	ctx context.Context,
	secret *secretsDomain.Secret,
) (*secretsDomain.Secret, error) {
	plaintext, err := s.keyring.Decrypt(ctx, keyring.Envelope{
		DekID:      secret.DekID,
		Ciphertext: secret.Ciphertext,
		Nonce:      secret.Nonce,
	})
	if err != nil {
		return nil, keyring.ErrDecryptionFailed
	}

	secret.Plaintext = plaintext
	return secret, nil
}

// Delete performs a soft delete on all versions of a secret by its path.
func (s *secretUseCase) Delete(ctx context.Context, path string) error {
	return s.secretRepo.Delete(ctx, path)
}

// ListCursor retrieves secrets without their values, ordered by path with cursor pagination.
func (s *secretUseCase) ListCursor(
	ctx context.Context,
	afterPath *string,
	limit int,
) ([]*secretsDomain.Secret, error) {
	return s.secretRepo.ListCursor(ctx, afterPath, limit)
}

// PurgeDeleted permanently removes soft-deleted secrets older than specified days.
func (s *secretUseCase) PurgeDeleted(ctx context.Context, olderThanDays int, dryRun bool) (int64, error) {
	if olderThanDays < 0 {
		return 0, errors.New("olderThanDays must be non-negative")
	}

	olderThan := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	return s.secretRepo.HardDelete(ctx, olderThan, dryRun)
}

// NewSecretUseCase creates a new secret use case backed by a Keyring.
func NewSecretUseCase(
	txManager database.TxManager,
	kr keyring.Keyring,
	secretRepo SecretRepository,
	secretValueSizeLimit int,
) SecretUseCase {
	return &secretUseCase{
		txManager:            txManager,
		keyring:              kr,
		secretRepo:           secretRepo,
		secretValueSizeLimit: secretValueSizeLimit,
	}
}
