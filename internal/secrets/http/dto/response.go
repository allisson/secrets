// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"time"

	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// SecretResponse represents a secret in API responses.
// SECURITY: The Value field contains plaintext and is only included in GET responses.
// Must be transmitted over HTTPS in production.
type SecretResponse struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Version   uint      `json:"version"`
	Value     []byte    `json:"value,omitempty"` // Only included in GET responses
	CreatedAt time.Time `json:"created_at"`
}

// MapSecretToCreateResponse converts a domain secret to an API response for POST operations.
// The plaintext value is excluded for security (only metadata is returned on creation).
func MapSecretToCreateResponse(secret *secretsDomain.Secret) SecretResponse {
	return SecretResponse{
		ID:        secret.ID.String(),
		Path:      secret.Path,
		Version:   secret.Version,
		CreatedAt: secret.CreatedAt,
	}
}

// MapSecretToGetResponse converts a domain secret to an API response for GET operations.
// The plaintext value is included in the response. SECURITY: Caller must zero plaintext
// from the domain object after mapping using cryptoDomain.Zero(secret.Plaintext).
func MapSecretToGetResponse(secret *secretsDomain.Secret) SecretResponse {
	return SecretResponse{
		ID:        secret.ID.String(),
		Path:      secret.Path,
		Version:   secret.Version,
		Value:     secret.Plaintext, // Include decrypted value
		CreatedAt: secret.CreatedAt,
	}
}
