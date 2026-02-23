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

	// ErrClientLocked indicates the client is temporarily locked due to too many failed
	// authentication attempts. The client must wait for the lockout period to expire.
	ErrClientLocked = errors.Wrap(errors.ErrLocked, "client is locked")

	// ErrSignatureInvalid indicates the audit log HMAC signature verification failed.
	// This typically means the audit log data has been tampered with after creation.
	ErrSignatureInvalid = errors.Wrap(errors.ErrInvalidInput, "audit log signature is invalid")

	// ErrSignatureMissing indicates the audit log does not have a cryptographic signature.
	// This is expected for legacy logs created before signature implementation.
	ErrSignatureMissing = errors.Wrap(errors.ErrNotFound, "audit log signature is missing")

	// ErrKekNotFoundForLog indicates the KEK referenced by an audit log signature
	// was not found in the KEK chain. This should not occur if KEK retention policy
	// is properly enforced (ON DELETE RESTRICT constraint).
	ErrKekNotFoundForLog = errors.Wrap(
		errors.ErrNotFound,
		"kek not found for audit log signature verification",
	)
)
