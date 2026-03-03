package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HashService provides cryptographic hashing for deterministic token lookups.
type HashService interface {
	Hash(value []byte, salt []byte) string
}

type sha256HashService struct{}

// NewSHA256HashService creates a new SHA-256 hash service.
func NewSHA256HashService() HashService {
	return &sha256HashService{}
}

// Hash computes the hash of the input value.
// If salt is provided, it uses HMAC-SHA256 with the salt as the key.
// If salt is empty, it falls back to simple SHA-256 for backward compatibility.
func (s *sha256HashService) Hash(value []byte, salt []byte) string {
	if len(salt) == 0 {
		hash := sha256.Sum256(value)
		return hex.EncodeToString(hash[:])
	}

	h := hmac.New(sha256.New, salt)
	h.Write(value)
	return hex.EncodeToString(h.Sum(nil))
}
