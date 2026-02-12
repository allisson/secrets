// Package usecase defines business logic interfaces for KEK operations and repository contracts.
package usecase

import (
	"context"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// KekRepository defines persistence operations for Key Encryption Keys.
// Implementations must support transaction-aware operations via context propagation.
type KekRepository interface {
	// Create stores a new KEK in the repository.
	Create(ctx context.Context, kek *cryptoDomain.Kek) error

	// Update modifies an existing KEK in the repository.
	Update(ctx context.Context, kek *cryptoDomain.Kek) error

	// List retrieves all KEKs ordered by version descending (newest first).
	List(ctx context.Context) ([]*cryptoDomain.Kek, error)
}

// KekUseCase defines business logic operations for Key Encryption Key management.
// It orchestrates KEK lifecycle including creation, rotation, and unwrapping.
type KekUseCase interface {
	// Create generates and persists a new KEK using the active master key.
	Create(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error

	// Rotate performs atomic KEK rotation by creating a new KEK with incremented version.
	Rotate(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error

	// Unwrap decrypts all KEKs from the database and returns them in a KekChain for in-memory use.
	Unwrap(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain) (*cryptoDomain.KekChain, error)
}
