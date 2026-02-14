// Package service provides authentication-related services for secret generation and token management.
// Implements secure random token generation and Argon2id password hashing for client credentials.
package service

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/allisson/go-pwdhash"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// secretService implements SecretService using Argon2id for password hashing.
type secretService struct {
	hasher *pwdhash.PasswordHasher
}

// GenerateSecret creates a new cryptographically secure 32-byte random secret.
// The secret is base64-encoded for easy transmission and storage.
func (s *secretService) GenerateSecret() (plainSecret string, hashedSecret string, error error) {
	// Generate 32 random bytes (256 bits)
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", apperrors.Wrap(err, "failed to generate random secret")
	}

	// Encode to base64 for text representation
	plainSecret = base64.URLEncoding.EncodeToString(randomBytes)

	// Hash the secret
	hashedSecret, err := s.HashSecret(plainSecret)
	if err != nil {
		return "", "", err
	}

	return plainSecret, hashedSecret, nil
}

// HashSecret hashes a plain text secret using Argon2id.
func (s *secretService) HashSecret(plainSecret string) (hashedSecret string, error error) {
	hashedSecret, err := s.hasher.Hash([]byte(plainSecret))
	if err != nil {
		return "", apperrors.Wrap(err, "failed to hash secret")
	}
	return hashedSecret, nil
}

// CompareSecret performs a constant-time comparison between a plain secret and its hash.
func (s *secretService) CompareSecret(plainSecret string, hashedSecret string) bool {
	ok, err := s.hasher.Verify([]byte(plainSecret), hashedSecret)
	if err != nil {
		return false
	}
	return ok
}

// NewSecretService creates a new SecretService instance using Argon2id hashing.
// Uses the Moderate policy for a balance between security and performance.
func NewSecretService() SecretService {
	hasher, err := pwdhash.New(
		pwdhash.WithPolicy(pwdhash.PolicyModerate),
	)
	if err != nil {
		// This should never happen with valid policy
		panic(err)
	}

	return &secretService{
		hasher: hasher,
	}
}
