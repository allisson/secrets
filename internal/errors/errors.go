// Package errors provides standardized domain errors for business logic.
package errors

import (
	"errors"
	"fmt"
)

// Standard domain errors that can be used across all domain modules.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict indicates a conflict with existing data.
	ErrConflict = errors.New("conflict")

	// ErrInvalidInput indicates the input data is invalid or fails validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates missing or invalid authentication credentials.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates insufficient permissions.
	ErrForbidden = errors.New("forbidden")

	// ErrLocked indicates the resource is temporarily locked.
	ErrLocked = errors.New("locked")
)

// New creates a new error with the given message.
func New(message string) error {
	return errors.New(message)
}

// Wrap wraps an error with additional context while preserving the error chain.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Is reports whether any error in err's tree matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target.
func As(err error, target any) bool {
	return errors.As(err, target)
}
