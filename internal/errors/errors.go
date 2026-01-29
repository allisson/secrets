// Package errors provides standardized domain errors that express business intent
// rather than infrastructure details. These errors should be used by use cases
// and mapped to appropriate HTTP status codes by handlers.
package errors

import (
	"errors"
	"fmt"
)

// Standard domain errors that can be used across all domain modules.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict indicates a conflict with existing data (e.g., duplicate key).
	ErrConflict = errors.New("conflict")

	// ErrInvalidInput indicates the input data is invalid or fails validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates the request lacks valid authentication credentials.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the authenticated user doesn't have permission.
	ErrForbidden = errors.New("forbidden")
)

// New creates a new error with the given message.
// This is a convenience wrapper around errors.New for consistency.
func New(message string) error {
	return errors.New(message)
}

// Wrap wraps an error with additional context while preserving the error chain.
// Use this to add context at each layer without losing the original error type.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Is reports whether any error in err's tree matches target.
// This is a convenience wrapper around errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target.
// This is a convenience wrapper around errors.As.
func As(err error, target any) bool {
	return errors.As(err, target)
}
