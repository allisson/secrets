// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	validation "github.com/jellydator/validation"

	customValidation "github.com/allisson/secrets/internal/validation"
)

// CreateOrUpdateSecretRequest contains the parameters for creating or updating a secret.
// The path is extracted from the URL parameter, not the request body.
type CreateOrUpdateSecretRequest struct {
	Value string `json:"value"` // base64-encoded plaintext
}

// Validate checks if the create or update secret request is valid.
func (r *CreateOrUpdateSecretRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Value,
			validation.Required,
			customValidation.NotBlank,
			customValidation.Base64,
		),
	)
}
