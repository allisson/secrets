package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ClientRepository defines persistence operations for authentication clients.
// Implementations must support transaction-aware operations via context propagation.
type ClientRepository interface {
	// Create stores a new client in the repository.
	Create(ctx context.Context, client *Client) error

	// Update modifies an existing client in the repository.
	Update(ctx context.Context, client *Client) error

	// Get retrieves a client by ID. Returns ErrClientNotFound if not found.
	Get(ctx context.Context, clientID uuid.UUID) (*Client, error)

	// ListCursor retrieves clients ordered by ID descending (newest first) with cursor-based pagination.
	// If afterID is provided, returns clients with ID less than afterID (DESC order).
	// Returns empty slice if no clients found. Limit is pre-validated (1-1000).
	ListCursor(ctx context.Context, afterID *uuid.UUID, limit int) ([]*Client, error)

	// UpdateLockState atomically updates the failed attempt counter and lock expiry.
	// Pass failedAttempts=0 and lockedUntil=nil to reset the lock on successful auth.
	UpdateLockState(ctx context.Context, clientID uuid.UUID, failedAttempts int, lockedUntil *time.Time) error
}

// TokenRepository defines persistence operations for authentication tokens.
// Implementations must support transaction-aware operations via context propagation.
type TokenRepository interface {
	// Create stores a new token in the repository.
	Create(ctx context.Context, token *Token) error

	// Update modifies an existing token in the repository.
	Update(ctx context.Context, token *Token) error

	// Get retrieves a token by ID. Returns ErrTokenNotFound if not found.
	Get(ctx context.Context, tokenID uuid.UUID) (*Token, error)

	// GetByTokenHash retrieves a token by its SHA-256 hash value.
	// Returns ErrTokenNotFound if no token matches the hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*Token, error)

	// RevokeByTokenID marks a specific token as revoked by setting its revoked_at timestamp.
	RevokeByTokenID(ctx context.Context, tokenID uuid.UUID) error

	// RevokeByClientID marks all active tokens for a specific client as revoked.
	RevokeByClientID(ctx context.Context, clientID uuid.UUID) error

	// PurgeExpiredAndRevoked permanently deletes tokens that are either expired or revoked
	// and were created before the specified timestamp. Returns the number of deleted tokens.
	PurgeExpiredAndRevoked(ctx context.Context, olderThan time.Time) (int64, error)
}

// AuditLogRepository defines persistence operations for audit logs.
// Implementations must support transaction-aware operations via context propagation.
type AuditLogRepository interface {
	// Create stores a new audit log entry recording an authorization decision.
	// Returns error if the audit log ID already exists or database operation fails.
	Create(ctx context.Context, auditLog *AuditLog) error

	// Get retrieves a single audit log by ID. Returns error if not found.
	// Used for signature verification of specific audit logs.
	Get(ctx context.Context, id uuid.UUID) (*AuditLog, error)

	// ListCursor retrieves audit logs ordered by created_at descending (newest first) with cursor-based pagination
	// and optional time-based filtering. If afterID is provided, returns logs with ID greater than afterID (UUIDv7 ordering).
	// Accepts createdAtFrom and createdAtTo as optional filters (nil means no filter). Both boundaries are inclusive (>= and <=).
	// Accepts clientID as an optional filter (nil means no filter).
	// All timestamps are expected in UTC. Returns empty slice if no audit logs found. Limit is pre-validated (1-1000).
	ListCursor(
		ctx context.Context,
		afterID *uuid.UUID,
		limit int,
		createdAtFrom, createdAtTo *time.Time,
		clientID *uuid.UUID,
	) ([]*AuditLog, error)

	// DeleteOlderThan removes audit logs with created_at before the specified timestamp.
	// When dryRun is true, returns count via SELECT COUNT(*) without deletion. When false,
	// executes DELETE and returns affected rows. Supports transaction-aware operations via
	// context propagation. All timestamps are expected in UTC.
	DeleteOlderThan(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}
