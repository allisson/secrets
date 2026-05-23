// Package usecase implements transit encryption business logic.
//
// Coordinates between the keyring and the transit repository to manage transit keys
// with versioning and envelope encryption.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// nonceSize is the AEAD nonce length stored alongside ciphertext in the
// transit wire format. Both supported algorithms (AES-256-GCM,
// ChaCha20-Poly1305) use 12-byte nonces. If we add an algorithm with a
// different nonce size, this needs to be derived from the algorithm.
const nonceSize = 12

// transitKeyUseCase implements TransitKeyUseCase for managing transit keys.
type transitKeyUseCase struct {
	txManager   database.TxManager
	transitRepo TransitKeyRepository
	keyring     keyring.Keyring
}

// Create generates and persists a new transit key with version 1.
func (t *transitKeyUseCase) Create(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
	var transitKey *transitDomain.TransitKey

	err := t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		existingKey, err := t.transitRepo.GetByNameAndVersion(txCtx, name, 1)
		if err != nil && !apperrors.Is(err, transitDomain.ErrTransitKeyNotFound) {
			return err
		}
		if existingKey != nil {
			return transitDomain.ErrTransitKeyAlreadyExists
		}

		handle, err := t.keyring.AllocateDek(txCtx, alg)
		if err != nil {
			return err
		}

		transitKey = &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   1,
			DekID:     handle.DekID,
			CreatedAt: time.Now().UTC(),
		}
		return t.transitRepo.Create(txCtx, transitKey)
	})
	if err != nil {
		return nil, err
	}

	return transitKey, nil
}

// Rotate creates a new version of an existing transit key.
func (t *transitKeyUseCase) Rotate(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
	var newTransitKey *transitDomain.TransitKey

	err := t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		currentKey, err := t.transitRepo.GetByName(txCtx, name)
		if err != nil {
			if apperrors.Is(err, transitDomain.ErrTransitKeyNotFound) {
				newTransitKey, err = t.Create(txCtx, name, alg)
				return err
			}
			return err
		}

		handle, err := t.keyring.AllocateDek(txCtx, alg)
		if err != nil {
			return err
		}

		newTransitKey = &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   currentKey.Version + 1,
			DekID:     handle.DekID,
			CreatedAt: time.Now().UTC(),
		}
		return t.transitRepo.Create(txCtx, newTransitKey)
	})
	if err != nil {
		return nil, err
	}

	return newTransitKey, nil
}

// Get retrieves transit key metadata (including its algorithm) by name and optional version.
func (t *transitKeyUseCase) Get(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, cryptoDomain.Algorithm, error) {
	return t.transitRepo.GetTransitKey(ctx, name, version)
}

// Delete soft-deletes all versions of a transit key by name.
func (t *transitKeyUseCase) Delete(ctx context.Context, name string) error {
	return t.transitRepo.Delete(ctx, name)
}

// Encrypt encrypts plaintext using the latest version of a named transit key.
//
// The returned EncryptedBlob.Ciphertext is `nonce || ciphertext`, base64-encoded
// in the wire format `version:base64(...)`. See ADR-0002.
func (t *transitKeyUseCase) Encrypt(
	ctx context.Context,
	name string,
	plaintext, context []byte,
) (*transitDomain.EncryptedBlob, error) {
	transitKey, err := t.transitRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	handle := keyring.DekHandle{DekID: transitKey.DekID}
	ciphertext, nonce, err := t.keyring.EncryptWith(ctx, handle, plaintext, context)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to encrypt plaintext")
	}

	encryptedData := make([]byte, 0, len(nonce)+len(ciphertext))
	encryptedData = append(encryptedData, nonce...)
	encryptedData = append(encryptedData, ciphertext...)

	return &transitDomain.EncryptedBlob{
		Version:    transitKey.Version,
		Ciphertext: encryptedData,
		Plaintext:  nil,
	}, nil
}

// Decrypt decrypts ciphertext using the version specified in the encrypted blob.
func (t *transitKeyUseCase) Decrypt(
	ctx context.Context,
	name string,
	ciphertext string,
	context []byte,
) (*transitDomain.EncryptedBlob, error) {
	blob, err := transitDomain.NewEncryptedBlob(ciphertext)
	if err != nil {
		return nil, err
	}

	transitKey, err := t.transitRepo.GetByNameAndVersion(ctx, name, blob.Version)
	if err != nil {
		return nil, err
	}

	if len(blob.Ciphertext) < nonceSize {
		return nil, apperrors.Wrap(cryptoDomain.ErrDecryptionFailed, "ciphertext too short")
	}
	nonce := blob.Ciphertext[:nonceSize]
	encryptedData := blob.Ciphertext[nonceSize:]

	handle := keyring.DekHandle{DekID: transitKey.DekID}
	plaintext, err := t.keyring.DecryptWith(ctx, handle, encryptedData, nonce, context)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	return &transitDomain.EncryptedBlob{
		Version:    blob.Version,
		Ciphertext: nil,
		Plaintext:  plaintext,
	}, nil
}

// ListCursor retrieves transit keys ordered by name ascending with cursor pagination.
func (t *transitKeyUseCase) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) ([]*transitDomain.TransitKey, error) {
	return t.transitRepo.ListCursor(ctx, afterName, limit)
}

// PurgeDeleted permanently removes soft-deleted transit keys older than specified days.
func (t *transitKeyUseCase) PurgeDeleted(ctx context.Context, olderThanDays int, dryRun bool) (int64, error) {
	if olderThanDays < 0 {
		return 0, apperrors.New("olderThanDays must be a positive number")
	}

	olderThan := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	return t.transitRepo.HardDelete(ctx, olderThan, dryRun)
}

// NewTransitKeyUseCase creates a new TransitKeyUseCase backed by a Keyring.
func NewTransitKeyUseCase(
	txManager database.TxManager,
	transitRepo TransitKeyRepository,
	kr keyring.Keyring,
) TransitKeyUseCase {
	return &transitKeyUseCase{
		txManager:   txManager,
		transitRepo: transitRepo,
		keyring:     kr,
	}
}
