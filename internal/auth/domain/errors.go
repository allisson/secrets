package domain

import (
	"github.com/allisson/secrets/internal/errors"
)

// Authentication and authorization errors.
var (
	// ErrClientNotFound indicates a client with the specified ID was not found.
	ErrClientNotFound = errors.Wrap(errors.ErrNotFound, "client not found")

	// ErrTokenNotFound indicates a token with the specified ID was not found.
	ErrTokenNotFound = errors.Wrap(errors.ErrNotFound, "token not found")

	// ErrInvalidCredentials indicates the provided credentials are invalid.
	// This error is returned for both non-existent clients and incorrect secrets
	// to prevent user enumeration attacks.
	ErrInvalidCredentials = errors.Wrap(errors.ErrUnauthorized, "invalid credentials")

	// ErrClientInactive indicates the client exists but is not active.
	// Inactive clients cannot authenticate or issue tokens.
	ErrClientInactive = errors.Wrap(errors.ErrForbidden, "client is inactive")
)
