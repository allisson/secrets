package service

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// ChaCha20Poly1305Cipher implements the AEAD interface using ChaCha20-Poly1305.
//
// ChaCha20-Poly1305 is a high-speed authenticated encryption algorithm that
// combines the ChaCha20 stream cipher with the Poly1305 MAC for authentication.
// It's particularly efficient on platforms without hardware AES acceleration.
type ChaCha20Poly1305Cipher struct {
	aead cipher.AEAD
}

// NewChaCha20Poly1305 creates a new ChaCha20-Poly1305 cipher instance.
//
// The key must be exactly 32 bytes (256 bits). Keys should be generated
// using a cryptographically secure random number generator.
//
// Returns an error if the key size is invalid (not 32 bytes) or if
// cipher initialization fails.
func NewChaCha20Poly1305(key []byte) (*ChaCha20Poly1305Cipher, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
	}

	return &ChaCha20Poly1305Cipher{aead: aead}, nil
}

// Encrypt encrypts plaintext using ChaCha20-Poly1305 with optional additional authenticated data.
//
// The AAD is authenticated but not encrypted, allowing you to bind the ciphertext
// to additional context (e.g., user ID, timestamp) without encrypting it.
// Pass nil for AAD if no additional data needs to be authenticated.
//
// A unique 12-byte nonce is randomly generated for each encryption operation.
// The nonce must be stored alongside the ciphertext for later decryption.
//
// Returns the ciphertext (which includes the Poly1305 authentication tag), the nonce,
// and any error encountered during encryption.
func (c *ChaCha20Poly1305Cipher) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = c.aead.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using ChaCha20-Poly1305 with the provided nonce and AAD.
//
// The same AAD used during encryption must be provided for successful decryption.
// If the AAD doesn't match, authentication will fail and an error will be returned.
// Pass nil for AAD if no additional data was authenticated during encryption.
//
// This method verifies the Poly1305 authentication tag before returning plaintext,
// ensuring the ciphertext hasn't been tampered with.
//
// Returns the decrypted plaintext or an error if authentication fails,
// the nonce is invalid, or the ciphertext has been modified.
func (c *ChaCha20Poly1305Cipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}
