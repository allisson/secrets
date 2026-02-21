// Package usecase defines business logic interfaces for authentication and authorization operations.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// ClientRepository defines persistence operations for authentication clients.
// Implementations must support transaction-aware operations via context propagation.
type ClientRepository interface {
	// Create stores a new client in the repository.
	Create(ctx context.Context, client *authDomain.Client) error

	// Update modifies an existing client in the repository.
	Update(ctx context.Context, client *authDomain.Client) error

	// Get retrieves a client by ID. Returns ErrClientNotFound if not found.
	Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error)

	// List retrieves clients ordered by ID descending (newest first) with pagination.
	// Uses offset and limit for pagination control. Returns empty slice if no clients found.
	List(ctx context.Context, offset, limit int) ([]*authDomain.Client, error)
}

// TokenRepository defines persistence operations for authentication tokens.
// Implementations must support transaction-aware operations via context propagation.
type TokenRepository interface {
	// Create stores a new token in the repository.
	Create(ctx context.Context, token *authDomain.Token) error

	// Update modifies an existing token in the repository.
	Update(ctx context.Context, token *authDomain.Token) error

	// Get retrieves a token by ID. Returns ErrTokenNotFound if not found.
	Get(ctx context.Context, tokenID uuid.UUID) (*authDomain.Token, error)

	// GetByTokenHash retrieves a token by its SHA-256 hash value.
	// Returns ErrTokenNotFound if no token matches the hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*authDomain.Token, error)
}

// AuditLogRepository defines persistence operations for audit logs.
// Implementations must support transaction-aware operations via context propagation.
type AuditLogRepository interface {
	// Create stores a new audit log entry recording an authorization decision.
	// Returns error if the audit log ID already exists or database operation fails.
	Create(ctx context.Context, auditLog *authDomain.AuditLog) error

	// Get retrieves a single audit log by ID. Returns error if not found.
	// Used for signature verification of specific audit logs.
	Get(ctx context.Context, id uuid.UUID) (*authDomain.AuditLog, error)

	// List retrieves audit logs ordered by created_at descending (newest first) with pagination
	// and optional time-based filtering. Accepts createdAtFrom and createdAtTo as optional
	// filters (nil means no filter). Both boundaries are inclusive (>= and <=). All timestamps
	// are expected in UTC. Returns empty slice if no audit logs found.
	List(
		ctx context.Context,
		offset, limit int,
		createdAtFrom, createdAtTo *time.Time,
	) ([]*authDomain.AuditLog, error)

	// DeleteOlderThan removes audit logs with created_at before the specified timestamp.
	// When dryRun is true, returns count via SELECT COUNT(*) without deletion. When false,
	// executes DELETE and returns affected rows. Supports transaction-aware operations via
	// context propagation. All timestamps are expected in UTC.
	DeleteOlderThan(ctx context.Context, olderThan time.Time, dryRun bool) (int64, error)
}

// ClientUseCase defines business logic operations for managing authentication clients.
// It orchestrates client lifecycle including secret generation, policy management,
// and soft deletion while maintaining audit history.
type ClientUseCase interface {
	// Create generates a new authentication client with a cryptographically secure secret.
	// The secret is automatically generated using Argon2id hashing for secure storage.
	//
	// Returns the client ID and plain text secret. The plain secret is only returned once
	// and should be securely transmitted to the client administrator. The hashed version
	// is stored in the database for future authentication.
	//
	// Security Note: The returned PlainSecret must be transmitted securely (e.g., over TLS)
	// and never logged or stored by the caller. It should only be displayed once to the
	// client administrator during initial setup.
	Create(
		ctx context.Context,
		createClientInput *authDomain.CreateClientInput,
	) (*authDomain.CreateClientOutput, error)

	// Update modifies an existing client's configuration including name, active status,
	// and authorization policies. The client ID and secret remain unchanged.
	//
	// Only the fields in UpdateClientInput are modified. The client's secret, ID, and
	// creation timestamp are preserved. To disable a client's access, set IsActive to false.
	//
	// Returns ErrClientNotFound if the specified client doesn't exist.
	Update(ctx context.Context, clientID uuid.UUID, updateClientInput *authDomain.UpdateClientInput) error

	// Get retrieves a client by ID including its hashed secret and authorization policies.
	// The returned Client contains the hashed secret, not the plain text version.
	//
	// Returns ErrClientNotFound if the specified client doesn't exist.
	Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error)

	// List retrieves clients ordered by ID descending (newest first) with pagination.
	// Uses offset and limit for pagination control. Returns empty slice if no clients found.
	List(ctx context.Context, offset, limit int) ([]*authDomain.Client, error)

	// Delete performs a soft delete by setting IsActive to false, preventing authentication
	// while preserving the client record for audit purposes. The client's data remains in
	// the database but the client cannot authenticate until reactivated via Update.
	//
	// Returns ErrClientNotFound if the specified client doesn't exist.
	Delete(ctx context.Context, clientID uuid.UUID) error
}

