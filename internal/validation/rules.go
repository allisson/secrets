// Package validation provides custom validation rules for the application.
package validation

import (
	"regexp"
	"strings"
	"unicode"

	validation "github.com/jellydator/validation"

	apperrors "github.com/allisson/go-project-template/internal/errors"
)

var (
	// emailRegex is a basic email validation pattern
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// WrapValidationError wraps validation errors as domain ErrInvalidInput
func WrapValidationError(err error) error {
	if err == nil {
		return nil
	}
	return apperrors.Wrap(apperrors.ErrInvalidInput, err.Error())
}

// PasswordStrength validates password meets minimum security requirements
type PasswordStrength struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
}

// Validate checks if the password meets the configured requirements
func (p PasswordStrength) Validate(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return validation.NewError("validation_password_strength", "password must be a string")
	}

	if len(s) < p.MinLength {
		return validation.NewError(
			"validation_password_min_length",
			"password must be at least "+string(rune(p.MinLength+48))+" characters",
		)
	}

	if p.RequireUpper && !hasUpperCase(s) {
		return validation.NewError(
			"validation_password_uppercase",
			"password must contain at least one uppercase letter",
		)
	}

	if p.RequireLower && !hasLowerCase(s) {
		return validation.NewError(
			"validation_password_lowercase",
			"password must contain at least one lowercase letter",
		)
	}

	if p.RequireNumber && !hasNumber(s) {
		return validation.NewError("validation_password_number", "password must contain at least one number")
	}

	if p.RequireSpecial && !hasSpecialChar(s) {
		return validation.NewError(
			"validation_password_special",
			"password must contain at least one special character",
		)
	}

	return nil
}

// hasUpperCase checks if string contains uppercase letters
func hasUpperCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// hasLowerCase checks if string contains lowercase letters
func hasLowerCase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

// hasNumber checks if string contains numbers
func hasNumber(s string) bool {
	for _, r := range s {
		if unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

// hasSpecialChar checks if string contains special characters
func hasSpecialChar(s string) bool {
	for _, r := range s {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return true
		}
	}
	return false
}

// Email validates email format using regex
var Email = validation.NewStringRuleWithError(
	func(s string) bool {
		return emailRegex.MatchString(s)
	},
	validation.NewError("validation_email_format", "must be a valid email address"),
)

// NoWhitespace validates that string doesn't contain leading/trailing whitespace
var NoWhitespace = validation.NewStringRuleWithError(
	func(s string) bool {
		return s == strings.TrimSpace(s)
	},
	validation.NewError("validation_no_whitespace", "must not contain leading or trailing whitespace"),
)

// NotBlank validates that a string is not empty after trimming whitespace
var NotBlank = validation.NewStringRuleWithError(
	func(s string) bool {
		return strings.TrimSpace(s) != ""
	},
	validation.NewError("validation_not_blank", "must not be blank"),
)
