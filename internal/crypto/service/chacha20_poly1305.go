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
// It's designed by Daniel J. Bernstein and provides equivalent security to AES-256-GCM
// while being faster on platforms without hardware AES acceleration.
//
// Performance characteristics:
//   - Excellent performance on mobile devices and embedded systems
//   - Fast software implementation without requiring special CPU instructions
//   - Recommended for ARM processors, IoT devices, and systems without AES-NI
//   - Constant-time implementation resistant to timing attacks
//
// Security properties:
//   - 256-bit key size (maximum security level)
//   - 12-byte nonce (96 bits, randomly generated per encryption)
//   - 16-byte authentication tag (128 bits, appended to ciphertext)
//   - Authenticated encryption prevents tampering and forgery
//
// Thread safety:
//
//	The cipher instance is stateless and safe for concurrent use from multiple
//	goroutines. Each encryption operation generates a unique nonce independently.
//
// Example usage:
//
//	// Create cipher
//	key := make([]byte, 32)
//	rand.Read(key)
//	cipher, err := NewChaCha20Poly1305(key)
//	if err != nil {
//	    return err
//	}
//
//	// Encrypt with AAD
//	userID := []byte("user-123")
//	ciphertext, nonce, err := cipher.Encrypt(plaintext, userID)
//
//	// Decrypt (must use same AAD)
//	plaintext, err := cipher.Decrypt(ciphertext, nonce, userID)
type ChaCha20Poly1305Cipher struct {
	aead cipher.AEAD
}

// NewChaCha20Poly1305 creates a new ChaCha20-Poly1305 cipher instance.
//
// The key must be exactly 32 bytes (256 bits). Keys should be generated
// using crypto/rand for cryptographic security.
//
// This constructor initializes the ChaCha20-Poly1305 AEAD cipher from the
// golang.org/x/crypto/chacha20poly1305 package, which provides a secure
// and performant implementation.
//
// Parameters:
//   - key: A 32-byte (256-bit) encryption key
//
// Returns:
//   - A new ChaCha20Poly1305Cipher instance ready for encryption/decryption
//   - An error if the key size is invalid (not 32 bytes) or cipher initialization fails
//
// Example:
//
//	// Generate a secure key
//	key := make([]byte, 32)
//	if _, err := rand.Read(key); err != nil {
//	    return nil, err
//	}
//
//	// Create cipher
//	cipher, err := NewChaCha20Poly1305(key)
//	if err != nil {
//	    return nil, fmt.Errorf("failed to create cipher: %w", err)
//	}
func NewChaCha20Poly1305(key []byte) (*ChaCha20Poly1305Cipher, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
	}

	return &ChaCha20Poly1305Cipher{aead: aead}, nil
}

// Encrypt encrypts plaintext using ChaCha20-Poly1305 with optional additional authenticated data.
//
// The AAD (Additional Authenticated Data) is authenticated but not encrypted, allowing
// you to bind the ciphertext to additional context (e.g., user ID, record ID, timestamp)
// without encrypting it. This prevents ciphertext from being used in a different context
// even if intercepted. Pass nil for AAD if no additional data needs to be authenticated.
//
// A unique 12-byte nonce is randomly generated for each encryption operation using
// crypto/rand. The nonce must be stored alongside the ciphertext for later decryption.
// With ChaCha20-Poly1305, it's critical that nonces are never reused with the same key.
//
// The returned ciphertext includes the 16-byte Poly1305 authentication tag appended to the end.
//
// Parameters:
//   - plaintext: The data to encrypt (can be empty)
//   - aad: Additional data to authenticate but not encrypt (can be nil)
//
// Returns:
//   - ciphertext: The encrypted data with Poly1305 authentication tag appended
//   - nonce: The randomly generated 12-byte nonce used for this encryption
//   - err: Any error encountered during nonce generation or encryption
//
// Example:
//
//	// Encrypt user data with user ID as AAD
//	userID := []byte("user-123")
//	userData := []byte("sensitive information")
//	ciphertext, nonce, err := cipher.Encrypt(userData, userID)
//	if err != nil {
//	    return err
//	}
//
//	// Store ciphertext and nonce together (e.g., in database)
//	record := EncryptedRecord{
//	    UserID:     "user-123",
//	    Ciphertext: ciphertext,
//	    Nonce:      nonce,
//	}
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
// ensuring the ciphertext hasn't been tampered with. If verification fails, no plaintext
// is returned to prevent processing of potentially malicious data.
//
// Parameters:
//   - ciphertext: The encrypted data to decrypt (includes Poly1305 authentication tag)
//   - nonce: The 12-byte nonce that was used during encryption
//   - aad: The same additional data provided during encryption (can be nil)
//
// Returns:
//   - plaintext: The decrypted data
//   - err: An error if authentication fails, the nonce is invalid, or the ciphertext has been modified
//
// Example:
//
//	// Decrypt with the same AAD used during encryption
//	userID := []byte("user-123")
//	plaintext, err := cipher.Decrypt(record.Ciphertext, record.Nonce, userID)
//	if err != nil {
//	    // Authentication failed - ciphertext may be tampered or wrong key/AAD
//	    return fmt.Errorf("decryption failed: %w", err)
//	}
func (c *ChaCha20Poly1305Cipher) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}
