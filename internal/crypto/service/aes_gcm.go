package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// AESGCMCipher implements AEAD using AES-256-GCM.
type AESGCMCipher struct {
	aead cipher.AEAD
}

// NewAESGCM creates a new AES-256-GCM cipher instance.
// Returns an error if key is not exactly 32 bytes.
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

// Encrypt encrypts plaintext using AES-256-GCM with optional AAD.
func (a *AESGCMCipher) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, a.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = a.aead.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided nonce and AAD.
func (a *AESGCMCipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := a.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// NonceSize returns the size of the nonce required by the AES-GCM cipher.
func (a *AESGCMCipher) NonceSize() int {
	return a.aead.NonceSize()
}
