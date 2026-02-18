// Package domain defines core tokenization domain models for data tokenization.
// Supports multiple token formats (UUID, Numeric, Luhn-preserving, Alphanumeric) with configurable deterministic behavior.
package domain

import (
	"errors"
)

// FormatType defines the token format type.
type FormatType string

const (
	FormatUUID           FormatType = "uuid"
	FormatNumeric        FormatType = "numeric"
	FormatLuhnPreserving FormatType = "luhn-preserving"
	FormatAlphanumeric   FormatType = "alphanumeric"
)

// Validate checks if the format type is valid.
func (f FormatType) Validate() error {
	switch f {
	case FormatUUID, FormatNumeric, FormatLuhnPreserving, FormatAlphanumeric:
		return nil
	default:
		return errors.New("invalid format type")
	}
}

// String returns the string representation of the format type.
func (f FormatType) String() string {
	return string(f)
}
