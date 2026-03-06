// Package domain defines core domain models and errors for secrets.
package domain

import (
	"github.com/allisson/secrets/internal/errors"
)

// Secret-specific error definitions.
var (
	// ErrSecretNotFound indicates the secret was not found at the specified path.
	ErrSecretNotFound = errors.Wrap(errors.ErrNotFound, "secret not found")

	// ErrSecretValueTooLarge indicates the secret value exceeds the maximum allowed size.
	ErrSecretValueTooLarge = errors.Wrap(errors.ErrTooLarge, "secret value too large")

	// ErrInvalidSecretPath indicates the secret path fails validation.
	ErrInvalidSecretPath = errors.Wrap(errors.ErrInvalidInput, "invalid secret path format")
)
