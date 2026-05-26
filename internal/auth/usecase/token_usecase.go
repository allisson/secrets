// Package usecase implements business logic orchestration for authentication operations.
package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/config"
)

// CompareSecretFunc verifies a plaintext secret against its stored hash.
// Bound in DI from the production password-hash policy; tests inject trivial closures.
type CompareSecretFunc func(plain, hashed string) bool

// tokenUseCase implements TokenUseCase interface for managing authentication tokens.
type tokenUseCase struct {
	config          *config.Config
	clientRepo      authDomain.ClientRepository
	tokenRepo       authDomain.TokenRepository
	auditLogUseCase AuditLogUseCase
	compareSecret   CompareSecretFunc
	logger          *slog.Logger
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
) (result *authDomain.IssueTokenOutput, err error) {
	// Get the client by ID
	client, err := t.clientRepo.Get(ctx, issueTokenInput.ClientID)
	if err != nil {
		// If client not found, return generic error to prevent enumeration
		if errors.Is(err, authDomain.ErrClientNotFound) {
			return nil, authDomain.ErrInvalidCredentials
		}
		return nil, err
	}

	outcome := client.AttemptLogin(
		issueTokenInput.ClientSecret,
		t.compareSecret,
		authDomain.LockoutPolicy{
			MaxAttempts: t.config.LockoutMaxAttempts,
			Duration:    t.config.LockoutDuration,
		},
		time.Now().UTC(),
	)

	if needsLockStatePersist(client, outcome) {
		if persistErr := t.clientRepo.UpdateLockState(
			ctx, client.ID, outcome.FailedAttempts, outcome.LockedUntil,
		); persistErr != nil {
			if t.logger != nil {
				t.logger.Error(
					"failed to persist client lockout state",
					slog.String("client_id", client.ID.String()),
					slog.Any("error", persistErr),
				)
			}
			// On non-authenticated paths, preserve the existing security property:
			// don't leak DB failure modes to attackers. The lockout counter may be
			// briefly stale; the next attempt will reconcile.
			if outcome.Decision != authDomain.DecisionAuthenticated {
				return nil, authDomain.ErrInvalidCredentials
			}
			// On the authenticated path, fail loudly so no token is issued against
			// an unreconciled lock state.
			return nil, persistErr
		}
	}

	switch outcome.Decision {
	case authDomain.DecisionLocked:
		return nil, authDomain.ErrClientLocked
	case authDomain.DecisionInactive:
		return nil, authDomain.ErrClientInactive
	case authDomain.DecisionBadSecret:
		return nil, authDomain.ErrInvalidCredentials
	}

	// Generate a new token
	plainToken, tokenHash, err := authDomain.MintToken()
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
	if err = t.tokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	// Return the plain token with expiration time
	return &authDomain.IssueTokenOutput{
		PlainToken: plainToken,
		ExpiresAt:  expiresAt,
	}, nil
}

// Authenticate validates a raw bearer token and returns the associated client. Validates token
// is not expired/revoked and client is active. Returns ErrInvalidCredentials for
// invalid/expired/revoked tokens or missing clients to prevent enumeration attacks.
// Returns ErrClientInactive if the client is not active. All time comparisons use UTC.
func (t *tokenUseCase) Authenticate(
	ctx context.Context,
	rawToken string,
) (result *authDomain.Client, err error) {
	// Get the token by hash
	token, err := t.tokenRepo.GetByTokenHash(ctx, authDomain.HashTokenPlain(rawToken))
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

// Revoke marks a specific token as revoked given its raw bearer value.
func (t *tokenUseCase) Revoke(ctx context.Context, rawToken string) (err error) {
	// Get the token by hash
	token, err := t.tokenRepo.GetByTokenHash(ctx, authDomain.HashTokenPlain(rawToken))
	if err != nil {
		return err
	}

	// Revoke the token
	if err = t.tokenRepo.RevokeByTokenID(ctx, token.ID); err != nil {
		return err
	}

	// Record audit log
	// Note: RequestID is generated here as it's not currently available in the context
	// from the authentication middleware.
	_ = t.auditLogUseCase.Create(
		ctx,
		uuid.Must(uuid.NewV7()),
		token.ClientID,
		authDomain.DeleteCapability,
		"/v1/token",
		map[string]any{
			"token_id": token.ID.String(),
			"action":   "token_revoked",
		},
	)

	return nil
}

// PurgeExpiredAndRevoked permanently deletes tokens that are either expired or revoked
// and were created before the specified number of days ago.
func (t *tokenUseCase) PurgeExpiredAndRevoked(ctx context.Context, days int) (count int64, err error) {
	// Standard project validation for days/retention parameters
	// If days is 0, it means delete anything that is already expired or revoked.
	if days < 0 {
		return 0, errors.New("days must be greater than or equal to 0")
	}

	olderThan := time.Now().UTC().AddDate(0, 0, -days)
	count, err = t.tokenRepo.PurgeExpiredAndRevoked(ctx, olderThan)
	return
}

// needsLockStatePersist reports whether the outcome changes the persisted lockout state.
// Locked and Inactive decisions never change state. BadSecret always changes state
// (FailedAttempts is incremented). Authenticated changes state only when the previous
// FailedAttempts or LockedUntil was non-zero (a reset).
func needsLockStatePersist(client *authDomain.Client, outcome authDomain.LoginOutcome) bool {
	switch outcome.Decision {
	case authDomain.DecisionBadSecret:
		return true
	case authDomain.DecisionAuthenticated:
		return client.FailedAttempts > 0 || client.LockedUntil != nil
	default:
		return false
	}
}

// NewTokenUseCase creates a new TokenUseCase with the provided dependencies.
// A nil logger is permitted; lockout-state persistence failures will be swallowed silently.
func NewTokenUseCase(
	config *config.Config,
	clientRepo authDomain.ClientRepository,
	tokenRepo authDomain.TokenRepository,
	auditLogUseCase AuditLogUseCase,
	compareSecret CompareSecretFunc,
	logger *slog.Logger,
) TokenUseCase {
	return &tokenUseCase{
		config:          config,
		clientRepo:      clientRepo,
		tokenRepo:       tokenRepo,
		auditLogUseCase: auditLogUseCase,
		compareSecret:   compareSecret,
		logger:          logger,
	}
}
