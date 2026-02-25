// Package usecase defines the interfaces and implementations for secret management use cases.
// Use cases orchestrate operations between repositories and services to implement business
// logic for managing encrypted secrets with automatic versioning.
package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// DekRepository defines the interface for Data Encryption Key persistence operations.
type DekRepository interface {
	// Create stores a new DEK in the repository using transaction support from context.
	Create(ctx context.Context, dek *cryptoDomain.Dek) error

	// Get retrieves a DEK by its ID. Returns ErrDekNotFound if not found.
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

// SecretRepository defines the interface for Secret persistence operations.
type SecretRepository interface {
	// Create stores a new secret in the repository using transaction support from context.
	Create(ctx context.Context, secret *secretsDomain.Secret) error

	// Delete soft deletes a secret by marking it with DeletedAt timestamp.
	Delete(ctx context.Context, secretID uuid.UUID) error

	// GetByPath retrieves the latest version of a secret by its path. Returns ErrSecretNotFound if not found.
	GetByPath(ctx context.Context, path string) (*secretsDomain.Secret, error)

	// GetByPathAndVersion retrieves a specific version of a secret. Returns ErrSecretNotFound if not found.
	GetByPathAndVersion(ctx context.Context, path string, version uint) (*secretsDomain.Secret, error)

	// List retrieves secrets ordered by path ascending with pagination.
	// Returns the latest version for each secret. Uses offset and limit for pagination.
	List(ctx context.Context, offset, limit int) ([]*secretsDomain.Secret, error)
}

// SecretUseCase defines the interface for secret management business logic.
type SecretUseCase interface {
	// CreateOrUpdate creates a new secret or increments the version if path exists.
	// Encrypts the value with a new DEK for each version. Returns the created/updated secret.
	CreateOrUpdate(ctx context.Context, path string, value []byte) (*secretsDomain.Secret, error)

	// Get retrieves and decrypts a secret by its path (latest version).
	//
	// Security Note: The returned Secret contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling cryptoDomain.Zero(secret.Plaintext).
	Get(ctx context.Context, path string) (*secretsDomain.Secret, error)

	// GetByVersion retrieves and decrypts a secret by its path and specific version.
	//
	// Security Note: The returned Secret contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling cryptoDomain.Zero(secret.Plaintext).
	GetByVersion(ctx context.Context, path string, version uint) (*secretsDomain.Secret, error)

	// Delete soft deletes all versions of a secret by path, marking them with DeletedAt timestamp.
	// Preserves encrypted data for audit purposes while preventing future access.
	Delete(ctx context.Context, path string) error

	// List retrieves secrets without their values, ordered by path with pagination.
	// Returns empty slice if no secrets found.
	List(ctx context.Context, offset, limit int) ([]*secretsDomain.Secret, error)
}
