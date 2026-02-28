// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"time"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// TokenizationKeyResponse represents a tokenization key in API responses.
type TokenizationKeyResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Version         uint      `json:"version"`
	FormatType      string    `json:"format_type"`
	IsDeterministic bool      `json:"is_deterministic"`
	DekID           string    `json:"dek_id"`
	CreatedAt       time.Time `json:"created_at"`
}

// MapTokenizationKeyToResponse converts a domain tokenization key to an API response.
func MapTokenizationKeyToResponse(key *tokenizationDomain.TokenizationKey) TokenizationKeyResponse {
	return TokenizationKeyResponse{
		ID:              key.ID.String(),
		Name:            key.Name,
		Version:         key.Version,
		FormatType:      string(key.FormatType),
		IsDeterministic: key.IsDeterministic,
		DekID:           key.DekID.String(),
		CreatedAt:       key.CreatedAt,
	}
}

// TokenizeResponse represents the result of tokenizing a value.
type TokenizeResponse struct {
	Token     string         `json:"token"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
}

// MapTokenToTokenizeResponse converts a domain token to a tokenize API response.
func MapTokenToTokenizeResponse(token *tokenizationDomain.Token) TokenizeResponse {
	return TokenizeResponse{
		Token:     token.Token,
		Metadata:  token.Metadata,
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}
}

// DetokenizeResponse represents the result of detokenizing a token.
type DetokenizeResponse struct {
	Plaintext string         `json:"plaintext"` // Base64-encoded plaintext
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ValidateTokenResponse represents the result of validating a token.
type ValidateTokenResponse struct {
	Valid bool `json:"valid"`
}
