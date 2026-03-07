// Package usecase defines interfaces and implementations for tokenization use cases.
// Provides format-preserving token generation with configurable deterministic behavior and full lifecycle management.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// DekRepository defines the interface for DEK persistence operations.
type DekRepository interface {
	Create(ctx context.Context, dek *cryptoDomain.Dek) error
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

// TokenizationKeyRepository defines the interface for tokenization key persistence.
type TokenizationKeyRepository interface {
	Create(ctx context.Context, key *tokenizationDomain.TokenizationKey) error
	Delete(ctx context.Context, keyID uuid.UUID) error
	Get(ctx context.Context, keyID uuid.UUID) (*tokenizationDomain.TokenizationKey, error)
	GetByName(ctx context.Context, name string) (*tokenizationDomain.TokenizationKey, error)
	GetByNameAndVersion(
		ctx context.Context,
		name string,
		version uint,
	) (*tokenizationDomain.TokenizationKey, error)

	// ListCursor retrieves tokenization keys ordered by name ascending with cursor-based pagination.
	// If afterName is provided, returns keys with name greater than afterName (ASC order).
	// Returns the latest version for each key. Filters out soft-deleted keys.
	// Returns empty slice if no keys found. Limit is pre-validated (1-1000).
	ListCursor(
		ctx context.Context,
		afterName *string,
		limit int,
	) ([]*tokenizationDomain.TokenizationKey, error)

	// HardDelete permanently removes soft-deleted tokenization keys older than the specified time.
	// It must also cascade the deletion to any associated tokens in the tokenization_tokens table.
	// Only affects keys where deleted_at IS NOT NULL.
	// If dryRun is true, returns count of keys without performing deletion.
	// Returns the number of keys that were (or would be) deleted.
	HardDelete(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}

// TokenRepository defines the interface for token mapping persistence.
type TokenRepository interface {
	Create(ctx context.Context, token *tokenizationDomain.Token) error
	GetByToken(ctx context.Context, token string) (*tokenizationDomain.Token, error)
	GetByValueHash(ctx context.Context, keyID uuid.UUID, valueHash string) (*tokenizationDomain.Token, error)
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

// TokenizationKeyUseCase defines the interface for tokenization key management operations.
type TokenizationKeyUseCase interface {
	// Create generates a new tokenization key with version 1 and an associated DEK.
	// The key name must be unique.
	Create(
		ctx context.Context,
		name string,
		formatType tokenizationDomain.FormatType,
		isDeterministic bool,
		alg cryptoDomain.Algorithm,
	) (*tokenizationDomain.TokenizationKey, error)

	// Rotate creates a new version of an existing tokenization key by incrementing the version number.
	// Generates a new DEK for the new version while preserving old versions for detokenization.
	Rotate(
		ctx context.Context,
		name string,
		formatType tokenizationDomain.FormatType,
		isDeterministic bool,
		alg cryptoDomain.Algorithm,
	) (*tokenizationDomain.TokenizationKey, error)

	// Delete soft deletes a tokenization key and all its versions by key ID.
	Delete(ctx context.Context, keyID uuid.UUID) error

	// GetByName retrieves a single tokenization key by its name.
	// Returns the latest version for the key. Filters out soft-deleted keys.
	GetByName(ctx context.Context, name string) (*tokenizationDomain.TokenizationKey, error)

	// ListCursor retrieves tokenization keys ordered by name ascending with cursor-based pagination.
	// If afterName is provided, returns keys with name greater than afterName (ASC order).
	// Returns the latest version for each key. Filters out soft-deleted keys.
	// Returns empty slice if no keys found. Limit is pre-validated (1-1000).
	ListCursor(
		ctx context.Context,
		afterName *string,
		limit int,
	) ([]*tokenizationDomain.TokenizationKey, error)

	// PurgeDeleted permanently removes soft-deleted tokenization keys older than specified days.
	// It also removes all tokens associated with those keys.
	// If dryRun is true, returns count of keys without performing deletion.
	// Returns the number of keys that were (or would be) deleted.
	PurgeDeleted(ctx context.Context, olderThanDays int, dryRun bool) (int64, error)
}

// TokenizationUseCase defines the interface for token generation and management operations.
type TokenizationUseCase interface {
	// Tokenize generates a token for the given plaintext value using the latest version of the named key.
	// In deterministic mode, returns the existing token if the value has been tokenized before.
	// Metadata is optional display data (e.g., last 4 digits, expiry date) stored unencrypted.
	Tokenize(
		ctx context.Context,
		keyName string,
		plaintext []byte,
		metadata map[string]any,
		expiresAt *time.Time,
	) (*tokenizationDomain.Token, error)

	// Detokenize retrieves the original plaintext value for a given token.
	// Returns ErrTokenNotFound if token doesn't exist, ErrTokenExpired if expired, ErrTokenRevoked if revoked.
	// Security Note: Callers MUST zero the returned plaintext after use: cryptoDomain.Zero(plaintext).
	Detokenize(ctx context.Context, token string) (plaintext []byte, metadata map[string]any, err error)

	// Validate checks if a token exists and is valid (not expired or revoked).
	Validate(ctx context.Context, token string) (bool, error)

	// Revoke marks a token as revoked, preventing further detokenization.
	Revoke(ctx context.Context, token string) error

	// CleanupExpired deletes tokens that expired more than the specified number of days ago.
	// Returns the number of deleted tokens. Use dryRun=true to preview count without deletion.
	CleanupExpired(ctx context.Context, days int, dryRun bool) (int64, error)
}
