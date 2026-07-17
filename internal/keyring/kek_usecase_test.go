package keyring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKekUseCase(store kekStore) *kekUseCase {
	return &kekUseCase{
		txManager:  fakeTxManager{},
		kekRepo:    store,
		keyManager: newKeyManager(newAEADManager()),
	}
}

func TestKekUseCase_Create_MasterKeyNotInChain_ReturnsError(t *testing.T) {
	ctx := context.Background()
	// Chain declares an active ID but holds no key material for it.
	chain := NewMasterKeyChain("active")
	uc := newTestKekUseCase(newMemKekStore())

	err := uc.Create(ctx, chain, AESGCM)

	require.ErrorIs(t, err, ErrMasterKeyNotFound)
}

func TestKekUseCase_Create_PersistsVersionOneKek(t *testing.T) {
	ctx := context.Background()
	chain, _ := newTestMasterKeyChain(t, "active")
	store := newMemKekStore()
	uc := newTestKekUseCase(store)

	require.NoError(t, uc.Create(ctx, chain, AESGCM))

	keks, err := store.list(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 1)
	assert.Equal(t, uint(1), keks[0].version)
	assert.Equal(t, "active", keks[0].masterKeyID)
}

func TestKekUseCase_Rotate_EmptyStore_CreatesFirstKek(t *testing.T) {
	ctx := context.Background()
	chain, _ := newTestMasterKeyChain(t, "active")
	store := newMemKekStore()
	uc := newTestKekUseCase(store)

	require.NoError(t, uc.Rotate(ctx, chain, AESGCM))

	keks, err := store.list(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 1)
	assert.Equal(t, uint(1), keks[0].version)
}

func TestKekUseCase_Rotate_IncrementsVersion(t *testing.T) {
	ctx := context.Background()
	chain, mk := newTestMasterKeyChain(t, "active")
	store := newMemKekStore()

	// Seed an existing KEK at version 5.
	km := newKeyManager(newAEADManager())
	existing, err := km.createKek(mk, AESGCM)
	require.NoError(t, err)
	existing.version = 5
	require.NoError(t, store.create(ctx, &existing))

	uc := newTestKekUseCase(store)
	require.NoError(t, uc.Rotate(ctx, chain, AESGCM))

	keks, err := store.list(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 2)
	// list is version-descending; the rotated KEK is version 6.
	assert.Equal(t, uint(6), keks[0].version)
}
