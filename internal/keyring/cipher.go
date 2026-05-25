package keyring

import (
	"crypto/aes"
	gocipher "crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

type aead interface {
	Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error)
	Decrypt(ciphertext, nonce, aad []byte) ([]byte, error)
	NonceSize() int
}

type aeadManager interface {
	createCipher(key []byte, alg Algorithm) (aead, error)
}

type aesgcmCipher struct {
	cipher gocipher.AEAD
}

func newAESGCM(key []byte) (*aesgcmCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be exactly 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	c, err := gocipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &aesgcmCipher{cipher: c}, nil
}

func (a *aesgcmCipher) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, a.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext = a.cipher.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

func (a *aesgcmCipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := a.cipher.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

func (a *aesgcmCipher) NonceSize() int {
	return a.cipher.NonceSize()
}

type chacha20Cipher struct {
	cipher gocipher.AEAD
}

func newChaCha20Poly1305(key []byte) (*chacha20Cipher, error) {
	c, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
	}
	return &chacha20Cipher{cipher: c}, nil
}

func (c *chacha20Cipher) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, c.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext = c.cipher.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

func (c *chacha20Cipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := c.cipher.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

func (c *chacha20Cipher) NonceSize() int {
	return c.cipher.NonceSize()
}

type aeadManagerService struct{}

func newAEADManager() aeadManager {
	return &aeadManagerService{}
}

func (am *aeadManagerService) createCipher(key []byte, alg Algorithm) (aead, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	switch alg {
	case AESGCM:
		return newAESGCM(key)
	case ChaCha20:
		return newChaCha20Poly1305(key)
	default:
		return nil, ErrUnsupportedAlgorithm
	}
}
