package service

import (
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// AEADManagerService implements the AEADManager interface for creating AEAD cipher instances.
type AEADManagerService struct{}

// NewAEADManager creates a new AEADManagerService.
func NewAEADManager() *AEADManagerService {
	return &AEADManagerService{}
}

// CreateCipher creates an AEAD cipher instance for the specified algorithm.
// Returns ErrInvalidKeySize if key is not 32 bytes or ErrUnsupportedAlgorithm if algorithm is unknown.
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
