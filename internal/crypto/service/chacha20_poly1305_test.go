package service

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChaCha20Poly1305(t *testing.T) {
	t.Run("valid 256-bit key", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		cipher, err := NewChaCha20Poly1305(key)
		assert.NoError(t, err)
		assert.NotNil(t, cipher)
	})

	t.Run("invalid key size", func(t *testing.T) {
		key := make([]byte, 16) // Invalid size (ChaCha20-Poly1305 requires 32 bytes)
		_, err := rand.Read(key)
		require.NoError(t, err)

		cipher, err := NewChaCha20Poly1305(key)
		assert.Error(t, err)
		assert.Nil(t, cipher)
	})

	t.Run("invalid key size - too large", func(t *testing.T) {
		key := make([]byte, 64) // Invalid size
		_, err := rand.Read(key)
		require.NoError(t, err)

		cipher, err := NewChaCha20Poly1305(key)
		assert.Error(t, err)
		assert.Nil(t, cipher)
	})
}

func TestChaCha20Poly1305Cipher_Encrypt(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	cipher, err := NewChaCha20Poly1305(key)
	require.NoError(t, err)

	t.Run("encrypt with plaintext and AAD", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		aad := []byte("additional authenticated data")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		assert.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.NotNil(t, nonce)
		assert.NotEqual(t, plaintext, ciphertext)
		assert.Equal(t, 12, len(nonce)) // ChaCha20-Poly1305 standard nonce size
	})

	t.Run("encrypt without AAD", func(t *testing.T) {
		plaintext := []byte("Hello, World!")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
		assert.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.NotNil(t, nonce)
	})

	t.Run("encrypt empty plaintext", func(t *testing.T) {
		plaintext := []byte("")
		aad := []byte("aad")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		assert.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.NotNil(t, nonce)
	})

	t.Run("nonce is unique for each encryption", func(t *testing.T) {
		plaintext := []byte("test")
		aad := []byte("aad")

		_, nonce1, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		_, nonce2, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		assert.NotEqual(t, nonce1, nonce2)
	})
}

func TestChaCha20Poly1305Cipher_Decrypt(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	cipher, err := NewChaCha20Poly1305(key)
	require.NoError(t, err)

	t.Run("decrypt successfully", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		aad := []byte("additional authenticated data")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		decrypted, err := cipher.Decrypt(ciphertext, nonce, aad)
		assert.NoError(t, err)
		assert.True(t, bytes.Equal(plaintext, decrypted))
	})

	t.Run("decrypt with wrong AAD fails", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		aad := []byte("correct aad")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		wrongAAD := []byte("wrong aad")
		decrypted, err := cipher.Decrypt(ciphertext, nonce, wrongAAD)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("decrypt with wrong nonce fails", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		aad := []byte("aad")

		ciphertext, _, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		wrongNonce := make([]byte, 12)
		_, err = rand.Read(wrongNonce)
		require.NoError(t, err)

		decrypted, err := cipher.Decrypt(ciphertext, wrongNonce, aad)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("decrypt with tampered ciphertext fails", func(t *testing.T) {
		plaintext := []byte("Hello, World!")
		aad := []byte("aad")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		// Tamper with ciphertext
		if len(ciphertext) > 0 {
			ciphertext[0] ^= 1
		}

		decrypted, err := cipher.Decrypt(ciphertext, nonce, aad)
		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("decrypt empty ciphertext", func(t *testing.T) {
		plaintext := []byte("")
		aad := []byte("aad")

		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)

		decrypted, err := cipher.Decrypt(ciphertext, nonce, aad)
		assert.NoError(t, err)
		assert.True(t, bytes.Equal(plaintext, decrypted))
	})
}

func TestChaCha20Poly1305Cipher_EncryptDecrypt_Integration(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	cipher, err := NewChaCha20Poly1305(key)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		plaintext []byte
		aad       []byte
	}{
		{
			name:      "short message",
			plaintext: []byte("test"),
			aad:       []byte("metadata"),
		},
		{
			name:      "long message",
			plaintext: bytes.Repeat([]byte("a"), 10000),
			aad:       []byte("large data"),
		},
		{
			name:      "message with unicode",
			plaintext: []byte("Hello 世界! 🔐"),
			aad:       []byte("unicode test"),
		},
		{
			name:      "message with special characters",
			plaintext: []byte("!@#$%^&*()_+-=[]{}|;:',.<>?/~`"),
			aad:       []byte("special"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, nonce, err := cipher.Encrypt(tc.plaintext, tc.aad)
			require.NoError(t, err)

			decrypted, err := cipher.Decrypt(ciphertext, nonce, tc.aad)
			require.NoError(t, err)

			assert.True(t, bytes.Equal(tc.plaintext, decrypted))
		})
	}
}
