package keyring

import (
	"context"
	"database/sql"
)

// Bootstrap loads all KEKs from the database, decrypts them using masterKeyChain,
// and returns a ready-to-use Keyring. alg is the default Algorithm used for new DEKs.
// Returns ErrKekNotFound if no KEKs exist or ErrMasterKeyNotFound if a KEK references
// a master key that is absent from the chain.
func Bootstrap(
	ctx context.Context,
	masterKeyChain *MasterKeyChain,
	db *sql.DB,
	alg Algorithm,
) (Keyring, error) {
	return bootstrapWith(ctx, masterKeyChain, newKekRepository(db), newDekRepository(db), alg)
}

// bootstrapWith holds the KEK-loading orchestration behind the kekStore/dekStore
// seams so it can be unit-tested without a live database: it lists the persisted
// KEKs, decrypts each under its master key, and assembles the ready-to-use Keyring.
func bootstrapWith(
	ctx context.Context,
	masterKeyChain *MasterKeyChain,
	kekRepo kekStore,
	dekRepo dekStore,
	alg Algorithm,
) (Keyring, error) {
	aeadMgr := newAEADManager()
	km := newKeyManager(aeadMgr)

	keks, err := kekRepo.list(ctx)
	if err != nil {
		return nil, err
	}
	if len(keks) == 0 {
		return nil, ErrKekNotFound
	}

	for _, k := range keks {
		mk, ok := masterKeyChain.get(k.masterKeyID)
		if !ok {
			return nil, ErrMasterKeyNotFound
		}
		key, err := km.decryptKek(k, mk)
		if err != nil {
			return nil, err
		}
		k.key = key
	}

	chain := newKekChain(keks)
	return &keyringImpl{
		kekChain:     chain,
		dekStore:     dekRepo,
		aeadManager:  aeadMgr,
		keyManager:   km,
		dekAlgorithm: alg,
	}, nil
}
