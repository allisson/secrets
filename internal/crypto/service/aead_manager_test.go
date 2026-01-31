package service

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

func TestNewAEADManager(t *testing.T) {
	manager := NewAEADManager()
	assert.NotNil(t, manager)
}

func TestAEADManagerService_CreateCipher(t *testing.T) {
	manager := NewAEADManager()
	validKey := make([]byte, 32)
	_, err := rand.Read(validKey)
	require.NoError(t, err)

	t.Run("create AES-GCM cipher", func(t *testing.T) {
		cipher, err := manager.CreateCipher(validKey, cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.NotNil(t, cipher)

		// Verify cipher is of the correct type
		_, ok := cipher.(*AESGCMCipher)
		assert.True(t, ok, "cipher should be of type *AESGCMCipher")
	})

	t.Run("create ChaCha20-Poly1305 cipher", func(t *testing.T) {
		cipher, err := manager.CreateCipher(validKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)
		assert.NotNil(t, cipher)

		// Verify cipher is of the correct type
		_, ok := cipher.(*ChaCha20Poly1305Cipher)
		assert.True(t, ok, "cipher should be of type *ChaCha20Poly1305Cipher")
	})

	t.Run("create cipher with unsupported algorithm", func(t *testing.T) {
		_, err := manager.CreateCipher(validKey, cryptoDomain.Algorithm("unsupported"))
		assert.ErrorIs(t, err, cryptoDomain.ErrUnsupportedAlgorithm)
	})

	t.Run("create cipher with invalid key size - too short", func(t *testing.T) {
		shortKey := make([]byte, 16)
		_, err := manager.CreateCipher(shortKey, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})

	t.Run("create cipher with invalid key size - too long", func(t *testing.T) {
		longKey := make([]byte, 64)
		_, err := manager.CreateCipher(longKey, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})

	t.Run("create cipher with empty key", func(t *testing.T) {
		emptyKey := []byte{}
		_, err := manager.CreateCipher(emptyKey, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})

	t.Run("create cipher with nil key", func(t *testing.T) {
		_, err := manager.CreateCipher(nil, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})
}

func TestAEADManagerService_CreateCipher_Functional(t *testing.T) {
	manager := NewAEADManager()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	t.Run("created AES-GCM cipher can encrypt and decrypt", func(t *testing.T) {
		cipher, err := manager.CreateCipher(key, cryptoDomain.AESGCM)
		require.NoError(t, err)

		plaintext := []byte("secret message")
		aad := []byte("additional data")

		// Encrypt
		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.NotNil(t, nonce)

		// Decrypt
		decrypted, err := cipher.Decrypt(ciphertext, nonce, aad)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("created ChaCha20-Poly1305 cipher can encrypt and decrypt", func(t *testing.T) {
		cipher, err := manager.CreateCipher(key, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		plaintext := []byte("secret message")
		aad := []byte("additional data")

		// Encrypt
		ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		assert.NotNil(t, nonce)

		// Decrypt
		decrypted, err := cipher.Decrypt(ciphertext, nonce, aad)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("ciphers created with different algorithms are independent", func(t *testing.T) {
		key1 := make([]byte, 32)
		_, err := rand.Read(key1)
		require.NoError(t, err)

		key2 := make([]byte, 32)
		_, err = rand.Read(key2)
		require.NoError(t, err)

		cipher1, err := manager.CreateCipher(key1, cryptoDomain.AESGCM)
		require.NoError(t, err)

		cipher2, err := manager.CreateCipher(key2, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		plaintext := []byte("test data")

		// Encrypt with cipher1
		ciphertext1, nonce1, err := cipher1.Encrypt(plaintext, nil)
		require.NoError(t, err)

		// Encrypt with cipher2
		ciphertext2, nonce2, err := cipher2.Encrypt(plaintext, nil)
		require.NoError(t, err)

		// Ciphertexts should be different
		assert.NotEqual(t, ciphertext1, ciphertext2)

		// Each cipher can decrypt its own ciphertext
		decrypted1, err := cipher1.Decrypt(ciphertext1, nonce1, nil)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted1)

		decrypted2, err := cipher2.Decrypt(ciphertext2, nonce2, nil)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted2)
	})

	t.Run("multiple ciphers can be created from the same manager", func(t *testing.T) {
		key1 := make([]byte, 32)
		_, err := rand.Read(key1)
		require.NoError(t, err)

		key2 := make([]byte, 32)
		_, err = rand.Read(key2)
		require.NoError(t, err)

		// Create multiple ciphers
		cipher1, err := manager.CreateCipher(key1, cryptoDomain.AESGCM)
		require.NoError(t, err)

		cipher2, err := manager.CreateCipher(key2, cryptoDomain.AESGCM)
		require.NoError(t, err)

		cipher3, err := manager.CreateCipher(key1, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		// All ciphers should be functional
		plaintext := []byte("test")

		_, _, err = cipher1.Encrypt(plaintext, nil)
		require.NoError(t, err)

		_, _, err = cipher2.Encrypt(plaintext, nil)
		require.NoError(t, err)

		_, _, err = cipher3.Encrypt(plaintext, nil)
		require.NoError(t, err)
	})
}

func TestAEADManagerService_CreateCipher_EdgeCases(t *testing.T) {
	manager := NewAEADManager()

	t.Run("create cipher with empty algorithm", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := manager.CreateCipher(key, cryptoDomain.Algorithm(""))
		assert.ErrorIs(t, err, cryptoDomain.ErrUnsupportedAlgorithm)
	})

	t.Run("create cipher with case-sensitive algorithm", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		require.NoError(t, err)

		// Algorithm constants are lowercase, verify case sensitivity
		_, err = manager.CreateCipher(key, cryptoDomain.Algorithm("AES-GCM"))
		assert.ErrorIs(t, err, cryptoDomain.ErrUnsupportedAlgorithm)

		_, err = manager.CreateCipher(key, cryptoDomain.Algorithm("CHACHA20-POLY1305"))
		assert.ErrorIs(t, err, cryptoDomain.ErrUnsupportedAlgorithm)
	})

	t.Run("create cipher with exact 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		cipher, err := manager.CreateCipher(key, cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.NotNil(t, cipher)
	})

	t.Run("create cipher with 31-byte key", func(t *testing.T) {
		key := make([]byte, 31)
		_, err := manager.CreateCipher(key, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})

	t.Run("create cipher with 33-byte key", func(t *testing.T) {
		key := make([]byte, 33)
		_, err := manager.CreateCipher(key, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})
}

func TestAEADManagerService_Integration(t *testing.T) {
	t.Run("manager works with key manager for full encryption flow", func(t *testing.T) {
		aeadManager := NewAEADManager()
		keyManager := NewKeyManager(aeadManager)

		// Generate master key
		masterKeyBytes := make([]byte, 32)
		_, err := rand.Read(masterKeyBytes)
		require.NoError(t, err)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: masterKeyBytes,
		}

		// Create KEK
		kek, err := keyManager.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Create DEK
		dek, err := keyManager.CreateDek(kek, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		// Decrypt DEK
		dekKey, err := keyManager.DecryptDek(dek, kek)
		require.NoError(t, err)

		// Create cipher with decrypted DEK
		cipher, err := aeadManager.CreateCipher(dekKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		// Encrypt application data
		plaintext := []byte("sensitive application data")
		ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
		require.NoError(t, err)

		// Decrypt application data
		decrypted, err := cipher.Decrypt(ciphertext, nonce, nil)
		require.NoError(t, err)

		assert.Equal(t, plaintext, decrypted)
	})
}
