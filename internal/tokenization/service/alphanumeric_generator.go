package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const alphanumericChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

type alphanumericGenerator struct{}

// NewAlphanumericGenerator creates a new alphanumeric token generator. Generates
// cryptographically secure random alphanumeric tokens using [A-Za-z0-9].
func NewAlphanumericGenerator() TokenGenerator {
	return &alphanumericGenerator{}
}

// Generate creates a cryptographically secure random alphanumeric token of the specified length.
// Uses characters from [A-Za-z0-9]. Returns an error if length is less than 1 or greater than 255.
func (g *alphanumericGenerator) Generate(length int) (string, error) {
	if length < 1 {
		return "", errors.New("length must be at least 1")
	}
	if length > 255 {
		return "", errors.New("length must not exceed 255")
	}

	token := make([]byte, length)
	charsLen := big.NewInt(int64(len(alphanumericChars)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		token[i] = alphanumericChars[n.Int64()]
	}

	return string(token), nil
}

// Validate checks if the token contains only alphanumeric characters [A-Za-z0-9].
func (g *alphanumericGenerator) Validate(token string) error {
	if len(token) == 0 {
		return errors.New("token cannot be empty")
	}

	for _, c := range token {
		if !isAlphanumeric(c) {
			return errors.New("token must contain only alphanumeric characters [A-Za-z0-9]")
		}
	}

	return nil
}

// isAlphanumeric checks if a character is alphanumeric [A-Za-z0-9].
func isAlphanumeric(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}
