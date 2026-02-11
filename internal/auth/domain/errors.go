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
)
