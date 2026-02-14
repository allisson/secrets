// Package usecase defines interfaces and implementations for transit encryption use cases.
// Provides versioned encryption/decryption operations with automatic key rotation support.
package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// DekRepository defines the interface for DEK persistence operations.
type DekRepository interface {
	// Create stores a new DEK in the repository using transaction support from context.
	Create(ctx context.Context, dek *cryptoDomain.Dek) error

	// Get retrieves a DEK by its ID. Returns ErrDekNotFound if not found.
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

// TransitKeyRepository defines the interface for transit key persistence.
type TransitKeyRepository interface {
	// Create stores a new transit key in the repository using transaction support from context.
	Create(ctx context.Context, transitKey *transitDomain.TransitKey) error

	// Delete soft deletes a transit key by marking it with DeletedAt timestamp.
	Delete(ctx context.Context, transitKeyID uuid.UUID) error

	// GetByName retrieves the latest version of a transit key by name. Returns ErrTransitKeyNotFound if not found.
	GetByName(ctx context.Context, name string) (*transitDomain.TransitKey, error)

	// GetByNameAndVersion retrieves a specific version of a transit key. Returns ErrTransitKeyNotFound if not found.
	GetByNameAndVersion(ctx context.Context, name string, version uint) (*transitDomain.TransitKey, error)
}

// TransitKeyUseCase defines the interface for transit encryption operations.
type TransitKeyUseCase interface {
	// Create generates a new transit key with version 1 and an associated DEK for encryption.
	// The transit key name must be unique. Returns the created transit key.
	Create(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)

	// Rotate creates a new version of an existing transit key by incrementing the version number.
	// Generates a new DEK for the new version while preserving old versions for decryption.
	Rotate(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)

	// Delete soft deletes a transit key and all its versions by transit key ID.
	Delete(ctx context.Context, transitKeyID uuid.UUID) error

	// Encrypt encrypts plaintext using the latest version of the named transit key.
	// Returns an EncryptedBlob with format "version:base64-ciphertext" for storage or transmission.
	Encrypt(ctx context.Context, name string, plaintext []byte) (*transitDomain.EncryptedBlob, error)

	// Decrypt decrypts ciphertext using the version specified in the encrypted blob.
	// The ciphertext parameter should be in format "version:base64-ciphertext".
	//
	// Security Note: The returned EncryptedBlob contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling cryptoDomain.Zero(blob.Plaintext).
	Decrypt(ctx context.Context, name string, ciphertext string) (*transitDomain.EncryptedBlob, error)
}
