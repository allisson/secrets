// Package usecase defines business logic interfaces for authentication and authorization operations.
package usecase

import (
	"context"

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

	GetByTokenHash(ctx context.Context, tokenHash string) (*authDomain.Token, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, auditLog *authDomain.AuditLog) error
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

	// Delete performs a soft delete by setting IsActive to false, preventing authentication
	// while preserving the client record for audit purposes. The client's data remains in
	// the database but the client cannot authenticate until reactivated via Update.
	//
	// Returns ErrClientNotFound if the specified client doesn't exist.
	Delete(ctx context.Context, clientID uuid.UUID) error
}

type TokenUseCase interface {
	Issue(
		ctx context.Context,
		issueTokenInput *authDomain.IssueTokenInput,
	) (*authDomain.IssueTokenOutput, error)

	Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error)
}

type AuditLogUseCase interface {
	Create(
		ctx context.Context,
		requestID uuid.UUID,
		clientID uuid.UUID,
		capability authDomain.Capability,
		path string,
		metadata map[string]any,
	) error
}
