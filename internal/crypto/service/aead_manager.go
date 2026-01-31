package service

import (
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// AEADManagerService implements the AEADManager interface for creating AEAD cipher instances.
//
// This service acts as a factory for creating authenticated encryption cipher instances
// based on the specified algorithm. It supports both AES-256-GCM and ChaCha20-Poly1305
// algorithms, providing a unified interface for cipher creation.
//
// The manager validates key sizes and algorithm support before creating cipher instances,
// ensuring that only properly configured ciphers are returned.
//
// Usage example:
//
//	manager := NewAEADManager()
//	key := make([]byte, 32) // 256-bit key
//	rand.Read(key)
//
//	// Create AES-GCM cipher
//	cipher, err := manager.CreateCipher(key, cryptoDomain.AESGCM)
//	if err != nil {
//	    // handle error
//	}
//
//	// Use cipher to encrypt data
//	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
type AEADManagerService struct{}

// NewAEADManager creates a new AEADManagerService instance.
func NewAEADManager() *AEADManagerService {
	return &AEADManagerService{}
}

// CreateCipher creates an AEAD cipher instance based on the specified algorithm.
//
// This method acts as a factory that instantiates the appropriate cipher implementation
// (AES-GCM or ChaCha20-Poly1305) based on the provided algorithm parameter.
//
// The key must be exactly 32 bytes (256 bits) for both algorithms. Keys should be
// generated using a cryptographically secure random number generator (crypto/rand).
//
// Algorithm selection guidelines:
//   - Use AESGCM on systems with hardware AES acceleration (AES-NI)
//   - Use ChaCha20 on mobile devices or systems without AES-NI
//   - Both provide equivalent 256-bit security
//
// Parameters:
//   - key: The encryption key (must be exactly 32 bytes)
//   - alg: The algorithm to use (AESGCM or ChaCha20)
//
// Returns:
//   - An AEAD cipher instance ready for encryption/decryption
//   - ErrInvalidKeySize if the key is not 32 bytes
//   - ErrUnsupportedAlgorithm if the algorithm is not supported
//
// Example:
//
//	// Create cipher for AES-GCM
//	cipher, err := manager.CreateCipher(key, cryptoDomain.AESGCM)
//	if err != nil {
//	    return err
//	}
//
//	// Encrypt data
//	ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
func (am *AEADManagerService) CreateCipher(key []byte, alg cryptoDomain.Algorithm) (AEAD, error) {
	// Validate key size
	if len(key) != 32 {
		return nil, cryptoDomain.ErrInvalidKeySize
	}

	// Create cipher based on algorithm
	switch alg {
	case cryptoDomain.AESGCM:
		return NewAESGCM(key)
	case cryptoDomain.ChaCha20:
		return NewChaCha20Poly1305(key)
	default:
		return nil, cryptoDomain.ErrUnsupportedAlgorithm
	}
}
