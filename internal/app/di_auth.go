package app

import (
	"context"
	"fmt"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authRepository "github.com/allisson/secrets/internal/auth/repository"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

// SecretService returns the secret service for authentication operations.
func (c *Container) SecretService() authService.SecretService {
	c.secretServiceInit.Do(func() {
		c.secretService = c.initSecretService()
	})
	return c.secretService
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

// TokenService returns the token service for authentication operations.
func (c *Container) TokenService() authService.TokenService {
	c.tokenServiceInit.Do(func() {
		c.tokenService = c.initTokenService()
	})
	return c.tokenService
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

// initSecretService creates the secret service for authentication.
func (c *Container) initSecretService() authService.SecretService {
	return authService.NewSecretService()
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

	secretService := c.SecretService()

	return authUseCase.NewClientUseCase(
		txManager,
		clientRepository,
		tokenRepository,
		auditLogUseCase,
		secretService,
	), nil
}

// initTokenService creates the token service for authentication.
func (c *Container) initTokenService() authService.TokenService {
	return authService.NewTokenService()
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

	secretService := c.SecretService()
	tokenService := c.TokenService()

	return authUseCase.NewTokenUseCase(
		c.config,
		clientRepository,
		tokenRepository,
		auditLogUseCase,
		secretService,
		tokenService,
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
