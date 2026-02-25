package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
)

type dekUseCase struct {
	txManager  database.TxManager
	dekRepo    DekRepository
	keyManager cryptoService.KeyManager
}

// Rewrap finds DEKs that are not encrypted with the specified KEK ID,
// decrypts them using their old KEKs, and re-encrypts them with the new KEK.
// Returns the number of DEKs rewrapped in this batch.
func (d *dekUseCase) Rewrap(
	ctx context.Context,
	kekChain *cryptoDomain.KekChain,
	newKekID uuid.UUID,
	batchSize int,
) (int, error) {
	// 1. Fetch batch of DEKs not using the new KEK ID
	deks, err := d.dekRepo.GetBatchNotKekID(ctx, newKekID, batchSize)
	if err != nil {
		return 0, err
	}

	if len(deks) == 0 {
		return 0, nil
	}

	// 2. Get the new KEK from the chain
	newKek, ok := kekChain.Get(newKekID)
	if !ok {
		return 0, cryptoDomain.ErrKekNotFound
	}
	if newKek.Key == nil {
		return 0, cryptoDomain.ErrDecryptionFailed // or another appropriate error indicating unwrapped KEK is needed
	}

	// 3. Process each DEK in the batch
	for _, dek := range deks {
		// Get the old KEK
		oldKek, ok := kekChain.Get(dek.KekID)
		if !ok {
			return 0, cryptoDomain.ErrKekNotFound
		}

		// Decrypt the DEK plaintext key using the old KEK
		dekKey, err := d.keyManager.DecryptDek(dek, oldKek)
		if err != nil {
			return 0, err
		}

		// Encrypt the DEK plaintext key using the new KEK
		encryptedKey, nonce, err := d.keyManager.EncryptDek(dekKey, newKek)
		if err != nil {
			cryptoDomain.Zero(dekKey)
			return 0, err
		}
		cryptoDomain.Zero(dekKey)

		// Update DEK entity
		dek.KekID = newKekID
		dek.EncryptedKey = encryptedKey
		dek.Nonce = nonce

		// Save updated DEK
		if err := d.dekRepo.Update(ctx, dek); err != nil {
			return 0, err
		}
	}

	return len(deks), nil
}

// NewDekUseCase creates a new DekUseCase instance.
func NewDekUseCase(
	txManager database.TxManager,
	dekRepo DekRepository,
	keyManager cryptoService.KeyManager,
) DekUseCase {
	return &dekUseCase{
		txManager:  txManager,
		dekRepo:    dekRepo,
		keyManager: keyManager,
	}
}
