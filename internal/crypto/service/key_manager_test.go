package service

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

func TestNewKeyManager(t *testing.T) {
	aeadManager := NewAEADManager()
	km := NewKeyManager(aeadManager)
	assert.NotNil(t, km)
	assert.NotNil(t, km.aeadManager)
}

func TestKeyManagerService_CreateKek(t *testing.T) {
	aeadManager := NewAEADManager()
	km := NewKeyManager(aeadManager)
	masterKeyBytes := make([]byte, 32)
	_, err := rand.Read(masterKeyBytes)
	require.NoError(t, err)

	masterKey := &cryptoDomain.MasterKey{
		ID:  "test-master-key",
		Key: masterKeyBytes,
	}

	t.Run("create KEK with AES-GCM", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		assert.NotEqual(t, "", kek.ID.String())
		assert.Equal(t, "test-master-key", kek.MasterKeyID)
		assert.Equal(t, cryptoDomain.AESGCM, kek.Algorithm)
		assert.NotNil(t, kek.EncryptedKey)
		assert.NotNil(t, kek.Key)
		assert.Equal(t, 32, len(kek.Key))
		assert.NotNil(t, kek.Nonce)
		assert.Equal(t, uint(1), kek.Version)
		assert.False(t, kek.CreatedAt.IsZero())
	})

	t.Run("create KEK with ChaCha20-Poly1305", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		assert.NotEqual(t, "", kek.ID.String())
		assert.Equal(t, "test-master-key", kek.MasterKeyID)
		assert.Equal(t, cryptoDomain.ChaCha20, kek.Algorithm)
		assert.NotNil(t, kek.EncryptedKey)
		assert.NotNil(t, kek.Key)
		assert.Equal(t, 32, len(kek.Key))
		assert.NotNil(t, kek.Nonce)
		assert.Equal(t, uint(1), kek.Version)
		assert.False(t, kek.CreatedAt.IsZero())
	})

	t.Run("create KEK with unsupported algorithm", func(t *testing.T) {
		_, err := km.CreateKek(masterKey, cryptoDomain.Algorithm("invalid"))
		assert.ErrorIs(t, err, cryptoDomain.ErrUnsupportedAlgorithm)
	})

	t.Run("create KEK with invalid master key size", func(t *testing.T) {
		invalidMasterKey := &cryptoDomain.MasterKey{
			ID:  "invalid-key",
			Key: make([]byte, 16),
		}
		_, err := km.CreateKek(invalidMasterKey, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})

	t.Run("verify KEK can be decrypted with master key", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Verify master key ID is stored
		assert.Equal(t, "test-master-key", kek.MasterKeyID)

		// Decrypt the KEK to verify it was encrypted correctly
		aead, err := NewAESGCM(masterKey.Key)
		require.NoError(t, err)

		decryptedKey, err := aead.Decrypt(kek.EncryptedKey, kek.Nonce, nil)
		require.NoError(t, err)
		assert.Equal(t, kek.Key, decryptedKey)
	})
}

