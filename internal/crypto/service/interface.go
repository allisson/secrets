// Package service provides cryptographic services for envelope encryption.
// Implements AEAD ciphers (AES-256-GCM, ChaCha20-Poly1305) for KEK/DEK management.
package service

import (
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// AEAD defines the interface for Authenticated Encryption with Associated Data.
type AEAD interface {
	// Encrypt encrypts plaintext with optional AAD and returns ciphertext and nonce.
	Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error)

	// Decrypt decrypts ciphertext using the provided nonce and AAD.
	Decrypt(ciphertext, nonce, aad []byte) ([]byte, error)
}

// AEADManager defines the interface for creating AEAD cipher instances.
type AEADManager interface {
	// CreateCipher creates an AEAD cipher instance for the specified algorithm.
	CreateCipher(key []byte, alg cryptoDomain.Algorithm) (AEAD, error)
}

// KeyManager defines the interface for managing KEKs and DEKs in envelope encryption.
type KeyManager interface {
	// CreateKek creates a new KEK encrypted with the master key.
	CreateKek(
		masterKey *cryptoDomain.MasterKey,
		alg cryptoDomain.Algorithm,
	) (cryptoDomain.Kek, error)

	// DecryptKek decrypts a KEK using the master key.
	DecryptKek(kek *cryptoDomain.Kek, masterKey *cryptoDomain.MasterKey) ([]byte, error)

	// CreateDek creates a new DEK encrypted with the KEK.
	CreateDek(kek *cryptoDomain.Kek, alg cryptoDomain.Algorithm) (cryptoDomain.Dek, error)

	// DecryptDek decrypts a DEK using the KEK.
	DecryptDek(dek *cryptoDomain.Dek, kek *cryptoDomain.Kek) ([]byte, error)
}
