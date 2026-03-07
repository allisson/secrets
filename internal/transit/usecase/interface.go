// Package usecase defines interfaces and implementations for transit encryption use cases.
// Provides versioned encryption/decryption operations with automatic key rotation support.
package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// Re-export repository interfaces for convenience and backward compatibility if needed.
// However, the canonical location is now internal/transit/domain/repository.go.
type DekRepository = transitDomain.DekRepository
type TransitKeyRepository = transitDomain.TransitKeyRepository

// TransitKeyUseCase defines the interface for transit encryption operations.
type TransitKeyUseCase interface {
	// Create generates a new transit key with version 1 and an associated DEK for encryption.
	// The transit key name must be unique. Returns the created transit key.
	Create(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)

	// Rotate creates a new version of an existing transit key by incrementing the version number.
	// Generates a new DEK for the new version while preserving old versions for decryption.
	Rotate(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)

	// Get retrieves transit key metadata (including its algorithm) by name and optional version.
	// If version is 0, the latest version is retrieved.
	Get(ctx context.Context, name string, version uint) (*transitDomain.TransitKey, cryptoDomain.Algorithm, error)

	// Delete soft deletes a transit key and all its versions by transit key ID.
	Delete(ctx context.Context, transitKeyID uuid.UUID) error

	// Encrypt encrypts plaintext using the latest version of the named transit key.
	// Optional context (AAD) can be provided for additional security.
	// Returns an EncryptedBlob with format "version:base64-ciphertext" for storage or transmission.
	Encrypt(ctx context.Context, name string, plaintext, context []byte) (*transitDomain.EncryptedBlob, error)

	// Decrypt decrypts ciphertext using the version specified in the encrypted blob.
	// Optional context (AAD) MUST match the one used during encryption.
	// The ciphertext parameter should be in format "version:base64-ciphertext".
	//
	// Security Note: The returned EncryptedBlob contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling cryptoDomain.Zero(blob.Plaintext).
	Decrypt(
		ctx context.Context,
		name string,
		ciphertext string,
		context []byte,
	) (*transitDomain.EncryptedBlob, error)

	// ListCursor retrieves transit keys ordered by name ascending with cursor-based pagination.
	// If afterName is provided, returns keys with name greater than afterName (ASC order).
	// Returns the latest version for each key. Filters out soft-deleted keys.
	// Returns empty slice if no keys found. Limit is pre-validated (1-1000).
	ListCursor(ctx context.Context, afterName *string, limit int) ([]*transitDomain.TransitKey, error)

	// PurgeDeleted permanently removes soft-deleted transit keys older than specified days.
	// If dryRun is true, returns count without performing deletion.
	// Returns the number of keys that were (or would be) deleted.
	PurgeDeleted(ctx context.Context, olderThanDays int, dryRun bool) (int64, error)
}
