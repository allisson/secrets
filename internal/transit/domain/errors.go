// Package domain defines the transit encryption domain models and types.
package domain

import (
	"github.com/allisson/secrets/internal/errors"
)

// Transit encryption error definitions.
//
// These domain-specific errors wrap standard errors from internal/errors
// to provide context for transit encryption failures.
var (
	// ErrInvalidBlobFormat indicates the encrypted blob format is invalid.
	//
	// The expected format is: "version:ciphertext-base64"
	// This error is returned when the input string doesn't have exactly 2 parts
	// separated by colons.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrInvalidBlobFormat = errors.Wrap(errors.ErrInvalidInput, "invalid encrypted blob format")

	// ErrInvalidBlobVersion indicates the version string cannot be parsed.
	//
	// The version must be a valid non-negative integer that fits in a uint.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrInvalidBlobVersion = errors.Wrap(errors.ErrInvalidInput, "invalid encrypted blob version")

	// ErrInvalidBlobBase64 indicates the ciphertext is not valid base64.
	//
	// The ciphertext must be a valid base64-encoded string using standard encoding.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrInvalidBlobBase64 = errors.Wrap(errors.ErrInvalidInput, "invalid encrypted blob base64")

	// ErrTransitKeyNotFound indicates the transit key was not found.
	//
	// This error is returned when attempting to retrieve a transit key by name
	// that either doesn't exist or has been soft-deleted.
	//
	// HTTP Status: 404 Not Found
	ErrTransitKeyNotFound = errors.Wrap(errors.ErrNotFound, "transit key not found")
)
