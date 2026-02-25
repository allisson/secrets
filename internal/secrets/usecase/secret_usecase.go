// Package usecase implements business logic orchestration for secret management.
// This package coordinates between cryptographic services, repositories, and domain logic
// to implement secure secret storage and retrieval with automatic versioning.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// secretUseCase implements the SecretUseCase interface for managing secrets.
type secretUseCase struct {
	txManager    database.TxManager
	dekRepo      DekRepository
	secretRepo   SecretRepository
	kekChain     *cryptoDomain.KekChain
	aeadManager  cryptoService.AEADManager
	keyManager   cryptoService.KeyManager
	dekAlgorithm cryptoDomain.Algorithm
}

// CreateOrUpdate creates a new secret or creates a new version of an existing secret.
func (s *secretUseCase) CreateOrUpdate(
	ctx context.Context,
	path string,
	value []byte,
) (*secretsDomain.Secret, error) {
	activeKek, found := s.kekChain.Get(s.kekChain.ActiveKekID())
	if !found {
		return nil, cryptoDomain.ErrKekNotFound
	}

	return s.createOrUpdateSecret(ctx, path, value, activeKek)
}

// createOrUpdateSecret is a helper method that handles the secret creation/update logic.
func (s *secretUseCase) createOrUpdateSecret(
	ctx context.Context,
	path string,
	value []byte,
	kek *cryptoDomain.Kek,
) (*secretsDomain.Secret, error) {
	var version uint = 1

	// Check if secret already exists to determine the version
	existingSecret, err := s.secretRepo.GetByPath(ctx, path)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}
	if existingSecret != nil {
		version = existingSecret.Version + 1
	}

	// Execute the creation within a transaction
	var newSecret *secretsDomain.Secret
	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Create a new DEK for this secret
		dek, err := s.keyManager.CreateDek(kek, s.dekAlgorithm)
		if err != nil {
			return err
		}

		// Persist the DEK
		if err := s.dekRepo.Create(txCtx, &dek); err != nil {
			return err
		}

		// Decrypt the DEK first to get the plaintext key
		dekKey, err := s.keyManager.DecryptDek(&dek, kek)
		if err != nil {
			return err
		}
		defer cryptoDomain.Zero(dekKey)

		// Create cipher with the decrypted DEK key
		cipher, err := s.aeadManager.CreateCipher(dekKey, s.dekAlgorithm)
		if err != nil {
			return err
		}

		// Encrypt the secret value
		ciphertext, nonce, err := cipher.Encrypt(value, nil)
		if err != nil {
			return err
		}

		// Create the secret entity
		newSecret = &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      dek.ID,
			Ciphertext: ciphertext,
			Nonce:      nonce,
			CreatedAt:  time.Now().UTC(),
		}

		// Persist the secret
		if err := s.secretRepo.Create(txCtx, newSecret); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newSecret, nil
}

// Get retrieves and decrypts a secret by its path (latest version).
func (s *secretUseCase) Get(ctx context.Context, path string) (*secretsDomain.Secret, error) {
	// Retrieve the secret by path
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
	// Retrieve the secret by path and version
	secret, err := s.secretRepo.GetByPathAndVersion(ctx, path, version)
	if err != nil {
		return nil, err
	}

	return s.decryptSecret(ctx, secret)
}

// decryptSecret is a helper method that decrypts a secret's ciphertext.
func (s *secretUseCase) decryptSecret(
	ctx context.Context,
	secret *secretsDomain.Secret,
) (*secretsDomain.Secret, error) {
	// Retrieve the DEK
	dek, err := s.dekRepo.Get(ctx, secret.DekID)
	if err != nil {
		return nil, err
	}

	// Retrieve the KEK needed to decrypt the DEK
	kek, found := s.kekChain.Get(dek.KekID)
	if !found {
		return nil, cryptoDomain.ErrKekNotFound
	}

	// Decrypt the DEK
	dekKey, err := s.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}
	defer cryptoDomain.Zero(dekKey)

	// Create cipher with the decrypted DEK key
	cipher, err := s.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Decrypt the secret value
	plaintext, err := cipher.Decrypt(secret.Ciphertext, secret.Nonce, nil)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	// Populate the plaintext field
	secret.Plaintext = plaintext

	return secret, nil
}

// Delete performs a soft delete on a secret by its path.
func (s *secretUseCase) Delete(ctx context.Context, path string) error {
	// Retrieve the secret by path to get its ID
	secret, err := s.secretRepo.GetByPath(ctx, path)
	if err != nil {
		return err
	}

	// Perform soft delete
	return s.secretRepo.Delete(ctx, secret.ID)
}

// List retrieves secrets without their values, ordered by path with pagination.
func (s *secretUseCase) List(ctx context.Context, offset, limit int) ([]*secretsDomain.Secret, error) {
	return s.secretRepo.List(ctx, offset, limit)
}

// NewSecretUseCase creates a new secret use case instance with the provided dependencies.
func NewSecretUseCase(
	txManager database.TxManager,
	dekRepo DekRepository,
	secretRepo SecretRepository,
	kekChain *cryptoDomain.KekChain,
	aeadManager cryptoService.AEADManager,
	keyManager cryptoService.KeyManager,
	dekAlgorithm cryptoDomain.Algorithm,
) SecretUseCase {
	return &secretUseCase{
		txManager:    txManager,
		dekRepo:      dekRepo,
		secretRepo:   secretRepo,
		kekChain:     kekChain,
		aeadManager:  aeadManager,
		keyManager:   keyManager,
		dekAlgorithm: dekAlgorithm,
	}
}
