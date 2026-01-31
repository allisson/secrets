package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// AESGCMCipher implements the AEAD interface using AES-256-GCM
// (Advanced Encryption Standard with Galois/Counter Mode).
//
// AES-GCM provides authenticated encryption with associated data, combining
// the confidentiality of AES encryption with the authenticity of GMAC.
// This implementation uses AES-256 with a 256-bit key for maximum security.
type AESGCMCipher struct {
	aead cipher.AEAD
}

// NewAESGCM creates a new AES-256-GCM cipher instance.
//
// The key must be exactly 32 bytes (256 bits). Using a shorter or longer
// key will result in an error. Keys should be generated using a
// cryptographically secure random number generator.
//
// Returns an error if the key size is invalid or if cipher initialization fails.
func NewAESGCM(key []byte) (*AESGCMCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be exactly 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &AESGCMCipher{aead: aead}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with optional additional authenticated data.
//
// The AAD is authenticated but not encrypted, allowing you to bind the ciphertext
// to additional context (e.g., user ID, timestamp) without encrypting it.
// Pass nil for AAD if no additional data needs to be authenticated.
//
// A unique 12-byte nonce is randomly generated for each encryption operation.
// The nonce must be stored alongside the ciphertext for later decryption.
//
// Returns the ciphertext (which includes the authentication tag), the nonce,
// and any error encountered during encryption.
func (a *AESGCMCipher) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, a.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = a.aead.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided nonce and AAD.
//
// The same AAD used during encryption must be provided for successful decryption.
// If the AAD doesn't match, authentication will fail and an error will be returned.
// Pass nil for AAD if no additional data was authenticated during encryption.
//
// This method verifies the authentication tag before returning plaintext,
// ensuring the ciphertext hasn't been tampered with.
//
// Returns the decrypted plaintext or an error if authentication fails,
// the nonce is invalid, or the ciphertext has been modified.
func (a *AESGCMCipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := a.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}
