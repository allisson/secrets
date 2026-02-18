// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"fmt"

	validation "github.com/jellydator/validation"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// CreateTokenizationKeyRequest contains the parameters for creating a new tokenization key.
type CreateTokenizationKeyRequest struct {
	Name            string `json:"name"`
	FormatType      string `json:"format_type"`      // "uuid", "numeric", "luhn-preserving", "alphanumeric"
	IsDeterministic bool   `json:"is_deterministic"` // If true, same value produces same token
	Algorithm       string `json:"algorithm"`        // "aes-gcm" or "chacha20-poly1305"
}

// Validate checks if the create tokenization key request is valid.
func (r *CreateTokenizationKeyRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 255),
		),
		validation.Field(&r.FormatType,
			validation.Required,
			customValidation.NotBlank,
			validation.By(validateFormatType),
		),
		validation.Field(&r.Algorithm,
			validation.Required,
			customValidation.NotBlank,
			validation.By(validateAlgorithm),
		),
	)
}

// RotateTokenizationKeyRequest contains the parameters for rotating a tokenization key.
type RotateTokenizationKeyRequest struct {
	FormatType      string `json:"format_type"`      // "uuid", "numeric", "luhn-preserving", "alphanumeric"
	IsDeterministic bool   `json:"is_deterministic"` // If true, same value produces same token
	Algorithm       string `json:"algorithm"`        // "aes-gcm" or "chacha20-poly1305"
}

// Validate checks if the rotate tokenization key request is valid.
func (r *RotateTokenizationKeyRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.FormatType,
			validation.Required,
			customValidation.NotBlank,
			validation.By(validateFormatType),
		),
		validation.Field(&r.Algorithm,
			validation.Required,
			customValidation.NotBlank,
			validation.By(validateAlgorithm),
		),
	)
}

// TokenizeRequest contains the parameters for tokenizing a value.
type TokenizeRequest struct {
	Plaintext string         `json:"plaintext"` // Base64-encoded plaintext
	Metadata  map[string]any `json:"metadata,omitempty"`
	TTL       *int           `json:"ttl,omitempty"` // Time-to-live in seconds (optional)
}

// Validate checks if the tokenize request is valid.
func (r *TokenizeRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Plaintext,
			validation.Required,
			customValidation.NotBlank,
			customValidation.Base64,
		),
		validation.Field(&r.TTL,
			validation.When(r.TTL != nil, validation.Min(1)),
		),
	)
}

// DetokenizeRequest contains the parameters for detokenizing a value.
type DetokenizeRequest struct {
	Token string `json:"token"`
}

// Validate checks if the detokenize request is valid.
func (r *DetokenizeRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required,
			customValidation.NotBlank,
		),
	)
}

// ValidateTokenRequest contains the parameters for validating a token.
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// Validate checks if the validate token request is valid.
func (r *ValidateTokenRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required,
			customValidation.NotBlank,
		),
	)
}

// RevokeTokenRequest contains the parameters for revoking a token.
type RevokeTokenRequest struct {
	Token string `json:"token"`
}

// Validate checks if the revoke token request is valid.
func (r *RevokeTokenRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required,
			customValidation.NotBlank,
		),
	)
}

// validateFormatType validates that the format type is supported.
func validateFormatType(value interface{}) error {
	formatType, ok := value.(string)
	if !ok {
		return validation.NewError("validation_format_type", "must be a string")
	}

	_, err := ParseFormatType(formatType)
	return err
}

// ParseFormatType converts a string to a tokenizationDomain.FormatType.
// Returns an error if the format type is not supported.
func ParseFormatType(formatType string) (tokenizationDomain.FormatType, error) {
	switch formatType {
	case "uuid":
		return tokenizationDomain.FormatUUID, nil
	case "numeric":
		return tokenizationDomain.FormatNumeric, nil
	case "luhn-preserving":
		return tokenizationDomain.FormatLuhnPreserving, nil
	case "alphanumeric":
		return tokenizationDomain.FormatAlphanumeric, nil
	default:
		return "", fmt.Errorf(
			"invalid format type: must be 'uuid', 'numeric', 'luhn-preserving', or 'alphanumeric'",
		)
	}
}

// validateAlgorithm validates that the algorithm is supported.
func validateAlgorithm(value interface{}) error {
	alg, ok := value.(string)
	if !ok {
		return validation.NewError("validation_algorithm_type", "must be a string")
	}

	_, err := ParseAlgorithm(alg)
	return err
}

// ParseAlgorithm converts a string to a cryptoDomain.Algorithm.
// Returns an error if the algorithm is not supported.
func ParseAlgorithm(alg string) (cryptoDomain.Algorithm, error) {
	switch alg {
	case "aes-gcm":
		return cryptoDomain.AESGCM, nil
	case "chacha20-poly1305":
		return cryptoDomain.ChaCha20, nil
	default:
		return "", fmt.Errorf("invalid algorithm: must be 'aes-gcm' or 'chacha20-poly1305'")
	}
}
