// Package usecase implements transit encryption business logic.
//
// Coordinates between the keyring and the transit repository to manage transit keys
// with versioning and envelope encryption.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	metricsLib "github.com/allisson/secrets/internal/metrics"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// transitKeyUseCase implements TransitKeyUseCase for managing transit keys.
type transitKeyUseCase struct {
	txManager   database.TxManager
	transitRepo TransitKeyRepository
	keyring     keyring.Keyring
	metrics     metricsLib.BusinessMetrics
}

// Create generates and persists a new transit key with version 1.
func (t *transitKeyUseCase) Create(
	ctx context.Context,
	name string,
	alg keyring.Algorithm,
) (result *transitDomain.TransitKey, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_key_create", start, err) }()

	var transitKey *transitDomain.TransitKey

	err = t.txManager.WithTx(ctx, func(txCtx context.Context) error {
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
	alg keyring.Algorithm,
) (result *transitDomain.TransitKey, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_key_rotate", start, err) }()

	var newTransitKey *transitDomain.TransitKey

	err = t.txManager.WithTx(ctx, func(txCtx context.Context) error {
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
) (key *transitDomain.TransitKey, alg keyring.Algorithm, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_key_get", start, err) }()
	key, alg, err = t.transitRepo.GetTransitKey(ctx, name, version)
	return
}

// Delete soft-deletes all versions of a transit key by name.
func (t *transitKeyUseCase) Delete(ctx context.Context, name string) (err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_key_delete", start, err) }()
	err = t.transitRepo.Delete(ctx, name)
	return
}

// Encrypt encrypts plaintext using the latest version of a named transit key.
//
// The returned EncryptedBlob.Ciphertext is `nonce || ciphertext`, base64-encoded
// in the wire format `version:base64(...)`. See ADR-0002.
func (t *transitKeyUseCase) Encrypt(
	ctx context.Context,
	name string,
	plaintext, context []byte,
) (result *transitDomain.EncryptedBlob, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_encrypt", start, err) }()

	transitKey, err := t.transitRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	handle := keyring.DekHandle{DekID: transitKey.DekID}
	ciphertext, nonce, err := t.keyring.EncryptWith(ctx, handle, plaintext, context)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to encrypt plaintext")
	}

	blob := transitDomain.NewFramedBlob(transitKey.Version, nonce, ciphertext)
	return &blob, nil
}

// Decrypt decrypts ciphertext using the version specified in the encrypted blob.
func (t *transitKeyUseCase) Decrypt(
	ctx context.Context,
	name string,
	ciphertext string,
	context []byte,
) (result *transitDomain.EncryptedBlob, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_decrypt", start, err) }()

	blob, err := transitDomain.NewEncryptedBlob(ciphertext)
	if err != nil {
		return nil, err
	}

	transitKey, err := t.transitRepo.GetByNameAndVersion(ctx, name, blob.Version)
	if err != nil {
		return nil, err
	}

	nonce, encryptedData, err := blob.SplitNonce()
	if err != nil {
		return nil, keyring.ErrDecryptionFailed
	}

	handle := keyring.DekHandle{DekID: transitKey.DekID}
	plaintext, err := t.keyring.DecryptWith(ctx, handle, encryptedData, nonce, context)
	if err != nil {
		return nil, keyring.ErrDecryptionFailed
	}

	return &transitDomain.EncryptedBlob{
		Version:   blob.Version,
		Plaintext: plaintext,
	}, nil
}

// ListCursor retrieves transit keys ordered by name ascending with cursor pagination.
func (t *transitKeyUseCase) ListCursor(
	ctx context.Context,
	afterName *string,
	limit int,
) (result []*transitDomain.TransitKey, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_key_list", start, err) }()
	result, err = t.transitRepo.ListCursor(ctx, afterName, limit)
	return
}

// PurgeDeleted permanently removes soft-deleted transit keys older than specified days.
func (t *transitKeyUseCase) PurgeDeleted(
	ctx context.Context,
	olderThanDays int,
	dryRun bool,
) (count int64, err error) {
	start := time.Now()
	defer func() { metricsLib.Record(ctx, t.metrics, "transit", "transit_key_purge_deleted", start, err) }()

	if olderThanDays < 0 {
		return 0, apperrors.New("olderThanDays must be a positive number")
	}

	olderThan := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	count, err = t.transitRepo.HardDelete(ctx, olderThan, dryRun)
	return
}

// NewTransitKeyUseCase creates a new TransitKeyUseCase backed by a Keyring.
func NewTransitKeyUseCase(
	txManager database.TxManager,
	transitRepo TransitKeyRepository,
	kr keyring.Keyring,
	bm metricsLib.BusinessMetrics,
) TransitKeyUseCase {
	return &transitKeyUseCase{
		txManager:   txManager,
		transitRepo: transitRepo,
		keyring:     kr,
		metrics:     bm,
	}
}
