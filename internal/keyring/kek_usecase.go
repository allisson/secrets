package keyring

import (
	"context"
	"database/sql"

	"github.com/allisson/secrets/internal/database"
)

// KekUseCase manages KEK lifecycle operations. It is the only caller that
// writes KEK rows; all read access goes through Bootstrap.
type KekUseCase interface {
	// Create generates a new KEK wrapped under the active master key and persists it.
	Create(ctx context.Context, masterKeyChain *MasterKeyChain, alg Algorithm) error

	// Rotate creates a new KEK version wrapped under the active master key,
	// incrementing the version counter relative to the current highest KEK.
	Rotate(ctx context.Context, masterKeyChain *MasterKeyChain, alg Algorithm) error
}

type kekUseCase struct {
	txManager  database.TxManager
	kekRepo    kekStore
	keyManager keyManager
}

func (k *kekUseCase) getMasterKey(masterKeyChain *MasterKeyChain, id string) (*MasterKey, error) {
	masterKey, ok := masterKeyChain.Get(id)
	if !ok {
		return nil, ErrMasterKeyNotFound
	}
	return masterKey, nil
}

func (k *kekUseCase) Create(ctx context.Context, masterKeyChain *MasterKeyChain, alg Algorithm) error {
	masterKey, err := k.getMasterKey(masterKeyChain, masterKeyChain.ActiveMasterKeyID())
	if err != nil {
		return err
	}

	kk, err := k.keyManager.createKek(masterKey, alg)
	if err != nil {
		return err
	}

	return k.kekRepo.create(ctx, &kk)
}

func (k *kekUseCase) Rotate(ctx context.Context, masterKeyChain *MasterKeyChain, alg Algorithm) error {
	masterKey, err := k.getMasterKey(masterKeyChain, masterKeyChain.ActiveMasterKeyID())
	if err != nil {
		return err
	}

	return k.txManager.WithTx(ctx, func(ctx context.Context) error {
		keks, err := k.kekRepo.list(ctx)
		if err != nil {
			return err
		}

		if len(keks) == 0 {
			return k.Create(ctx, masterKeyChain, alg)
		}

		currentKek := keks[0]

		kk, err := k.keyManager.createKek(masterKey, alg)
		if err != nil {
			return err
		}

		kk.version = currentKek.version + 1
		return k.kekRepo.create(ctx, &kk)
	})
}

// NewKekUseCase returns a KekUseCase backed by the given transaction manager and database.
func NewKekUseCase(txManager database.TxManager, db *sql.DB) KekUseCase {
	am := newAEADManager()
	km := newKeyManager(am)
	return &kekUseCase{
		txManager:  txManager,
		kekRepo:    newKekRepository(db),
		keyManager: km,
	}
}
