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
	txManager     database.TxManager
	clientRepo    ClientRepository
	secretService authService.SecretService
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

// List retrieves clients ordered by ID descending with pagination support.
// Returns empty slice if no clients found.
func (c *clientUseCase) List(ctx context.Context, offset, limit int) ([]*authDomain.Client, error) {
	return c.clientRepo.List(ctx, offset, limit)
}

// Unlock clears the lockout state for a client, resetting failed_attempts and locked_until.
// Returns ErrClientNotFound if the client doesn't exist.
func (c *clientUseCase) Unlock(ctx context.Context, clientID uuid.UUID) error {
	if _, err := c.clientRepo.Get(ctx, clientID); err != nil {
		return err
	}
	return c.clientRepo.UpdateLockState(ctx, clientID, 0, nil)
}

// NewClientUseCase creates a new ClientUseCase with the provided dependencies.
func NewClientUseCase(
	txManager database.TxManager,
	clientRepo ClientRepository,
	secretService authService.SecretService,
) ClientUseCase {
	return &clientUseCase{
		txManager:     txManager,
		clientRepo:    clientRepo,
		secretService: secretService,
	}
}
