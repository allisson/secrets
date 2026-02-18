package service

import (
	"errors"

	"github.com/google/uuid"
)

type uuidGenerator struct{}

// NewUUIDGenerator creates a new UUID token generator. Generates UUIDv7 tokens.
func NewUUIDGenerator() TokenGenerator {
	return &uuidGenerator{}
}

// Generate creates a new UUIDv7 token. The length parameter is ignored for UUID generation.
func (g *uuidGenerator) Generate(length int) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// Validate checks if the token is a valid UUID format.
func (g *uuidGenerator) Validate(token string) error {
	_, err := uuid.Parse(token)
	if err != nil {
		return errors.New("invalid UUID format")
	}
	return nil
}
