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

// Token length constraints
const (
	// MaxTokenLength is the maximum allowed token length for format-preserving tokens.
	// This limit applies to Numeric, Luhn-Preserving, and Alphanumeric formats.
	MaxTokenLength = 255

	// MinLuhnTokenLength is the minimum token length required for Luhn algorithm validation.
	// Luhn check requires at least 2 digits (payload + check digit).
	MinLuhnTokenLength = 2

	// MaxPlaintextSize is the maximum allowed plaintext size for tokenization (64 KB).
	// This limit prevents DoS attacks from extremely large inputs and ensures reasonable
	// encryption performance.
	MaxPlaintextSize = 65536 // 64 KB
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
