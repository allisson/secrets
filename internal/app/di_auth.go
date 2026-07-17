package app

import (
	"context"
	"fmt"

	"github.com/allisson/go-pwdhash"
	"github.com/gin-gonic/gin"

	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authRepository "github.com/allisson/secrets/internal/auth/repository"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PasswordHasher returns a process-singleton pwdhash.PasswordHasher configured with
// the production policy. Used by HashSecret and CompareSecret closures below.
func (c *Container) PasswordHasher() *pwdhash.PasswordHasher {
	c.passwordHasherInit.Do(func() {
		h, err := pwdhash.New(pwdhash.WithPolicy(pwdhash.PolicyModerate))
		if err != nil {
			// PolicyModerate is a constant known to be valid; failure here is a programming bug.
			panic(err)
		}
		c.passwordHasher = h
	})
	return c.passwordHasher
}

// HashSecret returns the HashSecretFunc closure bound to the production hasher.
func (c *Container) HashSecret() authUseCase.HashSecretFunc {
	h := c.PasswordHasher()
	return func(plain string) (string, error) {
		hashed, err := h.Hash([]byte(plain))
		if err != nil {
			return "", apperrors.Wrap(err, "failed to hash secret")
		}
		return hashed, nil
	}
}

// CompareSecret returns the CompareSecretFunc closure bound to the production hasher.
func (c *Container) CompareSecret() authUseCase.CompareSecretFunc {
	h := c.PasswordHasher()
	return func(plain, hashed string) bool {
		ok, err := h.Verify([]byte(plain), hashed)
		return err == nil && ok
	}
}

// ClientUseCase returns the client use case.
func (c *Container) ClientUseCase(ctx context.Context) (authUseCase.ClientUseCase, error) {
	return c.clientUseCase.get(func() (authUseCase.ClientUseCase, error) {
		return c.initClientUseCase(ctx)
	})
}

// TokenUseCase returns the token use case.
func (c *Container) TokenUseCase(ctx context.Context) (authUseCase.TokenUseCase, error) {
	return c.tokenUseCase.get(func() (authUseCase.TokenUseCase, error) {
		return c.initTokenUseCase(ctx)
	})
}

// AuditLogUseCase returns the audit log use case.
func (c *Container) AuditLogUseCase(ctx context.Context) (authUseCase.AuditLogUseCase, error) {
	return c.auditLogUseCase.get(func() (authUseCase.AuditLogUseCase, error) {
		return c.initAuditLogUseCase(ctx)
	})
}

// Authorizer returns the per-route authorizer shared by every feature's Route
// Module. Memoized so repeated builder calls (e.g. in tests) reuse one instance.
func (c *Container) Authorizer(ctx context.Context) (*authHTTP.Authorizer, error) {
	return c.authorizer.get(func() (*authHTTP.Authorizer, error) {
		return c.initAuthorizer(ctx)
	})
}

// initClientUseCase creates the client use case with all its dependencies.
func (c *Container) initClientUseCase(ctx context.Context) (authUseCase.ClientUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for client use case: %w", err)
	}

	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for client use case: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for client use case: %w", err)
	}

	return authUseCase.NewClientUseCase(
		txManager,
		authRepository.NewClientRepository(db),
		authRepository.NewTokenRepository(db),
		auditLogUseCase,
		c.HashSecret(),
	), nil
}

// initTokenUseCase creates the token use case with all its dependencies.
func (c *Container) initTokenUseCase(ctx context.Context) (authUseCase.TokenUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for token use case: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for token use case: %w", err)
	}

	return authUseCase.NewTokenUseCase(
		c.config,
		authRepository.NewClientRepository(db),
		authRepository.NewTokenRepository(db),
		auditLogUseCase,
		c.CompareSecret(),
		c.Logger(),
	), nil
}

// initAuditLogUseCase creates the audit log use case with all its dependencies.
func (c *Container) initAuditLogUseCase(ctx context.Context) (authUseCase.AuditLogUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for audit log use case: %w", err)
	}

	keySigner, err := c.KeySigner(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get key signer for audit log use case: %w", err)
	}

	return authUseCase.NewAuditLogUseCase(authRepository.NewAuditLogRepository(db), keySigner), nil
}

// initAuthorizer builds the per-route authorizer from the shared audit log use case.
func (c *Container) initAuthorizer(ctx context.Context) (*authHTTP.Authorizer, error) {
	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for authorizer: %w", err)
	}

	return authHTTP.NewAuthorizer(auditLogUseCase, c.Logger()), nil
}

// buildAuthModule assembles the auth Route Module: use cases → handlers →
// module. The optional token rate limiter is built last so the error paths above
// never start its background goroutine.
func (c *Container) buildAuthModule(ctx context.Context) (*authHTTP.Module, error) {
	clientUseCase, err := c.ClientUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client use case for auth module: %w", err)
	}

	tokenUseCase, err := c.TokenUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case for auth module: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for auth module: %w", err)
	}

	authz, err := c.Authorizer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorizer for auth module: %w", err)
	}

	bm, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics for auth module: %w", err)
	}

	clientHandler := authHTTP.NewClientHandler(clientUseCase, c.Logger())
	tokenHandler := authHTTP.NewTokenHandler(tokenUseCase, c.Logger())
	auditLogHandler := authHTTP.NewAuditLogHandler(auditLogUseCase, c.Logger())

	// Built last: TokenRateLimitMiddleware starts a cleanup goroutine, so it must
	// not run on any of the error paths above.
	var tokenRateLimitMiddleware gin.HandlerFunc
	if c.config.RateLimitTokenEnabled {
		tokenRateLimitMiddleware = authHTTP.TokenRateLimitMiddleware(
			ctx,
			c.config.RateLimitTokenRequestsPerSec,
			c.config.RateLimitTokenBurst,
			c.Logger(),
		)
	}

	return authHTTP.NewModule(
		clientHandler,
		tokenHandler,
		auditLogHandler,
		authz,
		bm,
		tokenRateLimitMiddleware,
	), nil
}
