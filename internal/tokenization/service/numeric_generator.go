package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

type numericGenerator struct{}

// NewNumericGenerator creates a new numeric token generator. Generates cryptographically
// secure random numeric strings of the specified length.
func NewNumericGenerator() TokenGenerator {
	return &numericGenerator{}
}

// Generate creates a cryptographically secure random numeric token of the specified length.
// Returns an error if length is less than 1 or greater than 255.
func (g *numericGenerator) Generate(length int) (string, error) {
	if length < 1 {
		return "", errors.New("length must be at least 1")
	}
	if length > tokenizationDomain.MaxTokenLength {
		return "", errors.New("length must not exceed 255")
	}

	digits := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate random digit: %w", err)
		}
		//nolint:gosec // n is bounded [0,9] by big.NewInt(10), safe conversion
		digits[i] = byte('0' + n.Int64())
	}

	return string(digits), nil
}

// Validate checks if the token contains only numeric characters.
func (g *numericGenerator) Validate(token string) error {
	if len(token) == 0 {
		return errors.New("token cannot be empty")
	}

	for _, c := range token {
		if c < '0' || c > '9' {
			return errors.New("token must contain only numeric characters")
		}
	}

	return nil
}
