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
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", apperrors.Wrap(err, "failed to generate random token")
	}

	plainToken = base64.URLEncoding.EncodeToString(randomBytes)

	h := sha256.Sum256([]byte(plainToken))
	tokenHash = hex.EncodeToString(h[:])

	return plainToken, tokenHash, nil
}

// NewTokenService creates a new TokenService instance using SHA-256 for token hashing.
func NewTokenService() TokenService {
	return &tokenService{}
}