// TokenUseCase defines business logic operations for token management.
// Handles token issuance with client authentication and token-based authentication validation.
type TokenUseCase interface {
	// Issue generates a new authentication token after validating client credentials.
	// Validates the client secret using Argon2id comparison and checks client is active.
	// Returns ErrInvalidCredentials for invalid credentials or inactive clients to prevent
	// enumeration attacks. Token expires based on system configuration (default 24 hours).
	Issue(
		ctx context.Context,
		issueTokenInput *authDomain.IssueTokenInput,
	) (*authDomain.IssueTokenOutput, error)

	// Authenticate validates a token hash and returns the associated client. Validates token
	// is not expired/revoked and client is active. Returns ErrInvalidCredentials for
	// invalid/expired/revoked tokens or missing clients to prevent enumeration attacks.
	// Returns ErrClientInactive if the client is not active. All time comparisons use UTC.
	Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error)
}

// AuditLogUseCase defines business logic operations for audit logging.
// Records authorization decisions for compliance and security monitoring.
type AuditLogUseCase interface {
	// Create records an authorization decision in the audit log for compliance monitoring.
	// Captures the request ID, client ID, capability, resource path, and optional metadata.
	// All audit log entries are immutable once created.
	Create(
		ctx context.Context,
		requestID uuid.UUID,
		clientID uuid.UUID,
		capability authDomain.Capability,
		path string,
		metadata map[string]any,
	) error

	// List retrieves audit logs ordered by created_at descending (newest first) with pagination
	// and optional time-based filtering. Accepts createdAtFrom and createdAtTo as optional
	// filters (nil means no filter). Both boundaries are inclusive (>= and <=). All timestamps
	// are expected in UTC. Returns empty slice if no audit logs found.
	List(
		ctx context.Context,
		offset, limit int,
		createdAtFrom, createdAtTo *time.Time,
	) ([]*authDomain.AuditLog, error)

	// DeleteOlderThan removes audit logs older than the specified number of days.
	// When dryRun is true, returns count without deletion. When false, executes DELETE
	// and returns affected rows. The cutoff date is calculated as current UTC time minus
	// the specified days.
	DeleteOlderThan(ctx context.Context, days int, dryRun bool) (int64, error)

	// VerifyIntegrity verifies the cryptographic signature of a specific audit log.
	// Returns nil if signature is valid, ErrSignatureMissing for unsigned legacy logs,
	// ErrKekNotFoundForLog if the KEK is missing from the chain, or ErrSignatureInvalid
	// if the log has been tampered with. This operation retrieves the log from the
	// repository and verifies using the KEK referenced by log.KekID.
	VerifyIntegrity(ctx context.Context, id uuid.UUID) error

	// VerifyBatch performs batch verification of audit logs within a time range.
	// Returns a detailed report including total checked, signed/unsigned counts,
	// valid/invalid counts, and IDs of logs with invalid signatures. Legacy unsigned
	// logs are counted separately and do not contribute to invalid count.
	VerifyBatch(ctx context.Context, startTime, endTime time.Time) (*VerificationReport, error)
}

// VerificationReport summarizes batch audit log verification results.
// Used by VerifyBatch to provide detailed integrity check statistics.
type VerificationReport struct {
	TotalChecked  int64       // Total number of audit logs checked
	SignedCount   int64       // Number of signed logs with signatures
	UnsignedCount int64       // Number of unsigned legacy logs
	ValidCount    int64       // Number of logs with valid signatures
	InvalidCount  int64       // Number of logs with invalid signatures
	InvalidLogs   []uuid.UUID // IDs of logs with invalid signatures (for investigation)
}
