// Package service provides token generation services for various formats.
// Supports UUID, numeric, Luhn-preserving, and alphanumeric token generation.
package service

// TokenGenerator defines the interface for token generation.
type TokenGenerator interface {
	Generate(length int) (string, error)
	Validate(token string) error
}
