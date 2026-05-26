package domain

import (
	"context"
	"time"
)

// SecretRepository defines the interface for Secret persistence operations.
type SecretRepository interface {
	// Create stores a new secret in the repository using transaction support from context.
	Create(ctx context.Context, secret *Secret) error

	// Delete soft deletes all versions of a secret by path, marking them with DeletedAt timestamp.
	Delete(ctx context.Context, path string) error

	// GetByPath retrieves the latest version of a secret by its path. Returns ErrSecretNotFound if not found.
	GetByPath(ctx context.Context, path string) (*Secret, error)

	// GetByPathAndVersion retrieves a specific version of a secret. Returns ErrSecretNotFound if not found.
	GetByPathAndVersion(ctx context.Context, path string, version uint) (*Secret, error)

	// ListCursor retrieves secrets ordered by path ascending with cursor-based pagination.
	// If afterPath is provided, returns secrets with path greater than afterPath (ASC order).
	// Returns the latest version for each secret. Filters out soft-deleted secrets.
	// Returns empty slice if no secrets found. Limit is pre-validated (1-1000).
	ListCursor(ctx context.Context, afterPath *string, limit int) ([]*Secret, error)

	// HardDelete permanently removes soft-deleted secrets older than the specified time.
	// Only affects secrets where deleted_at IS NOT NULL.
	// If dryRun is true, returns count without performing deletion.
	// Returns the number of secrets that were (or would be) deleted.
	HardDelete(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}
