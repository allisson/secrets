package app

import (
	"context"
	"fmt"

	authHTTP "github.com/allisson/secrets/internal/auth/http"
	authMySQL "github.com/allisson/secrets/internal/auth/repository/mysql"
	authPostgreSQL "github.com/allisson/secrets/internal/auth/repository/postgresql"
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

// ClientRepository returns the client repository based on database driver.
func (c *Container) ClientRepository(ctx context.Context) (authUseCase.ClientRepository, error) {
	var err error
	c.clientRepositoryInit.Do(func() {
		c.clientRepository, err = c.initClientRepository(ctx)
		if err != nil {
			c.initErrors.Store("clientRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("clientRepository"); ok {
		return nil, val.(error)
	}
	return c.clientRepository, nil
}

// ClientUseCase returns the client use case.
func (c *Container) ClientUseCase(ctx context.Context) (authUseCase.ClientUseCase, error) {
	var err error
	c.clientUseCaseInit.Do(func() {
		c.clientUseCase, err = c.initClientUseCase(ctx)
		if err != nil {
			c.initErrors.Store("clientUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("clientUseCase"); ok {
		return nil, val.(error)
	}
	return c.clientUseCase, nil
}

// TokenService returns the token service for authentication operations.
func (c *Container) TokenService() authService.TokenService {
	c.tokenServiceInit.Do(func() {
		c.tokenService = c.initTokenService()
	})
	return c.tokenService
}

// TokenRepository returns the token repository based on database driver.
func (c *Container) TokenRepository(ctx context.Context) (authUseCase.TokenRepository, error) {
	var err error
	c.tokenRepositoryInit.Do(func() {
		c.tokenRepository, err = c.initTokenRepository(ctx)
		if err != nil {
			c.initErrors.Store("tokenRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenRepository"); ok {
		return nil, val.(error)
	}
	return c.tokenRepository, nil
}

// AuditLogRepository returns the audit log repository based on database driver.
func (c *Container) AuditLogRepository(ctx context.Context) (authUseCase.AuditLogRepository, error) {
	var err error
	c.auditLogRepositoryInit.Do(func() {
		c.auditLogRepository, err = c.initAuditLogRepository(ctx)
		if err != nil {
			c.initErrors.Store("auditLogRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("auditLogRepository"); ok {
		return nil, val.(error)
	}
	return c.auditLogRepository, nil
}

// TokenUseCase returns the token use case.
func (c *Container) TokenUseCase(ctx context.Context) (authUseCase.TokenUseCase, error) {
	var err error
	c.tokenUseCaseInit.Do(func() {
		c.tokenUseCase, err = c.initTokenUseCase(ctx)
		if err != nil {
			c.initErrors.Store("tokenUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenUseCase"); ok {
		return nil, val.(error)
	}
	return c.tokenUseCase, nil
}

// AuditLogUseCase returns the audit log use case.
func (c *Container) AuditLogUseCase(ctx context.Context) (authUseCase.AuditLogUseCase, error) {
	var err error
	c.auditLogUseCaseInit.Do(func() {
		c.auditLogUseCase, err = c.initAuditLogUseCase(ctx)
		if err != nil {
			c.initErrors.Store("auditLogUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("auditLogUseCase"); ok {
		return nil, val.(error)
	}
	return c.auditLogUseCase, nil
}

// ClientHandler returns the HTTP handler for client management operations.
func (c *Container) ClientHandler(ctx context.Context) (*authHTTP.ClientHandler, error) {
	var err error
	c.clientHandlerInit.Do(func() {
		c.clientHandler, err = c.initClientHandler(ctx)
		if err != nil {
			c.initErrors.Store("clientHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("clientHandler"); ok {
		return nil, val.(error)
	}
	return c.clientHandler, nil
}

// TokenHandler returns the HTTP handler for token operations.
func (c *Container) TokenHandler(ctx context.Context) (*authHTTP.TokenHandler, error) {
	var err error
	c.tokenHandlerInit.Do(func() {
		c.tokenHandler, err = c.initTokenHandler(ctx)
		if err != nil {
			c.initErrors.Store("tokenHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenHandler"); ok {
		return nil, val.(error)
	}
	return c.tokenHandler, nil
}

// AuditLogHandler returns the HTTP handler for audit log operations.
func (c *Container) AuditLogHandler(ctx context.Context) (*authHTTP.AuditLogHandler, error) {
	var err error
	c.auditLogHandlerInit.Do(func() {
		c.auditLogHandler, err = c.initAuditLogHandler(ctx)
		if err != nil {
			c.initErrors.Store("auditLogHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("auditLogHandler"); ok {
		return nil, val.(error)
	}
	return c.auditLogHandler, nil
}

// initSecretService creates the secret service for authentication.
func (c *Container) initSecretService() authService.SecretService {
	return authService.NewSecretService()
}

// initClientRepository creates the client repository based on the database driver.
func (c *Container) initClientRepository(ctx context.Context) (authUseCase.ClientRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for client repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return authPostgreSQL.NewPostgreSQLClientRepository(db), nil
	case "mysql":
		return authMySQL.NewMySQLClientRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
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

	baseUseCase := authUseCase.NewClientUseCase(
		txManager,
		clientRepository,
		tokenRepository,
		auditLogUseCase,
		secretService,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for client use case: %w", err)
		}
		return authUseCase.NewClientUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTokenService creates the token service for authentication.
func (c *Container) initTokenService() authService.TokenService {
	return authService.NewTokenService()
}

// initTokenRepository creates the token repository based on the database driver.
func (c *Container) initTokenRepository(ctx context.Context) (authUseCase.TokenRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for token repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return authPostgreSQL.NewPostgreSQLTokenRepository(db), nil
	case "mysql":
		return authMySQL.NewMySQLTokenRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initAuditLogRepository creates the audit log repository based on the database driver.
func (c *Container) initAuditLogRepository(ctx context.Context) (authUseCase.AuditLogRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for audit log repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return authPostgreSQL.NewPostgreSQLAuditLogRepository(db), nil
	case "mysql":
		return authMySQL.NewMySQLAuditLogRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
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

	baseUseCase := authUseCase.NewTokenUseCase(
		c.config,
		clientRepository,
		tokenRepository,
		auditLogUseCase,
		secretService,
		tokenService,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for token use case: %w", err)
		}
		return authUseCase.NewTokenUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initAuditLogUseCase creates the audit log use case with all its dependencies.
func (c *Container) initAuditLogUseCase(ctx context.Context) (authUseCase.AuditLogUseCase, error) {
	auditLogRepository, err := c.AuditLogRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log repository for audit log use case: %w", err)
	}

	// Create audit signer service
	auditSigner := authService.NewAuditSigner()

	// Load KEK chain for signature verification
	kekChain, err := c.loadKekChain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for audit log use case: %w", err)
	}

	baseUseCase := authUseCase.NewAuditLogUseCase(auditLogRepository, auditSigner, kekChain)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for audit log use case: %w", err)
		}
		return authUseCase.NewAuditLogUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initClientHandler creates the client HTTP handler with all its dependencies.
func (c *Container) initClientHandler(ctx context.Context) (*authHTTP.ClientHandler, error) {
	clientUseCase, err := c.ClientUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client use case for client handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for client handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewClientHandler(clientUseCase, auditLogUseCase, logger), nil
}

// initTokenHandler creates the token HTTP handler with all its dependencies.
func (c *Container) initTokenHandler(ctx context.Context) (*authHTTP.TokenHandler, error) {
	tokenUseCase, err := c.TokenUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case for token handler: %w", err)
	}

	tokenService := c.TokenService()
	logger := c.Logger()

	return authHTTP.NewTokenHandler(tokenUseCase, tokenService, logger), nil
}

// initAuditLogHandler creates the audit log HTTP handler with all its dependencies.
func (c *Container) initAuditLogHandler(ctx context.Context) (*authHTTP.AuditLogHandler, error) {
	auditLogUseCase, err := c.AuditLogUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for audit log handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewAuditLogHandler(auditLogUseCase, logger), nil
}