func TestKeyManagerService_DecryptKek(t *testing.T) {
	aeadManager := NewAEADManager()
	km := NewKeyManager(aeadManager)
	masterKeyBytes := make([]byte, 32)
	_, err := rand.Read(masterKeyBytes)
	require.NoError(t, err)

	masterKey := &cryptoDomain.MasterKey{
		ID:  "test-master-key",
		Key: masterKeyBytes,
	}

	t.Run("decrypt KEK successfully with AES-GCM", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Store the original plaintext KEK for comparison
		originalKey := make([]byte, len(kek.Key))
		copy(originalKey, kek.Key)

		// Clear the plaintext key to simulate retrieving from database
		encryptedKek := kek
		encryptedKek.Key = nil

		// Decrypt the KEK
		decryptedKey, err := km.DecryptKek(&encryptedKek, masterKey)
		require.NoError(t, err)
		assert.NotNil(t, decryptedKey)
		assert.Equal(t, 32, len(decryptedKey))
		assert.Equal(t, originalKey, decryptedKey)
	})

	t.Run("decrypt KEK successfully with ChaCha20", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		// Store the original plaintext KEK for comparison
		originalKey := make([]byte, len(kek.Key))
		copy(originalKey, kek.Key)

		// Clear the plaintext key to simulate retrieving from database
		encryptedKek := kek
		encryptedKek.Key = nil

		// Decrypt the KEK
		decryptedKey, err := km.DecryptKek(&encryptedKek, masterKey)
		require.NoError(t, err)
		assert.NotNil(t, decryptedKey)
		assert.Equal(t, 32, len(decryptedKey))
		assert.Equal(t, originalKey, decryptedKey)
	})

	t.Run("decrypt KEK with wrong master key fails", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Create a different master key
		wrongMasterKeyBytes := make([]byte, 32)
		_, err = rand.Read(wrongMasterKeyBytes)
		require.NoError(t, err)

		wrongMasterKey := &cryptoDomain.MasterKey{
			ID:  "wrong-master-key",
			Key: wrongMasterKeyBytes,
		}

		encryptedKek := kek
		encryptedKek.Key = nil

		_, err = km.DecryptKek(&encryptedKek, wrongMasterKey)
		assert.ErrorIs(t, err, cryptoDomain.ErrDecryptionFailed)
	})

	t.Run("decrypt KEK with tampered ciphertext fails", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Tamper with the encrypted key
		tamperedKek := kek
		tamperedKek.Key = nil
		tamperedKek.EncryptedKey[0] ^= 0xFF

		_, err = km.DecryptKek(&tamperedKek, masterKey)
		assert.ErrorIs(t, err, cryptoDomain.ErrDecryptionFailed)
	})

	t.Run("decrypt KEK with invalid master key size", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		invalidMasterKey := &cryptoDomain.MasterKey{
			ID:  "invalid-key",
			Key: make([]byte, 16),
		}

		encryptedKek := kek
		encryptedKek.Key = nil

		_, err = km.DecryptKek(&encryptedKek, invalidMasterKey)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})

	t.Run("decrypt KEK with wrong nonce fails", func(t *testing.T) {
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Use wrong nonce
		wrongKek := kek
		wrongKek.Key = nil
		wrongKek.Nonce = make([]byte, 12)

		_, err = km.DecryptKek(&wrongKek, masterKey)
		assert.ErrorIs(t, err, cryptoDomain.ErrDecryptionFailed)
	})

	t.Run("decrypted KEK can be used to decrypt DEKs", func(t *testing.T) {
		// Create KEK
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Create DEK with the KEK
		dek, err := km.CreateDek(&kek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Simulate retrieving KEK from database (without plaintext key)
		encryptedKek := kek
		encryptedKek.Key = nil

		// Decrypt the KEK
		decryptedKekKey, err := km.DecryptKek(&encryptedKek, masterKey)
		require.NoError(t, err)

		// Use decrypted KEK to decrypt the DEK
		rekeyedKek := encryptedKek
		rekeyedKek.Key = decryptedKekKey

		dekKey, err := km.DecryptDek(&dek, &rekeyedKek)
		require.NoError(t, err)
		assert.NotNil(t, dekKey)
		assert.Equal(t, 32, len(dekKey))
	})
}

