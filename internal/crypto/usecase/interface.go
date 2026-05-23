// Package usecase defines business logic interfaces for KEK operations.
//
// DEK envelope encryption now lives in internal/keyring. This package only
// retains KEK lifecycle operations driven by the rotation CLI.
package usecase

import (
	"context"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// KekRepository defines persistence operations for Key Encryption Keys.
// Implementations must support transaction-aware operations via context propagation.
type KekRepository interface {
	Create(ctx context.Context, kek *cryptoDomain.Kek) error
	Update(ctx context.Context, kek *cryptoDomain.Kek) error
	List(ctx context.Context) ([]*cryptoDomain.Kek, error)
}

// KekUseCase defines business logic operations for Key Encryption Key management.
type KekUseCase interface {
	Create(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error
	Rotate(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error
	Unwrap(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain) (*cryptoDomain.KekChain, error)
}
