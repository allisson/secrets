// Package usecase implements transit encryption business logic.
//
// Coordinates between cryptographic services and repositories to manage transit keys
// with versioning and envelope encryption. Uses TxManager for transactional consistency.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// transitKeyUseCase implements TransitKeyUseCase for managing transit keys.
type transitKeyUseCase struct {
	txManager   database.TxManager
	transitRepo TransitKeyRepository
	dekRepo     DekRepository
	keyManager  cryptoService.KeyManager
	aeadManager cryptoService.AEADManager
	kekChain    *cryptoDomain.KekChain
}

// getKek retrieves a KEK from the chain by its ID.
func (t *transitKeyUseCase) getKek(kekID uuid.UUID) (*cryptoDomain.Kek, error) {
	kek, ok := t.kekChain.Get(kekID)
	if !ok {
		return nil, cryptoDomain.ErrKekNotFound
	}
	return kek, nil
}

// Create generates and persists a new transit key with version 1.
func (t *transitKeyUseCase) Create(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
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

	// Create transit key with version 1
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Version:   1,
		DekID:     dek.ID,
		CreatedAt: time.Now().UTC(),
	}

	// Persist transit key
	if err := t.transitRepo.Create(ctx, transitKey); err != nil {
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
		// Get latest transit key version
		currentKey, err := t.transitRepo.GetByName(txCtx, name)
		if err != nil {
			// If key doesn't exist, create first version
			if apperrors.Is(err, transitDomain.ErrTransitKeyNotFound) {
				newTransitKey, err = t.Create(txCtx, name, alg)
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

		// Create new transit key with incremented version
		newTransitKey = &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   currentKey.Version + 1,
			DekID:     dek.ID,
			CreatedAt: time.Now().UTC(),
		}

		// Persist new transit key
		return t.transitRepo.Create(txCtx, newTransitKey)
	})

	if err != nil {
		return nil, err
	}

	return newTransitKey, nil
}

// Delete soft-deletes a transit key by setting its deleted_at timestamp.
func (t *transitKeyUseCase) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	return t.transitRepo.Delete(ctx, transitKeyID)
}

// Encrypt encrypts plaintext using the latest version of a named transit key.
func (t *transitKeyUseCase) Encrypt(
	ctx context.Context,
	name string,
	plaintext []byte,
) (*transitDomain.EncryptedBlob, error) {
	// Get latest transit key version
	transitKey, err := t.transitRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get DEK by transit key's DekID
	dek, err := t.dekRepo.Get(ctx, transitKey.DekID)
	if err != nil {
		return nil, err
	}

	// Get KEK for decrypting DEK
	kek, err := t.getKek(dek.KekID)
	if err != nil {
		return nil, err
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Encrypt plaintext
	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to encrypt plaintext")
	}

	// Combine ciphertext and nonce (nonce is prepended to ciphertext by AEAD)
	// The AEAD Encrypt returns ciphertext with authentication tag, we need to store nonce separately
	//nolint:gocritic // intentionally creating new slice with combined nonce and ciphertext
	encryptedData := append(nonce, ciphertext...)

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
	ciphertext []byte,
) (*transitDomain.EncryptedBlob, error) {
	// Parse encrypted blob from ciphertext
	blob, err := transitDomain.NewEncryptedBlob(string(ciphertext))
	if err != nil {
		return nil, err
	}

	// Get transit key by name and version from blob
	transitKey, err := t.transitRepo.GetByNameAndVersion(ctx, name, blob.Version)
	if err != nil {
		return nil, err
	}

	// Get DEK by transit key's DekID
	dek, err := t.dekRepo.Get(ctx, transitKey.DekID)
	if err != nil {
		return nil, err
	}

	// Get KEK for decrypting DEK
	kek, err := t.getKek(dek.KekID)
	if err != nil {
		return nil, err
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Extract nonce and ciphertext from encrypted data
	// The nonce is prepended to the ciphertext
	nonceSize := 12 // Standard nonce size for AES-GCM and ChaCha20-Poly1305
	if len(blob.Ciphertext) < nonceSize {
		return nil, apperrors.Wrap(cryptoDomain.ErrDecryptionFailed, "ciphertext too short")
	}

	nonce := blob.Ciphertext[:nonceSize]
	encryptedData := blob.Ciphertext[nonceSize:]

	// Decrypt ciphertext
	plaintext, err := cipher.Decrypt(encryptedData, nonce, nil)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	return &transitDomain.EncryptedBlob{
		Version:    blob.Version,
		Ciphertext: nil,
		Plaintext:  plaintext,
	}, nil
}

// NewTransitKeyUseCase creates a new TransitKeyUseCase with injected dependencies.
func NewTransitKeyUseCase(
	txManager database.TxManager,
	transitRepo TransitKeyRepository,
	dekRepo DekRepository,
	keyManager cryptoService.KeyManager,
	aeadManager cryptoService.AEADManager,
	kekChain *cryptoDomain.KekChain,
) TransitKeyUseCase {
	return &transitKeyUseCase{
		txManager:   txManager,
		transitRepo: transitRepo,
		dekRepo:     dekRepo,
		keyManager:  keyManager,
		aeadManager: aeadManager,
		kekChain:    kekChain,
	}
}
