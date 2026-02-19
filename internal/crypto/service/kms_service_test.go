package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"gocloud.dev/secrets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateLocalSecretsURI generates a base64key:// URI for testing.
func generateLocalSecretsURI(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return "base64key://" + base64.URLEncoding.EncodeToString(key)
}

func TestKMSService_OpenKeeper(t *testing.T) {
	ctx := context.Background()
	kmsService := NewKMSService()

	t.Run("Success_LocalSecrets", func(t *testing.T) {
		keyURI := generateLocalSecretsURI(t)

		keeper, err := kmsService.OpenKeeper(ctx, keyURI)
		require.NoError(t, err)
		require.NotNil(t, keeper)

		// Verify it's actually a *secrets.Keeper
		_, ok := keeper.(*secrets.Keeper)
		assert.True(t, ok, "keeper should be *secrets.Keeper")

		// Cleanup
		defer func() {
			assert.NoError(t, keeper.Close())
		}()
	})

	t.Run("Error_InvalidURI", func(t *testing.T) {
		invalidURI := "invalid://uri"

		keeper, err := kmsService.OpenKeeper(ctx, invalidURI)
		assert.Error(t, err)
		assert.Nil(t, keeper)
		assert.Contains(t, err.Error(), "failed to open KMS keeper")
	})

	t.Run("Error_EmptyURI", func(t *testing.T) {
		keeper, err := kmsService.OpenKeeper(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, keeper)
	})
}

func TestKMSService_KeeperDecryptFunctionality(t *testing.T) {
	ctx := context.Background()
	kmsService := NewKMSService()
	keyURI := generateLocalSecretsURI(t)

	keeperInterface, err := kmsService.OpenKeeper(ctx, keyURI)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, keeperInterface.Close())
	}()

	// Type assert to get the actual *secrets.Keeper for Encrypt
	keeper, ok := keeperInterface.(*secrets.Keeper)
	require.True(t, ok, "keeper should be *secrets.Keeper")

	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "ShortText",
			plaintext: []byte("hello"),
		},
		{
			name: "LongText",
			plaintext: []byte(
				"This is a longer piece of text that should be encrypted and decrypted successfully",
			),
		},
		{
			name:      "BinaryData",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "MasterKeySize",
			plaintext: make([]byte, 32), // 32-byte master key
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt using the keeper
			ciphertext, err := keeper.Encrypt(ctx, tc.plaintext)
			require.NoError(t, err)
			assert.NotEqual(t, tc.plaintext, ciphertext)

			// Decrypt using the keeper interface (as used by domain layer)
			decrypted, err := keeperInterface.Decrypt(ctx, ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tc.plaintext, decrypted)
		})
	}
}

func TestKMSService_DecryptInvalidCiphertext(t *testing.T) {
	ctx := context.Background()
	kmsService := NewKMSService()
	keyURI := generateLocalSecretsURI(t)

	keeper, err := kmsService.OpenKeeper(ctx, keyURI)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, keeper.Close())
	}()

	invalidCiphertext := []byte("not a valid ciphertext")

	decrypted, err := keeper.Decrypt(ctx, invalidCiphertext)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
}

func TestKMSService_MultipleKeepers(t *testing.T) {
	ctx := context.Background()
	kmsService := NewKMSService()

	// Create two different keepers with different keys
	keyURI1 := generateLocalSecretsURI(t)
	keyURI2 := generateLocalSecretsURI(t)

	keeper1Interface, err := kmsService.OpenKeeper(ctx, keyURI1)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, keeper1Interface.Close())
	}()

	keeper2Interface, err := kmsService.OpenKeeper(ctx, keyURI2)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, keeper2Interface.Close())
	}()

	// Type assert to encrypt
	keeper1, ok := keeper1Interface.(*secrets.Keeper)
	require.True(t, ok)

	plaintext := []byte("test data")

	// Encrypt with keeper1
	ciphertext, err := keeper1.Encrypt(ctx, plaintext)
	require.NoError(t, err)

	// Decrypt with keeper1 should succeed
	decrypted1, err := keeper1Interface.Decrypt(ctx, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	// Decrypt with keeper2 should fail (different key)
	decrypted2, err := keeper2Interface.Decrypt(ctx, ciphertext)
	assert.Error(t, err)
	assert.Nil(t, decrypted2)
}
