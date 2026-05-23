package keyring_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/keyring"
)

func TestFake_EncryptDecrypt_RoundTrip(t *testing.T) {
	f := keyring.NewFake()
	ctx := context.Background()

	plaintext := []byte("hello world")

	env, err := f.Encrypt(ctx, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, env.Ciphertext)
	assert.NotEqual(t, [16]byte{}, [16]byte(env.DekID))

	got, err := f.Decrypt(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestFake_Decrypt_UnknownDekID(t *testing.T) {
	f := keyring.NewFake()
	other := keyring.NewFake()
	ctx := context.Background()

	env, err := other.Encrypt(ctx, []byte("x"))
	require.NoError(t, err)

	_, err = f.Decrypt(ctx, env)
	assert.Error(t, err)
}

func TestFake_AllocateDek_EncryptWith_DecryptWith(t *testing.T) {
	f := keyring.NewFake()
	ctx := context.Background()

	handle, err := f.AllocateDek(ctx, keyring.AESGCM)
	require.NoError(t, err)

	plaintext := []byte("payload")
	aad := []byte("ctx")

	ciphertext, nonce, err := f.EncryptWith(ctx, handle, plaintext, aad)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)
	assert.NotEmpty(t, nonce)

	got, err := f.DecryptWith(ctx, handle, ciphertext, nonce, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestFake_Rewrap_KnownDek(t *testing.T) {
	f := keyring.NewFake()
	ctx := context.Background()

	env, err := f.Encrypt(ctx, []byte("x"))
	require.NoError(t, err)

	require.NoError(t, f.Rewrap(ctx, env.DekID))

	got, err := f.Decrypt(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, []byte("x"), got)
}

func TestFake_FailureInjection(t *testing.T) {
	f := keyring.NewFake()
	ctx := context.Background()

	boom := errors.New("boom")

	f.FailEncrypt = boom
	_, err := f.Encrypt(ctx, []byte("x"))
	assert.ErrorIs(t, err, boom)
	f.FailEncrypt = nil

	env, err := f.Encrypt(ctx, []byte("x"))
	require.NoError(t, err)

	f.FailDecrypt = boom
	_, err = f.Decrypt(ctx, env)
	assert.ErrorIs(t, err, boom)
	f.FailDecrypt = nil

	f.FailAllocate = boom
	_, err = f.AllocateDek(ctx, keyring.AESGCM)
	assert.ErrorIs(t, err, boom)
	f.FailAllocate = nil

	f.FailRewrap = boom
	assert.ErrorIs(t, f.Rewrap(ctx, env.DekID), boom)
}