func TestKeyManagerService_CreateDek(t *testing.T) {
	aeadManager := NewAEADManager()
	km := NewKeyManager(aeadManager)
	masterKeyBytes := make([]byte, 32)
	_, err := rand.Read(masterKeyBytes)
	require.NoError(t, err)

	masterKey := &cryptoDomain.MasterKey{
		ID:  "test-master-key",
		Key: masterKeyBytes,
	}

	kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
	require.NoError(t, err)

	t.Run("create DEK with AES-GCM", func(t *testing.T) {
		dek, err := km.CreateDek(&kek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		assert.NotEqual(t, "", dek.ID.String())
		assert.Equal(t, kek.ID, dek.KekID)
		assert.Equal(t, cryptoDomain.AESGCM, dek.Algorithm)
		assert.NotNil(t, dek.EncryptedKey)
		assert.NotNil(t, dek.Nonce)
		assert.False(t, dek.CreatedAt.IsZero())
	})

	t.Run("create DEK with ChaCha20-Poly1305", func(t *testing.T) {
		dek, err := km.CreateDek(&kek, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		assert.NotEqual(t, "", dek.ID.String())
		assert.Equal(t, kek.ID, dek.KekID)
		assert.Equal(t, cryptoDomain.ChaCha20, dek.Algorithm)
		assert.NotNil(t, dek.EncryptedKey)
		assert.NotNil(t, dek.Nonce)
		assert.False(t, dek.CreatedAt.IsZero())
	})

	t.Run("create DEK with ChaCha20-Poly1305 KEK", func(t *testing.T) {
		chachaKek, err := km.CreateKek(masterKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		dek, err := km.CreateDek(&chachaKek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		assert.NotEqual(t, "", dek.ID.String())
		assert.Equal(t, chachaKek.ID, dek.KekID)
		assert.Equal(t, cryptoDomain.AESGCM, dek.Algorithm)
	})

	t.Run("create DEK with unsupported algorithm", func(t *testing.T) {
		_, err := km.CreateDek(&kek, cryptoDomain.Algorithm("invalid"))
		assert.NoError(t, err) // DEK creation doesn't validate the DEK algorithm
	})

	t.Run("create DEK with invalid KEK key size", func(t *testing.T) {
		invalidKek := kek
		invalidKek.Key = make([]byte, 16)

		_, err := km.CreateDek(&invalidKek, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})
}

func TestKeyManagerService_DecryptDek(t *testing.T) {
	aeadManager := NewAEADManager()
	km := NewKeyManager(aeadManager)
	masterKeyBytes := make([]byte, 32)
	_, err := rand.Read(masterKeyBytes)
	require.NoError(t, err)

	masterKey := &cryptoDomain.MasterKey{
		ID:  "test-master-key",
		Key: masterKeyBytes,
	}

	kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
	require.NoError(t, err)

	t.Run("decrypt DEK successfully", func(t *testing.T) {
		dek, err := km.CreateDek(&kek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		decryptedKey, err := km.DecryptDek(&dek, &kek)
		require.NoError(t, err)
		assert.NotNil(t, decryptedKey)
		assert.Equal(t, 32, len(decryptedKey))
	})

	t.Run("decrypt DEK with ChaCha20-Poly1305", func(t *testing.T) {
		chachaKek, err := km.CreateKek(masterKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		dek, err := km.CreateDek(&chachaKek, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		decryptedKey, err := km.DecryptDek(&dek, &chachaKek)
		require.NoError(t, err)
		assert.NotNil(t, decryptedKey)
		assert.Equal(t, 32, len(decryptedKey))
	})

	t.Run("decrypt DEK with wrong KEK fails", func(t *testing.T) {
		dek, err := km.CreateDek(&kek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		wrongKek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		_, err = km.DecryptDek(&dek, &wrongKek)
		assert.ErrorIs(t, err, cryptoDomain.ErrDecryptionFailed)
	})

	t.Run("decrypt DEK with tampered ciphertext fails", func(t *testing.T) {
		dek, err := km.CreateDek(&kek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Tamper with the encrypted key
		tamperedDek := dek
		tamperedDek.EncryptedKey[0] ^= 0xFF

		_, err = km.DecryptDek(&tamperedDek, &kek)
		assert.ErrorIs(t, err, cryptoDomain.ErrDecryptionFailed)
	})

	t.Run("decrypt DEK with invalid KEK key size", func(t *testing.T) {
		dek, err := km.CreateDek(&kek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		invalidKek := kek
		invalidKek.Key = make([]byte, 16)

		_, err = km.DecryptDek(&dek, &invalidKek)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})
}

func TestKeyManagerService_EncryptDek(t *testing.T) {
	aeadManager := NewAEADManager()
	km := NewKeyManager(aeadManager)
	masterKeyBytes := make([]byte, 32)
	_, err := rand.Read(masterKeyBytes)
	require.NoError(t, err)

	masterKey := &cryptoDomain.MasterKey{
		ID:  "test-master-key",
		Key: masterKeyBytes,
	}

	kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
	require.NoError(t, err)

	t.Run("encrypt DEK successfully with AES-GCM", func(t *testing.T) {
		plaintextDek := make([]byte, 32)
		_, err := rand.Read(plaintextDek)
		require.NoError(t, err)

		encryptedKey, nonce, err := km.EncryptDek(plaintextDek, &kek)
		require.NoError(t, err)

		assert.NotNil(t, encryptedKey)
		assert.NotNil(t, nonce)
		assert.NotEqual(t, plaintextDek, encryptedKey)
	})

	t.Run("encrypt DEK successfully with ChaCha20", func(t *testing.T) {
		chachaKek, err := km.CreateKek(masterKey, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		plaintextDek := make([]byte, 32)
		_, err = rand.Read(plaintextDek)
		require.NoError(t, err)

		encryptedKey, nonce, err := km.EncryptDek(plaintextDek, &chachaKek)
		require.NoError(t, err)

		assert.NotNil(t, encryptedKey)
		assert.NotNil(t, nonce)
		assert.NotEqual(t, plaintextDek, encryptedKey)
	})

	t.Run("encrypt DEK with invalid KEK key size", func(t *testing.T) {
		invalidKek := kek
		invalidKek.Key = make([]byte, 16)

		plaintextDek := make([]byte, 32)
		_, err = rand.Read(plaintextDek)
		require.NoError(t, err)

		_, _, err = km.EncryptDek(plaintextDek, &invalidKek)
		assert.ErrorIs(t, err, cryptoDomain.ErrInvalidKeySize)
	})
}

func TestKeyManagerService_EnvelopeEncryption(t *testing.T) {
	t.Run("full envelope encryption flow", func(t *testing.T) {
		aeadManager := NewAEADManager()
		km := NewKeyManager(aeadManager)

		// 1. Generate master key (normally stored securely, e.g., in a KMS)
		masterKeyBytes := make([]byte, 32)
		_, err := rand.Read(masterKeyBytes)
		require.NoError(t, err)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: masterKeyBytes,
		}

		// 2. Create KEK encrypted with master key
		kek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Verify master key ID is tracked
		assert.Equal(t, "test-master-key", kek.MasterKeyID)
		require.NoError(t, err)

		// 3. Create DEK encrypted with KEK
		dek, err := km.CreateDek(&kek, cryptoDomain.ChaCha20)
		require.NoError(t, err)

		// 4. Decrypt DEK when needed for data encryption/decryption
		decryptedDekKey, err := km.DecryptDek(&dek, &kek)
		require.NoError(t, err)

		// 5. Use decrypted DEK to encrypt/decrypt actual data
		cipher, err := NewChaCha20Poly1305(decryptedDekKey)
		require.NoError(t, err)

		plaintext := []byte("sensitive data to encrypt")
		ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
		require.NoError(t, err)

		decrypted, err := cipher.Decrypt(ciphertext, nonce, nil)
		require.NoError(t, err)

		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("key rotation scenario", func(t *testing.T) {
		aeadManager := NewAEADManager()
		km := NewKeyManager(aeadManager)

		masterKeyBytes := make([]byte, 32)
		_, err := rand.Read(masterKeyBytes)
		require.NoError(t, err)

		masterKey := &cryptoDomain.MasterKey{
			ID:  "test-master-key",
			Key: masterKeyBytes,
		}

		// Old KEK
		oldKek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// Verify master key ID is tracked
		assert.Equal(t, "test-master-key", oldKek.MasterKeyID)

		// Create DEK with old KEK
		dek, err := km.CreateDek(&oldKek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		// New KEK for rotation
		newKek, err := km.CreateKek(masterKey, cryptoDomain.AESGCM)
		require.NoError(t, err)
		newKek.Version = 2

		// Verify new KEK also tracks master key ID
		assert.Equal(t, "test-master-key", newKek.MasterKeyID)

		// Old DEKs should still be decryptable with old KEK
		decryptedKey, err := km.DecryptDek(&dek, &oldKek)
		require.NoError(t, err)
		assert.NotNil(t, decryptedKey)

		// New DEKs should be created with new KEK
		newDek, err := km.CreateDek(&newKek, cryptoDomain.AESGCM)
		require.NoError(t, err)

		newDecryptedKey, err := km.DecryptDek(&newDek, &newKek)
		require.NoError(t, err)
		assert.NotNil(t, newDecryptedKey)
		assert.NotEqual(t, decryptedKey, newDecryptedKey)
	})
}
