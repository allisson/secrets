package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TokenizationKeyRepository defines the interface for tokenization key persistence.
type TokenizationKeyRepository interface {
	Create(ctx context.Context, key *TokenizationKey) error
	Delete(ctx context.Context, name string) error
	Get(ctx context.Context, keyID uuid.UUID) (*TokenizationKey, error)
	GetByName(ctx context.Context, name string) (*TokenizationKey, error)
	GetByNameAndVersion(
		ctx context.Context,
		name string,
		version uint,
	) (*TokenizationKey, error)

	// ListCursor retrieves tokenization keys ordered by name ascending with cursor-based pagination.
	// If afterName is provided, returns keys with name greater than afterName (ASC order).
	// Returns the latest version for each key. Filters out soft-deleted keys.
	// Returns empty slice if no keys found. Limit is pre-validated (1-1000).
	ListCursor(
		ctx context.Context,
		afterName *string,
		limit int,
	) ([]*TokenizationKey, error)

	// HardDelete permanently removes soft-deleted tokenization keys older than the specified time.
	// It must also cascade the deletion to any associated tokens in the tokenization_tokens table.
	// Only affects keys where deleted_at IS NOT NULL.
	// If dryRun is true, returns count of keys without performing deletion.
	// Returns the number of keys that were (or would be) deleted.
	HardDelete(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}

// TokenRepository defines the interface for token mapping persistence.
type TokenRepository interface {
	Create(ctx context.Context, token *Token) error
	CreateBatch(ctx context.Context, tokens []*Token) error
	GetByToken(ctx context.Context, token string) (*Token, error)
	GetBatchByTokens(ctx context.Context, tokens []string) ([]*Token, error)
	GetByValueHash(ctx context.Context, keyID uuid.UUID, valueHash string) (*Token, error)
	Revoke(ctx context.Context, token string) error

	// DeleteExpired deletes tokens that expired before the specified timestamp.
	// Returns the number of deleted tokens. Uses transaction support via database.GetTx().
	// All timestamps are expected in UTC.
	DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error)

	// CountExpired counts tokens that expired before the specified timestamp without deleting them.
	// Returns the count of matching tokens. Uses transaction support via database.GetTx().
	// All timestamps are expected in UTC.
	CountExpired(ctx context.Context, olderThan time.Time) (int64, error)
}
