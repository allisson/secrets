// Package usecase implements business logic orchestration for authentication operations.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authService "github.com/allisson/secrets/internal/auth/service"
	"github.com/allisson/secrets/internal/config"
)

// tokenUseCase implements TokenUseCase interface for managing authentication tokens.
type tokenUseCase struct {
	config        *config.Config
	clientRepo    ClientRepository
	tokenRepo     TokenRepository
	secretService authService.SecretService
	tokenService  authService.TokenService
}

// Issue authenticates a client and generates a new authentication token.
// Validates client exists and is active, verifies the client secret, generates a new token
// with expiration from config, stores the token hash, and returns the plain token (only shown once).
//
// Security: Returns ErrInvalidCredentials for non-existent clients or wrong secrets to prevent
// user enumeration attacks. Returns ErrClientInactive if the client exists but is not active.
// Returns ErrClientLocked if the client is locked due to too many failed authentication attempts.
func (t *tokenUseCase) Issue(
	ctx context.Context,
	issueTokenInput *authDomain.IssueTokenInput,
) (*authDomain.IssueTokenOutput, error) {
	// Get the client by ID
	client, err := t.clientRepo.Get(ctx, issueTokenInput.ClientID)
	if err != nil {
		// If client not found, return generic error to prevent enumeration
		if errors.Is(err, authDomain.ErrClientNotFound) {
			return nil, authDomain.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check hard lock (active lock window)
	if client.LockedUntil != nil && time.Now().UTC().Before(*client.LockedUntil) {
		return nil, authDomain.ErrClientLocked
	}

	// Check if client is active
	if !client.IsActive {
		return nil, authDomain.ErrClientInactive
	}

	// Verify the client secret
	if !t.secretService.CompareSecret(issueTokenInput.ClientSecret, client.Secret) {
		newAttempts := client.FailedAttempts + 1
		var lockedUntil *time.Time
		if t.config.LockoutMaxAttempts > 0 && newAttempts >= t.config.LockoutMaxAttempts {
			lockExpiry := time.Now().UTC().Add(t.config.LockoutDuration)
			lockedUntil = &lockExpiry
		}
		// Best-effort: don't block on lock-state errors
		_ = t.clientRepo.UpdateLockState(ctx, client.ID, newAttempts, lockedUntil)
		return nil, authDomain.ErrInvalidCredentials
	}

	// Reset on success
	if client.FailedAttempts > 0 || client.LockedUntil != nil {
		_ = t.clientRepo.UpdateLockState(ctx, client.ID, 0, nil)
	}

	// Generate a new token
	plainToken, tokenHash, err := t.tokenService.GenerateToken()
	if err != nil {
		return nil, err
	}

	// Create the token entity with expiration from config
	expiresAt := time.Now().UTC().Add(t.config.AuthTokenExpiration)
	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: tokenHash,
		ClientID:  client.ID,
		ExpiresAt: expiresAt,
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Persist the token
	if err := t.tokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	// Return the plain token with expiration time
	return &authDomain.IssueTokenOutput{
		PlainToken: plainToken,
		ExpiresAt:  expiresAt,
	}, nil
}

// Authenticate validates a token hash and returns the associated client. Validates token
// is not expired/revoked and client is active. Returns ErrInvalidCredentials for
// invalid/expired/revoked tokens or missing clients to prevent enumeration attacks.
// Returns ErrClientInactive if the client is not active. All time comparisons use UTC.
func (t *tokenUseCase) Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error) {
	// Get the token by hash
	token, err := t.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		// If token not found, return generic error to prevent enumeration
		if errors.Is(err, authDomain.ErrTokenNotFound) {
			return nil, authDomain.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if token is expired
	if token.ExpiresAt.Before(time.Now().UTC()) {
		return nil, authDomain.ErrInvalidCredentials
	}

	// Check if token is revoked
	if token.RevokedAt != nil {
		return nil, authDomain.ErrInvalidCredentials
	}

	// Get the associated client
	client, err := t.clientRepo.Get(ctx, token.ClientID)
	if err != nil {
		// If client not found, return generic error (shouldn't happen due to FK, but handle gracefully)
		if errors.Is(err, authDomain.ErrClientNotFound) {
			return nil, authDomain.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if client is active
	if !client.IsActive {
		return nil, authDomain.ErrClientInactive
	}

	// Return the authenticated client
	return client, nil
}

// NewTokenUseCase creates a new TokenUseCase with the provided dependencies.
func NewTokenUseCase(
	config *config.Config,
	clientRepo ClientRepository,
	tokenRepo TokenRepository,
	secretService authService.SecretService,
	tokenService authService.TokenService,
) TokenUseCase {
	return &tokenUseCase{
		config:        config,
		clientRepo:    clientRepo,
		tokenRepo:     tokenRepo,
		secretService: secretService,
		tokenService:  tokenService,
	}
}
