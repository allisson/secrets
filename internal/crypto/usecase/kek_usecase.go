// Package usecase implements business logic orchestration for cryptographic operations.
package usecase

import (
	"context"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
)

// kekUseCase implements KEK lifecycle operations including creation, rotation, and unwrapping.
type kekUseCase struct {
	txManager  database.TxManager
	kekRepo    KekRepository
	keyManager cryptoService.KeyManager
}

// getMasterKey retrieves a master key from the chain by its ID.
func (k *kekUseCase) getMasterKey(
	masterKeyChain *cryptoDomain.MasterKeyChain, id string,
) (*cryptoDomain.MasterKey, error) {
	masterKey, ok := masterKeyChain.Get(id)
	if !ok {
		return nil, cryptoDomain.ErrMasterKeyNotFound
	}
	return masterKey, nil
}

// Create generates and persists a new KEK using the active master key.
func (k *kekUseCase) Create(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	alg cryptoDomain.Algorithm,
) error {
	masterKey, err := k.getMasterKey(masterKeyChain, masterKeyChain.ActiveMasterKeyID())
	if err != nil {
		return err
	}

	kek, err := k.keyManager.CreateKek(masterKey, alg)
	if err != nil {
		return err
	}

	return k.kekRepo.Create(ctx, &kek)
}

// Rotate performs atomic KEK rotation by creating a new KEK with incremented version.
// If no KEKs exist, creates the first KEK with version 1.
func (k *kekUseCase) Rotate(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	alg cryptoDomain.Algorithm,
) error {
	masterKey, err := k.getMasterKey(masterKeyChain, masterKeyChain.ActiveMasterKeyID())
	if err != nil {
		return err
	}

	return k.txManager.WithTx(ctx, func(ctx context.Context) error {
		keks, err := k.kekRepo.List(ctx)
		if err != nil {
			return err
		}

		// We don't have any registered keks, we created a new one.
		if len(keks) == 0 {
			return k.Create(ctx, masterKeyChain, alg)
		}

		currentKek := keks[0]

		kek, err := k.keyManager.CreateKek(masterKey, alg)
		if err != nil {
			return err
		}

		kek.Version = currentKek.Version + 1
		return k.kekRepo.Create(ctx, &kek)
	})
}

// Unwrap decrypts all KEKs from the database and returns them in a KekChain for in-memory use.
func (k *kekUseCase) Unwrap(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
) (*cryptoDomain.KekChain, error) {
	keks, err := k.kekRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, kek := range keks {
		masterKey, err := k.getMasterKey(masterKeyChain, kek.MasterKeyID)
		if err != nil {
			return nil, err
		}
		key, err := k.keyManager.DecryptKek(kek, masterKey)
		if err != nil {
			return nil, err
		}
		kek.Key = key
	}

	kekChain := cryptoDomain.NewKekChain(keks)

	return kekChain, nil
}

// NewKekUseCase creates a new KEK use case with the provided dependencies.
func NewKekUseCase(
	txManager database.TxManager,
	kekRepo KekRepository,
	keyManager cryptoService.KeyManager,
) KekUseCase {
	return &kekUseCase{
		txManager:  txManager,
		kekRepo:    kekRepo,
		keyManager: keyManager,
	}
}
