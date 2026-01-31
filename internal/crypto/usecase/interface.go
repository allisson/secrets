// Package usecase defines the business logic interfaces for cryptographic operations.
//
// This package contains interface definitions for repositories and use cases
// related to envelope encryption and key management. Implementations of these
// interfaces handle KEK and DEK management, key rotation, and encryption/decryption.
package usecase

import (
	"context"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// KekRepository defines the interface for Key Encryption Key persistence.
//
// This interface abstracts KEK storage operations, allowing different
// implementations for PostgreSQL, MySQL, or other data stores. It supports
// transaction-aware operations and key rotation workflows.
type KekRepository interface {
	// Create stores a new KEK in the repository
	Create(ctx context.Context, kek *cryptoDomain.Kek) error

	// Update modifies an existing KEK (typically used for deactivation during rotation)
	Update(ctx context.Context, kek *cryptoDomain.Kek) error

	// List retrieves all KEKs ordered by version descending
	List(ctx context.Context) ([]*cryptoDomain.Kek, error)
}
