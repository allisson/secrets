// Package validation provides custom validation rules for the application.
package validation

import (
	"strings"

	validation "github.com/jellydator/validation"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// WrapValidationError wraps validation errors as domain ErrInvalidInput.
func WrapValidationError(err error) error {
	if err == nil {
		return nil
	}
	return apperrors.Wrap(apperrors.ErrInvalidInput, err.Error())
}

// NotBlank validates that a string is not empty after trimming whitespace
var NotBlank = validation.NewStringRuleWithError(
	func(s string) bool {
		return strings.TrimSpace(s) != ""
	},
	validation.NewError("validation_not_blank", "must not be blank"),
)
