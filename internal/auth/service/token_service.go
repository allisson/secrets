package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// tokenService implements TokenService using SHA-256 for token hashing.
type tokenService struct{}

// GenerateToken creates a new cryptographically secure 32-byte random token.
// The token is base64 URL-encoded for easy transmission and storage.
// Returns the plain token and its SHA-256 hash.
func (t *tokenService) GenerateToken() (plainToken string, tokenHash string, error error) {
	// Generate 32 random bytes (256 bits)
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", apperrors.Wrap(err, "failed to generate random token")
	}

	// Encode to base64 URL-safe string for text representation
	plainToken = base64.URLEncoding.EncodeToString(randomBytes)

	// Hash the token using SHA-256
	tokenHash = t.HashToken(plainToken)

	return plainToken, tokenHash, nil
}

// HashToken hashes a plain text token using SHA-256.
// Returns the hash as a hexadecimal string.
func (t *tokenService) HashToken(plainToken string) string {
	hash := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(hash[:])
}

// NewTokenService creates a new TokenService instance using SHA-256 for token hashing.
func NewTokenService() TokenService {
	return &tokenService{}
}
