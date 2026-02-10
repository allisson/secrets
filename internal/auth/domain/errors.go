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

	// ErrPolicyNotFound indicates a policy with the specified name was not found.
	ErrPolicyNotFound = errors.Wrap(errors.ErrNotFound, "policy not found")

	// ErrClientPoliciesNotFound indicates a client-policy relationship was not found.
	ErrClientPoliciesNotFound = errors.Wrap(errors.ErrNotFound, "client policies not found")
)
