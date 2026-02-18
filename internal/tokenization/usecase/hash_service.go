package usecase

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashService provides cryptographic hashing for deterministic token lookups.
type HashService interface {
	Hash(value []byte) string
}

type sha256HashService struct{}

// NewSHA256HashService creates a new SHA-256 hash service.
func NewSHA256HashService() HashService {
	return &sha256HashService{}
}

// Hash computes the SHA-256 hash of the input value and returns it as a hex string.
func (s *sha256HashService) Hash(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
