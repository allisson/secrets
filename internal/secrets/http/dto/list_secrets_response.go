// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// ListSecretsResponse represents a paginated list of secrets in API responses.
type ListSecretsResponse struct {
	Data []SecretResponse `json:"data"`
}

// MapSecretsToListResponse converts a slice of domain secrets to a list response.
func MapSecretsToListResponse(secrets []*secretsDomain.Secret) ListSecretsResponse {
	data := make([]SecretResponse, 0, len(secrets))
	for _, secret := range secrets {
		data = append(data, SecretResponse{
			ID:        secret.ID.String(),
			Path:      secret.Path,
			Version:   secret.Version,
			CreatedAt: secret.CreatedAt,
		})
	}

	return ListSecretsResponse{
		Data: data,
	}
}
