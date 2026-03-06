// Package usecase implements business logic orchestration for authentication operations.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authService "github.com/allisson/secrets/internal/auth/service"
	"github.com/allisson/secrets/internal/database"
)

// clientUseCase implements ClientUseCase interface for managing client authentication.
type clientUseCase struct {
	txManager       database.TxManager
	clientRepo      ClientRepository
	tokenRepo       TokenRepository
	auditLogUseCase AuditLogUseCase
	secretService   authService.SecretService
}

// Create generates and persists a new Client with a random secret.
// Returns the client ID and plain text secret. The plain secret is only returned once
// and must be securely stored by the caller. The hashed version is stored in the database.
func (c *clientUseCase) Create(
	ctx context.Context,
	createClientInput *authDomain.CreateClientInput,
) (*authDomain.CreateClientOutput, error) {
	// Generate a secure random secret
	plainSecret, hashedSecret, err := c.secretService.GenerateSecret()
	if err != nil {
		return nil, err
	}

	// Create the client entity
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    hashedSecret,
		Name:      createClientInput.Name,
		IsActive:  createClientInput.IsActive,
		Policies:  createClientInput.Policies,
		CreatedAt: time.Now().UTC(),
	}

	// Persist the client
	if err := c.clientRepo.Create(ctx, client); err != nil {
		return nil, err
	}

	// Return the client ID and plain secret
	return &authDomain.CreateClientOutput{
		ID:          client.ID,
		PlainSecret: plainSecret,
		Name:        client.Name,
		IsActive:    client.IsActive,
		Policies:    client.Policies,
		CreatedAt:   client.CreatedAt,
	}, nil
}

// Update modifies an existing client's configuration.
// Only Name, IsActive, and Policies can be updated. The client secret and ID remain unchanged.
func (c *clientUseCase) Update(
	ctx context.Context,
	clientID uuid.UUID,
	updateClientInput *authDomain.UpdateClientInput,
) error {
	// Get the existing client
	client, err := c.clientRepo.Get(ctx, clientID)
	if err != nil {
		return err
	}

	// Update mutable fields
	client.Name = updateClientInput.Name
	client.IsActive = updateClientInput.IsActive
	client.Policies = updateClientInput.Policies

	// Persist the updated client
	return c.clientRepo.Update(ctx, client)
}

// Get retrieves a client by ID.
// Returns ErrClientNotFound if the client doesn't exist.
func (c *clientUseCase) Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error) {
	return c.clientRepo.Get(ctx, clientID)
}

// Delete performs a soft delete on a client by setting IsActive to false.
// This prevents the client from authenticating while preserving audit history.
func (c *clientUseCase) Delete(ctx context.Context, clientID uuid.UUID) error {
	// Get the existing client
	client, err := c.clientRepo.Get(ctx, clientID)
	if err != nil {
		return err
	}

	// Soft delete by deactivating
	client.IsActive = false

	// Persist the updated client
	return c.clientRepo.Update(ctx, client)
}

// ListCursor retrieves clients ordered by ID descending (newest first) with cursor-based pagination.
// If afterID is provided, returns clients with ID less than afterID (DESC order).
// Returns empty slice if no clients found. Limit is pre-validated (1-1000).
func (c *clientUseCase) ListCursor(
	ctx context.Context,
	afterID *uuid.UUID,
	limit int,
) ([]*authDomain.Client, error) {
	return c.clientRepo.ListCursor(ctx, afterID, limit)
}

// Unlock clears the lockout state for a client, resetting failed_attempts and locked_until.
// Returns ErrClientNotFound if the client doesn't exist.
func (c *clientUseCase) Unlock(ctx context.Context, clientID uuid.UUID) error {
	if _, err := c.clientRepo.Get(ctx, clientID); err != nil {
		return err
	}
	return c.clientRepo.UpdateLockState(ctx, clientID, 0, nil)
}

// RevokeTokens marks all active tokens for a specific client as revoked.
func (c *clientUseCase) RevokeTokens(ctx context.Context, clientID uuid.UUID) error {
	// Check if client exists
	if _, err := c.clientRepo.Get(ctx, clientID); err != nil {
		return err
	}

	// Revoke all tokens for the client
	if err := c.tokenRepo.RevokeByClientID(ctx, clientID); err != nil {
		return err
	}

	// Record audit log
	_ = c.auditLogUseCase.Create(
		ctx,
		uuid.Must(uuid.NewV7()),
		clientID,
		authDomain.DeleteCapability,
		"/v1/clients/"+clientID.String()+"/tokens",
		map[string]any{
			"action": "client_tokens_revoked",
		},
	)

	return nil
}

// RotateSecret generates a new secret for a client and revokes all active tokens.
func (c *clientUseCase) RotateSecret(
	ctx context.Context,
	clientID uuid.UUID,
) (*authDomain.CreateClientOutput, error) {
	var output *authDomain.CreateClientOutput

	err := c.txManager.WithTx(ctx, func(ctx context.Context) error {
		// Get the existing client
		client, err := c.clientRepo.Get(ctx, clientID)
		if err != nil {
			return err
		}

		// Generate a new secure random secret
		plainSecret, hashedSecret, err := c.secretService.GenerateSecret()
		if err != nil {
			return err
		}

		// Update the client entity
		client.Secret = hashedSecret
		if err := c.clientRepo.Update(ctx, client); err != nil {
			return err
		}

		// Revoke all tokens for the client
		if err := c.tokenRepo.RevokeByClientID(ctx, clientID); err != nil {
			return err
		}

		// Record audit log
		_ = c.auditLogUseCase.Create(
			ctx,
			uuid.Must(uuid.NewV7()),
			clientID,
			authDomain.RotateCapability,
			"/v1/clients/"+clientID.String()+"/rotate-secret",
			map[string]any{
				"action": "client_secret_rotated",
			},
		)

		output = &authDomain.CreateClientOutput{
			ID:          client.ID,
			PlainSecret: plainSecret,
			Name:        client.Name,
			IsActive:    client.IsActive,
			Policies:    client.Policies,
			CreatedAt:   client.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return output, nil
}

// NewClientUseCase creates a new ClientUseCase with the provided dependencies.
func NewClientUseCase(
	txManager database.TxManager,
	clientRepo ClientRepository,
	tokenRepo TokenRepository,
	auditLogUseCase AuditLogUseCase,
	secretService authService.SecretService,
) ClientUseCase {
	return &clientUseCase{
		txManager:       txManager,
		clientRepo:      clientRepo,
		tokenRepo:       tokenRepo,
		auditLogUseCase: auditLogUseCase,
		secretService:   secretService,
	}
}
