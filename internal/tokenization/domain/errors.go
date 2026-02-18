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
)
