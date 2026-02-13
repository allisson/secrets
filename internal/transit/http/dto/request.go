// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"fmt"

	validation "github.com/jellydator/validation"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// CreateTransitKeyRequest contains the parameters for creating a new transit key.
type CreateTransitKeyRequest struct {
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"` // "aes-gcm" or "chacha20-poly1305"
}

// Validate checks if the create transit key request is valid.
func (r *CreateTransitKeyRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 255),
		),
		validation.Field(&r.Algorithm,
			validation.Required,
			customValidation.NotBlank,
			validation.By(validateAlgorithm),
		),
	)
}

// RotateTransitKeyRequest contains the parameters for rotating a transit key.
type RotateTransitKeyRequest struct {
	Algorithm string `json:"algorithm"` // "aes-gcm" or "chacha20-poly1305"
}

// Validate checks if the rotate transit key request is valid.
func (r *RotateTransitKeyRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Algorithm,
			validation.Required,
			customValidation.NotBlank,
			validation.By(validateAlgorithm),
		),
	)
}

// EncryptRequest contains the parameters for encrypting data.
type EncryptRequest struct {
	Plaintext string `json:"plaintext"` // Base64-encoded plaintext
}

// Validate checks if the encrypt request is valid.
func (r *EncryptRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Plaintext,
			validation.Required,
			customValidation.NotBlank,
			customValidation.Base64,
		),
	)
}

// DecryptRequest contains the parameters for decrypting data.
type DecryptRequest struct {
	Ciphertext string `json:"ciphertext"` // Format: "version:base64-ciphertext"
}

// Validate checks if the decrypt request is valid.
func (r *DecryptRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Ciphertext,
			validation.Required,
			customValidation.NotBlank,
		),
	)
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
