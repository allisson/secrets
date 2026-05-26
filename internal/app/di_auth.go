package app

import (
	"context"
	"fmt"

	"github.com/allisson/go-pwdhash"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
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

// ClientRepository returns the client repository.
func (c *Container) ClientRepository(ctx context.Context) (authDomain.ClientRepository, error) {
	return c.clientRepository.get(func() (authDomain.ClientRepository, error) {
		return c.initClientRepository(ctx)
	})
}

// ClientUseCase returns the client use case.
func (c *Container) ClientUseCase(ctx context.Context) (authUseCase.ClientUseCase, error) {
	return c.clientUseCase.get(func() (authUseCase.ClientUseCase, error) {
		return c.initClientUseCase(ctx)
	})
}

// TokenRepository returns the token repository.
func (c *Container) TokenRepository(ctx context.Context) (authDomain.TokenRepository, error) {
	return c.tokenRepository.get(func() (authDomain.TokenRepository, error) {
		return c.initTokenRepository(ctx)
	})
}

// AuditLogRepository returns the audit log repository.
func (c *Container) AuditLogRepository(ctx context.Context) (authDomain.AuditLogRepository, error) {
	return c.auditLogRepository.get(func() (authDomain.AuditLogRepository, error) {
		return c.initAuditLogRepository(ctx)
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

// ClientHandler returns the HTTP handler for client management operations.
func (c *Container) ClientHandler(ctx context.Context) (*authHTTP.ClientHandler, error) {
	return c.clientHandler.get(func() (*authHTTP.ClientHandler, error) {
		return c.initClientHandler(ctx)
	})
}

// TokenHandler returns the HTTP handler for token operations.
func (c *Container) TokenHandler(ctx context.Context) (*authHTTP.TokenHandler, error) {
	return c.tokenHandler.get(func() (*authHTTP.TokenHandler, error) {
		return c.initTokenHandler(ctx)
	})
}

// AuditLogHandler returns the HTTP handler for audit log operations.
func (c *Container) AuditLogHandler(ctx context.Context) (*authHTTP.AuditLogHandler, error) {
	return c.auditLogHandler.get(func() (*authHTTP.AuditLogHandler, error) {
		return c.initAuditLogHandler(ctx)
	})
}

// initClientRepository creates the client repository.
func (c *Container) initClientRepository(ctx context.Context) (authDomain.ClientRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for client repository: %w", err)
	}

	return authRepository.NewClientRepository(db), nil
}

// initClientUseCase creates the client use case with all its dependencies.
func (c *Container) initClientUseCase(ctx context.Context) (authUseCase.ClientUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for client use case: %w", err)
	}

	clientRepository, err := c.ClientRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client repository for client use case: %w", err)
	}

	tokenRepository, err := c.TokenRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for client use case: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for client use case: %w", err)
	}

	return authUseCase.NewClientUseCase(
		txManager,
		clientRepository,
		tokenRepository,
		auditLogUseCase,
		c.HashSecret(),
	), nil
}

// initTokenRepository creates the token repository.
func (c *Container) initTokenRepository(ctx context.Context) (authDomain.TokenRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for token repository: %w", err)
	}

	return authRepository.NewTokenRepository(db), nil
}

// initAuditLogRepository creates the audit log repository.
func (c *Container) initAuditLogRepository(ctx context.Context) (authDomain.AuditLogRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for audit log repository: %w", err)
	}

	return authRepository.NewAuditLogRepository(db), nil
}

// initTokenUseCase creates the token use case with all its dependencies.
func (c *Container) initTokenUseCase(ctx context.Context) (authUseCase.TokenUseCase, error) {
	clientRepository, err := c.ClientRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client repository for token use case: %w", err)
	}

	tokenRepository, err := c.TokenRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for token use case: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for token use case: %w", err)
	}

	return authUseCase.NewTokenUseCase(
		c.config,
		clientRepository,
		tokenRepository,
		auditLogUseCase,
		c.CompareSecret(),
		c.Logger(),
	), nil
}

// initAuditLogUseCase creates the audit log use case with all its dependencies.
func (c *Container) initAuditLogUseCase(ctx context.Context) (authUseCase.AuditLogUseCase, error) {
	auditLogRepository, err := c.AuditLogRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log repository for audit log use case: %w", err)
	}

	keySigner, err := c.KeySigner(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get key signer for audit log use case: %w", err)
	}

	return authUseCase.NewAuditLogUseCase(auditLogRepository, keySigner), nil
}

// initClientHandler creates the client HTTP handler with all its dependencies.
func (c *Container) initClientHandler(ctx context.Context) (*authHTTP.ClientHandler, error) {
	clientUseCase, err := c.ClientUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client use case for client handler: %w", err)
	}

	return authHTTP.NewClientHandler(clientUseCase, c.Logger()), nil
}

// initTokenHandler creates the token HTTP handler with all its dependencies.
func (c *Container) initTokenHandler(ctx context.Context) (*authHTTP.TokenHandler, error) {
	tokenUseCase, err := c.TokenUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case for token handler: %w", err)
	}

	return authHTTP.NewTokenHandler(tokenUseCase, c.Logger()), nil
}

// initAuditLogHandler creates the audit log HTTP handler with all its dependencies.
func (c *Container) initAuditLogHandler(ctx context.Context) (*authHTTP.AuditLogHandler, error) {
	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for audit log handler: %w", err)
	}

	return authHTTP.NewAuditLogHandler(auditLogUseCase, c.Logger()), nil
}
