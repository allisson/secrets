package service

import (
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// NewTokenGenerator creates a new token generator based on the specified format type.
func NewTokenGenerator(formatType tokenizationDomain.FormatType) (TokenGenerator, error) {
	switch formatType {
	case tokenizationDomain.FormatUUID:
		return NewUUIDGenerator(), nil
	case tokenizationDomain.FormatNumeric:
		return NewNumericGenerator(), nil
	case tokenizationDomain.FormatLuhnPreserving:
		return NewLuhnGenerator(), nil
	case tokenizationDomain.FormatAlphanumeric:
		return NewAlphanumericGenerator(), nil
	default:
		return nil, tokenizationDomain.ErrInvalidFormatType
	}
}
