package domain

import (
	"github.com/allisson/secrets/internal/errors"
)

var (
	// ErrTokenizationKeyNotFound indicates the tokenization key was not found.
	ErrTokenizationKeyNotFound = errors.Wrap(errors.ErrNotFound, "tokenization key not found")

	// ErrTokenizationKeyAlreadyExists indicates a tokenization key with the same name and version already exists.
	ErrTokenizationKeyAlreadyExists = errors.Wrap(errors.ErrConflict, "tokenization key already exists")

	// ErrTokenNotFound indicates the token was not found.
	ErrTokenNotFound = errors.Wrap(errors.ErrNotFound, "token not found")

	// ErrTokenExpired indicates the token has expired.
	ErrTokenExpired = errors.Wrap(errors.ErrInvalidInput, "token has expired")

	// ErrTokenRevoked indicates the token has been revoked.
	ErrTokenRevoked = errors.Wrap(errors.ErrInvalidInput, "token has been revoked")

	// ErrInvalidFormatType indicates an invalid token format type was provided.
	ErrInvalidFormatType = errors.Wrap(errors.ErrInvalidInput, "invalid format type")

	// ErrInvalidTokenLength indicates the token length is invalid for the specified format.
	ErrInvalidTokenLength = errors.Wrap(errors.ErrInvalidInput, "invalid token length for format")

	// ErrValueTooLong indicates the value exceeds the maximum allowed length.
	ErrValueTooLong = errors.Wrap(errors.ErrInvalidInput, "value exceeds maximum length")

	// ErrPlaintextTooLarge indicates the plaintext exceeds maximum allowed size.
	ErrPlaintextTooLarge = errors.Wrap(errors.ErrInvalidInput, "plaintext exceeds maximum size of 64KB")

	// ErrPlaintextEmpty indicates the plaintext is empty.
	ErrPlaintextEmpty = errors.Wrap(errors.ErrInvalidInput, "plaintext cannot be empty")

	// ErrTokenLengthInvalid indicates the token length is invalid for the format.
	ErrTokenLengthInvalid = errors.Wrap(errors.ErrInvalidInput, "token length invalid for format type")

	// ErrTokenizationKeyNameEmpty indicates the tokenization key name is empty.
	ErrTokenizationKeyNameEmpty = errors.Wrap(errors.ErrInvalidInput, "tokenization key name cannot be empty")

	// ErrTokenizationKeyVersionInvalid indicates the version is invalid (must be > 0).
	ErrTokenizationKeyVersionInvalid = errors.Wrap(
		errors.ErrInvalidInput,
		"tokenization key version must be greater than 0",
	)

	// ErrTokenizationKeyDekIDInvalid indicates the DEK ID is invalid (nil UUID).
	ErrTokenizationKeyDekIDInvalid = errors.Wrap(
		errors.ErrInvalidInput,
		"tokenization key DEK ID cannot be nil",
	)
)
