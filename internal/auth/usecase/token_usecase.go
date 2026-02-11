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
//
// This method:
// 1. Validates the client exists and is active
// 2. Verifies the client secret matches
// 3. Generates a new token with expiration from config
// 4. Stores the token hash in the database
// 5. Returns the plain token to the caller (only shown once)
//
// Security Notes:
//   - Returns ErrInvalidCredentials for both non-existent clients and wrong secrets
//     to prevent user enumeration attacks
//   - Returns ErrClientInactive if the client exists but is not active
//   - The plain token is only returned once and should be transmitted securely
//   - Token expiration is set from Config.AuthTokenExpiration
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

	// Check if client is active
	if !client.IsActive {
		return nil, authDomain.ErrClientInactive
	}

	// Verify the client secret
	if !t.secretService.CompareSecret(issueTokenInput.ClientSecret, client.Secret) {
		return nil, authDomain.ErrInvalidCredentials
	}

	// Generate a new token
	plainToken, tokenHash, err := t.tokenService.GenerateToken()
	if err != nil {
		return nil, err
	}

	// Create the token entity with expiration from config
	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: tokenHash,
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(t.config.AuthTokenExpiration),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Persist the token
	if err := t.tokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	// Return the plain token
	return &authDomain.IssueTokenOutput{
		PlainToken: plainToken,
	}, nil
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
