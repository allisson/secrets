// Package usecase defines the interfaces and implementations for secret management use cases.
// Use cases orchestrate operations between the keyring and the secret repository to
// implement business logic for managing encrypted secrets with automatic versioning.
package usecase

import (
	"context"
	"time"

	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// SecretRepository defines the interface for Secret persistence operations.
type SecretRepository interface {
	// Create stores a new secret in the repository using transaction support from context.
	Create(ctx context.Context, secret *secretsDomain.Secret) error

	// Delete soft deletes all versions of a secret by path, marking them with DeletedAt timestamp.
	Delete(ctx context.Context, path string) error

	// GetByPath retrieves the latest version of a secret by its path. Returns ErrSecretNotFound if not found.
	GetByPath(ctx context.Context, path string) (*secretsDomain.Secret, error)

	// GetByPathAndVersion retrieves a specific version of a secret. Returns ErrSecretNotFound if not found.
	GetByPathAndVersion(ctx context.Context, path string, version uint) (*secretsDomain.Secret, error)

	// ListCursor retrieves secrets ordered by path ascending with cursor-based pagination.
	// If afterPath is provided, returns secrets with path greater than afterPath (ASC order).
	// Returns the latest version for each secret. Filters out soft-deleted secrets.
	// Returns empty slice if no secrets found. Limit is pre-validated (1-1000).
	ListCursor(ctx context.Context, afterPath *string, limit int) ([]*secretsDomain.Secret, error)

	// HardDelete permanently removes soft-deleted secrets older than the specified time.
	// Only affects secrets where deleted_at IS NOT NULL.
	// If dryRun is true, returns count without performing deletion.
	// Returns the number of secrets that were (or would be) deleted.
	HardDelete(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}

// SecretUseCase defines the interface for secret management business logic.
type SecretUseCase interface {
	// CreateOrUpdate creates a new secret or increments the version if path exists.
	// Encrypts the value with a fresh DEK via the keyring on each call.
	CreateOrUpdate(ctx context.Context, path string, value []byte) (*secretsDomain.Secret, error)

	// Get retrieves and decrypts a secret by its path (latest version).
	//
	// Security Note: The returned Secret contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling keyring.Zero(secret.Plaintext).
	Get(ctx context.Context, path string) (*secretsDomain.Secret, error)

	// GetByVersion retrieves and decrypts a secret by its path and specific version.
	//
	// Security Note: The returned Secret contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling keyring.Zero(secret.Plaintext).
	GetByVersion(ctx context.Context, path string, version uint) (*secretsDomain.Secret, error)

	// Delete soft deletes all versions of a secret by path, marking them with DeletedAt timestamp.
	// Preserves encrypted data for audit purposes while preventing future access.
	Delete(ctx context.Context, path string) error

	// ListCursor retrieves secrets ordered by path ascending with cursor-based pagination.
	// If afterPath is provided, returns secrets with path greater than afterPath (ASC order).
	// Returns secrets without their values. Filters out soft-deleted secrets.
	// Returns empty slice if no secrets found. Limit is pre-validated (1-1000).
	ListCursor(ctx context.Context, afterPath *string, limit int) ([]*secretsDomain.Secret, error)

	// PurgeDeleted permanently removes soft-deleted secrets older than specified days.
	// If dryRun is true, returns count without performing deletion.
	// Returns the number of secrets that were (or would be) deleted.
	PurgeDeleted(ctx context.Context, olderThanDays int, dryRun bool) (int64, error)
}
