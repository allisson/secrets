package domain

import (
	"context"
	"time"
)

// TransitKeyRepository defines the interface for transit key persistence.
type TransitKeyRepository interface {
	// Create stores a new transit key in the repository using transaction support from context.
	Create(ctx context.Context, transitKey *TransitKey) error

	// Delete soft deletes all versions of a transit key by name, marking them with DeletedAt timestamp.
	Delete(ctx context.Context, name string) error

	// GetByName retrieves the latest version of a transit key by name. Returns ErrTransitKeyNotFound if not found.
	GetByName(ctx context.Context, name string) (*TransitKey, error)

	// GetByNameAndVersion retrieves a specific version of a transit key. Returns ErrTransitKeyNotFound if not found.
	GetByNameAndVersion(ctx context.Context, name string, version uint) (*TransitKey, error)

	// GetTransitKey retrieves a transit key version by name and optional version (0 for latest),
	// including its associated encryption algorithm (populated in TransitKey.Algorithm).
	// Returns ErrTransitKeyNotFound if not found.
	GetTransitKey(ctx context.Context, name string, version uint) (*TransitKey, error)

	// ListCursor retrieves transit keys ordered by name ascending with cursor-based pagination.
	// If afterName is provided, returns keys with name greater than afterName (ASC order).
	// Returns the latest version for each key. Filters out soft-deleted keys.
	// Returns empty slice if no keys found. Limit is pre-validated (1-1000).
	ListCursor(ctx context.Context, afterName *string, limit int) ([]*TransitKey, error)

	// HardDelete permanently removes soft-deleted transit keys older than the specified time.
	// Only affects keys where deleted_at IS NOT NULL.
	// If dryRun is true, returns count without performing deletion.
	// Returns the number of keys that were (or would be) deleted.
	HardDelete(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}
