package app

import (
	"fmt"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoMySQL "github.com/allisson/secrets/internal/crypto/repository/mysql"
	cryptoPostgreSQL "github.com/allisson/secrets/internal/crypto/repository/postgresql"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	secretsMySQL "github.com/allisson/secrets/internal/secrets/repository/mysql"
	secretsPostgreSQL "github.com/allisson/secrets/internal/secrets/repository/postgresql"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
)

// DekRepository returns the DEK repository based on database driver.
func (c *Container) DekRepository() (secretsUseCase.DekRepository, error) {
	var err error
	c.dekRepositoryInit.Do(func() {
		c.dekRepository, err = c.initDekRepository()
		if err != nil {
			c.initErrors["dekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["dekRepository"]; exists {
		return nil, storedErr
	}
	return c.dekRepository, nil
}

// SecretRepository returns the secret repository based on database driver.
func (c *Container) SecretRepository() (secretsUseCase.SecretRepository, error) {
	var err error
	c.secretRepositoryInit.Do(func() {
		c.secretRepository, err = c.initSecretRepository()
		if err != nil {
			c.initErrors["secretRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["secretRepository"]; exists {
		return nil, storedErr
	}
	return c.secretRepository, nil
}

// SecretUseCase returns the secret use case.
func (c *Container) SecretUseCase() (secretsUseCase.SecretUseCase, error) {
	var err error
	c.secretUseCaseInit.Do(func() {
		c.secretUseCase, err = c.initSecretUseCase()
		if err != nil {
			c.initErrors["secretUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["secretUseCase"]; exists {
		return nil, storedErr
	}
	return c.secretUseCase, nil
}

// SecretHandler returns the HTTP handler for secret management operations.
func (c *Container) SecretHandler() (*secretsHTTP.SecretHandler, error) {
	var err error
	c.secretHandlerInit.Do(func() {
		c.secretHandler, err = c.initSecretHandler()
		if err != nil {
			c.initErrors["secretHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["secretHandler"]; exists {
		return nil, storedErr
	}
	return c.secretHandler, nil
}

// initDekRepository creates the DEK repository based on the database driver.
func (c *Container) initDekRepository() (secretsUseCase.DekRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for dek repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoPostgreSQL.NewPostgreSQLDekRepository(db), nil
	case "mysql":
		return cryptoMySQL.NewMySQLDekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initSecretRepository creates the secret repository based on the database driver.
func (c *Container) initSecretRepository() (secretsUseCase.SecretRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for secret repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return secretsPostgreSQL.NewPostgreSQLSecretRepository(db), nil
	case "mysql":
		return secretsMySQL.NewMySQLSecretRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initSecretUseCase creates the secret use case with all its dependencies.
func (c *Container) initSecretUseCase() (secretsUseCase.SecretUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for secret use case: %w", err)
	}

	dekRepository, err := c.DekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for secret use case: %w", err)
	}

	secretRepository, err := c.SecretRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret repository for secret use case: %w", err)
	}

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for secret use case: %w", err)
	}

	aeadManager := c.AEADManager()
	keyManager := c.KeyManager()

	baseUseCase := secretsUseCase.NewSecretUseCase(
		txManager,
		dekRepository,
		secretRepository,
		kekChain,
		aeadManager,
		keyManager,
		cryptoDomain.AESGCM,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for secret use case: %w", err)
		}
		return secretsUseCase.NewSecretUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initSecretHandler creates the secret HTTP handler with all its dependencies.
func (c *Container) initSecretHandler() (*secretsHTTP.SecretHandler, error) {
	secretUseCase, err := c.SecretUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret use case for secret handler: %w", err)
	}

	auditLogUseCase, err := c.AuditLogUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log use case for secret handler: %w", err)
	}

	logger := c.Logger()

	return secretsHTTP.NewSecretHandler(secretUseCase, auditLogUseCase, logger), nil
}
