// Package domain defines transit encryption domain models and errors.
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
	ErrInvalidBlobFormat = errors.Wrap(errors.ErrInvalidInput, "invalid encrypted blob format")

	// ErrInvalidBlobVersion indicates the version string cannot be parsed.
	ErrInvalidBlobVersion = errors.Wrap(errors.ErrInvalidInput, "invalid encrypted blob version")

	// ErrInvalidBlobBase64 indicates the ciphertext is not valid base64.
	ErrInvalidBlobBase64 = errors.Wrap(errors.ErrInvalidInput, "invalid encrypted blob base64")

	// ErrTransitKeyNotFound indicates the transit key was not found.
	ErrTransitKeyNotFound = errors.Wrap(errors.ErrNotFound, "transit key not found")

	// ErrTransitKeyAlreadyExists indicates a transit key with the same name and version already exists.
	ErrTransitKeyAlreadyExists = errors.Wrap(errors.ErrConflict, "transit key already exists")
)
