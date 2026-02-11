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
}
