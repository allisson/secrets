// Package dto provides data transfer objects for the user HTTP layer.
package dto

import (
	validation "github.com/jellydator/validation"

	appValidation "github.com/allisson/go-project-template/internal/validation"
)

// RegisterUserRequest represents the API request for user registration
type RegisterUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the RegisterUserRequest using the jellydator/validation library
// This provides comprehensive validation including:
// - Required field checks
// - Email format validation
// - Password strength requirements (min 8 chars, uppercase, lowercase, number, special char)
func (r *RegisterUserRequest) Validate() error {
	err := validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required.Error("name is required"),
			appValidation.NotBlank,
			validation.Length(1, 255).Error("name must be between 1 and 255 characters"),
		),
		validation.Field(&r.Email,
			validation.Required.Error("email is required"),
			appValidation.NotBlank,
			appValidation.Email,
			validation.Length(5, 255).Error("email must be between 5 and 255 characters"),
		),
		validation.Field(&r.Password,
			validation.Required.Error("password is required"),
			validation.Length(8, 128).Error("password must be between 8 and 128 characters"),
			appValidation.PasswordStrength{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			},
		),
	)
	return appValidation.WrapValidationError(err)
}
