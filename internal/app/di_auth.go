package app

import (
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
func (c *Container) ClientRepository() (authUseCase.ClientRepository, error) {
	var err error
	c.clientRepositoryInit.Do(func() {
		c.clientRepository, err = c.initClientRepository()
		if err != nil {
			c.initErrors["clientRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["clientRepository"]; exists {
		return nil, storedErr
	}
	return c.clientRepository, nil
}

// ClientUseCase returns the client use case.
func (c *Container) ClientUseCase() (authUseCase.ClientUseCase, error) {
	var err error
	c.clientUseCaseInit.Do(func() {
		c.clientUseCase, err = c.initClientUseCase()
		if err != nil {
			c.initErrors["clientUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["clientUseCase"]; exists {
		return nil, storedErr
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
func (c *Container) TokenRepository() (authUseCase.TokenRepository, error) {
	var err error
	c.tokenRepositoryInit.Do(func() {
		c.tokenRepository, err = c.initTokenRepository()
		if err != nil {
			c.initErrors["tokenRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenRepository"]; exists {
		return nil, storedErr
	}
	return c.tokenRepository, nil
}

// AuditLogRepository returns the audit log repository based on database driver.
func (c *Container) AuditLogRepository() (authUseCase.AuditLogRepository, error) {
	var err error
	c.auditLogRepositoryInit.Do(func() {
		c.auditLogRepository, err = c.initAuditLogRepository()
		if err != nil {
			c.initErrors["auditLogRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["auditLogRepository"]; exists {
		return nil, storedErr
	}
	return c.auditLogRepository, nil
}

// TokenUseCase returns the token use case.
func (c *Container) TokenUseCase() (authUseCase.TokenUseCase, error) {
	var err error
	c.tokenUseCaseInit.Do(func() {
		c.tokenUseCase, err = c.initTokenUseCase()
		if err != nil {
			c.initErrors["tokenUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenUseCase"]; exists {
		return nil, storedErr
	}
	return c.tokenUseCase, nil
}

// AuditLogUseCase returns the audit log use case.
func (c *Container) AuditLogUseCase() (authUseCase.AuditLogUseCase, error) {
	var err error
	c.auditLogUseCaseInit.Do(func() {
		c.auditLogUseCase, err = c.initAuditLogUseCase()
		if err != nil {
			c.initErrors["auditLogUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["auditLogUseCase"]; exists {
		return nil, storedErr
	}
	return c.auditLogUseCase, nil
}

// ClientHandler returns the HTTP handler for client management operations.
func (c *Container) ClientHandler() (*authHTTP.ClientHandler, error) {
	var err error
	c.clientHandlerInit.Do(func() {
		c.clientHandler, err = c.initClientHandler()
		if err != nil {
			c.initErrors["clientHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["clientHandler"]; exists {
		return nil, storedErr
	}
	return c.clientHandler, nil
}

// TokenHandler returns the HTTP handler for token operations.
func (c *Container) TokenHandler() (*authHTTP.TokenHandler, error) {
	var err error
	c.tokenHandlerInit.Do(func() {
		c.tokenHandler, err = c.initTokenHandler()
		if err != nil {
			c.initErrors["tokenHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenHandler"]; exists {
		return nil, storedErr
	}
	return c.tokenHandler, nil
}

// AuditLogHandler returns the HTTP handler for audit log operations.
func (c *Container) AuditLogHandler() (*authHTTP.AuditLogHandler, error) {
	var err error
	c.auditLogHandlerInit.Do(func() {
		c.auditLogHandler, err = c.initAuditLogHandler()
		if err != nil {
			c.initErrors["auditLogHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["auditLogHandler"]; exists {
		return nil, storedErr
	}
	return c.auditLogHandler, nil
}

// initSecretService creates the secret service for authentication.
func (c *Container) initSecretService() authService.SecretService {
	return authService.NewSecretService()
}

// initClientRepository creates the client repository based on the database driver.
func (c *Container) initClientRepository() (authUseCase.ClientRepository, error) {
	db, err := c.DB()
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
func (c *Container) initClientUseCase() (authUseCase.ClientUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for client use case: %w", err)
	}

	clientRepository, err := c.ClientRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get client repository for client use case: %w", err)
	}

	secretService := c.SecretService()

	baseUseCase := authUseCase.NewClientUseCase(txManager, clientRepository, secretService)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
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
func (c *Container) initTokenRepository() (authUseCase.TokenRepository, error) {
	db, err := c.DB()
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
func (c *Container) initAuditLogRepository() (authUseCase.AuditLogRepository, error) {
	db, err := c.DB()
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
func (c *Container) initTokenUseCase() (authUseCase.TokenUseCase, error) {
	clientRepository, err := c.ClientRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get client repository for token use case: %w", err)
	}

	tokenRepository, err := c.TokenRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for token use case: %w", err)
	}

	secretService := c.SecretService()
	tokenService := c.TokenService()

	baseUseCase := authUseCase.NewTokenUseCase(
		c.config,
		clientRepository,
		tokenRepository,
		secretService,
		tokenService,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for token use case: %w", err)
		}
		return authUseCase.NewTokenUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initAuditLogUseCase creates the audit log use case with all its dependencies.
func (c *Container) initAuditLogUseCase() (authUseCase.AuditLogUseCase, error) {
	auditLogRepository, err := c.AuditLogRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log repository for audit log use case: %w", err)
	}

	// Create audit signer service
	auditSigner := authService.NewAuditSigner()

	// Load KEK chain for signature verification
	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for audit log use case: %w", err)
	}

	baseUseCase := authUseCase.NewAuditLogUseCase(auditLogRepository, auditSigner, kekChain)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for audit log use case: %w", err)
		}
		return authUseCase.NewAuditLogUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initClientHandler creates the client HTTP handler with all its dependencies.
func (c *Container) initClientHandler() (*authHTTP.ClientHandler, error) {
	clientUseCase, err := c.ClientUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get client use case for client handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for client handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewClientHandler(clientUseCase, auditLogUseCase, logger), nil
}

// initTokenHandler creates the token HTTP handler with all its dependencies.
func (c *Container) initTokenHandler() (*authHTTP.TokenHandler, error) {
	tokenUseCase, err := c.TokenUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get token use case for token handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewTokenHandler(tokenUseCase, logger), nil
}

// initAuditLogHandler creates the audit log HTTP handler with all its dependencies.
func (c *Container) initAuditLogHandler() (*authHTTP.AuditLogHandler, error) {
	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for audit log handler: %w", err)
	}

	logger := c.Logger()

	return authHTTP.NewAuditLogHandler(auditLogUseCase, logger), nil
}
