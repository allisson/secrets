// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"time"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// CreateClientResponse contains the result of creating a new client.
// SECURITY: The secret is only returned once and must be saved securely.
type CreateClientResponse struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// ClientResponse represents a client in API responses (excludes secret).
type ClientResponse struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	IsActive  bool                        `json:"is_active"`
	Policies  []authDomain.PolicyDocument `json:"policies"`
	CreatedAt time.Time                   `json:"created_at"`
}

// MapClientToResponse converts a domain client to an API response.
func MapClientToResponse(client *authDomain.Client) ClientResponse {
	return ClientResponse{
		ID:        client.ID.String(),
		Name:      client.Name,
		IsActive:  client.IsActive,
		Policies:  client.Policies,
		CreatedAt: client.CreatedAt,
	}
}

// IssueTokenResponse contains the result of issuing a token.
// SECURITY: The token is only returned once and must be saved securely.
type IssueTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
