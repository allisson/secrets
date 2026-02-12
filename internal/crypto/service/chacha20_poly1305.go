package service

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// ChaCha20Poly1305Cipher implements AEAD using ChaCha20-Poly1305.
type ChaCha20Poly1305Cipher struct {
	aead cipher.AEAD
}

// NewChaCha20Poly1305 creates a new ChaCha20-Poly1305 cipher instance.
// Returns an error if key is not exactly 32 bytes.
func NewChaCha20Poly1305(key []byte) (*ChaCha20Poly1305Cipher, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
	}

	return &ChaCha20Poly1305Cipher{aead: aead}, nil
}

// Encrypt encrypts plaintext using ChaCha20-Poly1305 with optional AAD.
func (c *ChaCha20Poly1305Cipher) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = c.aead.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using ChaCha20-Poly1305 with the provided nonce and AAD.
func (c *ChaCha20Poly1305Cipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}
