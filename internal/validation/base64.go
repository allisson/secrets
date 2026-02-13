// Package validation provides custom validation rules for the application.
package validation

import (
	"encoding/base64"

	validation "github.com/jellydator/validation"
)

// Base64 validates that a string is valid base64-encoded data.
var Base64 = validation.By(func(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return validation.NewError("validation_base64_type", "must be a string")
	}
	if s == "" {
		return nil // Let Required handle empty strings
	}
	_, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return validation.NewError("validation_base64", "must be valid base64-encoded data")
	}
	return nil
})
