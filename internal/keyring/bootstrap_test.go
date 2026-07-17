package keyring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapWith_NoKeks_ReturnsErrKekNotFound(t *testing.T) {
	ctx := context.Background()
	chain, _ := newTestMasterKeyChain(t, "active")

	_, err := bootstrapWith(ctx, chain, newMemKekStore(), newMemDekStore(), AESGCM)

	require.ErrorIs(t, err, ErrKekNotFound)
}

func TestBootstrapWith_MissingMasterKey_ReturnsErrMasterKeyNotFound(t *testing.T) {
	ctx := context.Background()
	chain, _ := newTestMasterKeyChain(t, "active")

	// A KEK wrapped under a master key that is not in the chain.
	_, foreign := newTestMasterKeyChain(t, "other")
	km := newKeyManager(newAEADManager())
	k, err := km.createKek(foreign, AESGCM)
	require.NoError(t, err)

	kekStore := newMemKekStore()
	require.NoError(t, kekStore.create(ctx, &k))

	_, err = bootstrapWith(ctx, chain, kekStore, newMemDekStore(), AESGCM)

	require.ErrorIs(t, err, ErrMasterKeyNotFound)
}

func TestBootstrapWith_HappyPath_ReturnsUsableKeyring(t *testing.T) {
	ctx := context.Background()
	chain, mk := newTestMasterKeyChain(t, "active")

	km := newKeyManager(newAEADManager())
	k, err := km.createKek(mk, AESGCM)
	require.NoError(t, err)

	kekStore := newMemKekStore()
	require.NoError(t, kekStore.create(ctx, &k))

	kr, err := bootstrapWith(ctx, chain, kekStore, newMemDekStore(), AESGCM)
	require.NoError(t, err)
	require.NotNil(t, kr)

	// The assembled keyring decrypts what it encrypts — proves the loaded KEK
	// was unwrapped correctly and wired into the implementation.
	plaintext := []byte("bootstrapped secret")
	env, err := kr.Encrypt(ctx, plaintext)
	require.NoError(t, err)
	got, err := kr.Decrypt(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}
