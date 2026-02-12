// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	validation "github.com/jellydator/validation"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// CreateClientRequest contains the parameters for creating a new authentication client.
type CreateClientRequest struct {
	Name     string                      `json:"name"`
	IsActive bool                        `json:"is_active"`
	Policies []authDomain.PolicyDocument `json:"policies"`
}

// Validate checks if the create client request is valid.
func (r *CreateClientRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 255),
		),
		validation.Field(&r.Policies,
			validation.Required,
			validation.Each(validation.By(validatePolicyDocument)),
		),
	)
}

// UpdateClientRequest contains the parameters for updating an existing client.
type UpdateClientRequest struct {
	Name     string                      `json:"name"`
	IsActive bool                        `json:"is_active"`
	Policies []authDomain.PolicyDocument `json:"policies"`
}

// Validate checks if the update client request is valid.
func (r *UpdateClientRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 255),
		),
		validation.Field(&r.Policies,
			validation.Required,
			validation.Each(validation.By(validatePolicyDocument)),
		),
	)
}

// validatePolicyDocument validates a single policy document.
func validatePolicyDocument(value interface{}) error {
	policy, ok := value.(authDomain.PolicyDocument)
	if !ok {
		return validation.NewError("validation_policy_type", "must be a policy document")
	}

	return validation.ValidateStruct(&policy,
		validation.Field(&policy.Path,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 500),
		),
		validation.Field(&policy.Capabilities,
			validation.Required,
			validation.Length(1, 0), // At least one capability
		),
	)
}

// IssueTokenRequest contains the parameters for issuing an authentication token.
type IssueTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Validate checks if the issue token request is valid.
func (r *IssueTokenRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.ClientID,
			validation.Required,
			customValidation.NotBlank,
		),
		validation.Field(&r.ClientSecret,
			validation.Required,
			customValidation.NotBlank,
		),
	)
}
