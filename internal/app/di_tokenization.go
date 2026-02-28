package app

import (
	"fmt"

	cryptoMySQL "github.com/allisson/secrets/internal/crypto/repository/mysql"
	cryptoPostgreSQL "github.com/allisson/secrets/internal/crypto/repository/postgresql"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationMySQL "github.com/allisson/secrets/internal/tokenization/repository/mysql"
	tokenizationPostgreSQL "github.com/allisson/secrets/internal/tokenization/repository/postgresql"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// TokenizationKeyRepository returns the tokenization key repository.
func (c *Container) TokenizationKeyRepository() (tokenizationUseCase.TokenizationKeyRepository, error) {
	var err error
	c.tokenizationKeyRepositoryInit.Do(func() {
		c.tokenizationKeyRepository, err = c.initTokenizationKeyRepository()
		if err != nil {
			c.initErrors["tokenizationKeyRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationKeyRepository"]; exists {
		return nil, storedErr
	}
	return c.tokenizationKeyRepository, nil
}

// TokenizationTokenRepository returns the tokenization token repository.
func (c *Container) TokenizationTokenRepository() (tokenizationUseCase.TokenRepository, error) {
	var err error
	c.tokenizationTokenRepositoryInit.Do(func() {
		c.tokenizationTokenRepository, err = c.initTokenizationTokenRepository()
		if err != nil {
			c.initErrors["tokenizationTokenRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationTokenRepository"]; exists {
		return nil, storedErr
	}
	return c.tokenizationTokenRepository, nil
}

// TokenizationDekRepository returns the DEK repository for tokenization use case.
func (c *Container) TokenizationDekRepository() (tokenizationUseCase.DekRepository, error) {
	var err error
	c.tokenizationDekRepositoryInit.Do(func() {
		c.tokenizationDekRepository, err = c.initTokenizationDekRepository()
		if err != nil {
			c.initErrors["tokenizationDekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationDekRepository"]; exists {
		return nil, storedErr
	}
	return c.tokenizationDekRepository, nil
}

// TokenizationKeyUseCase returns the tokenization key use case.
func (c *Container) TokenizationKeyUseCase() (tokenizationUseCase.TokenizationKeyUseCase, error) {
	var err error
	c.tokenizationKeyUseCaseInit.Do(func() {
		c.tokenizationKeyUseCase, err = c.initTokenizationKeyUseCase()
		if err != nil {
			c.initErrors["tokenizationKeyUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationKeyUseCase"]; exists {
		return nil, storedErr
	}
	return c.tokenizationKeyUseCase, nil
}

// TokenizationUseCase returns the tokenization use case.
func (c *Container) TokenizationUseCase() (tokenizationUseCase.TokenizationUseCase, error) {
	var err error
	c.tokenizationUseCaseInit.Do(func() {
		c.tokenizationUseCase, err = c.initTokenizationUseCase()
		if err != nil {
			c.initErrors["tokenizationUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationUseCase"]; exists {
		return nil, storedErr
	}
	return c.tokenizationUseCase, nil
}

// TokenizationKeyHandler returns the tokenization key HTTP handler.
func (c *Container) TokenizationKeyHandler() (*tokenizationHTTP.TokenizationKeyHandler, error) {
	var err error
	c.tokenizationKeyHandlerInit.Do(func() {
		c.tokenizationKeyHandler, err = c.initTokenizationKeyHandler()
		if err != nil {
			c.initErrors["tokenizationKeyHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationKeyHandler"]; exists {
		return nil, storedErr
	}
	return c.tokenizationKeyHandler, nil
}

// TokenizationHandler returns the tokenization HTTP handler.
func (c *Container) TokenizationHandler() (*tokenizationHTTP.TokenizationHandler, error) {
	var err error
	c.tokenizationHandlerInit.Do(func() {
		c.tokenizationHandler, err = c.initTokenizationHandler()
		if err != nil {
			c.initErrors["tokenizationHandler"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["tokenizationHandler"]; exists {
		return nil, storedErr
	}
	return c.tokenizationHandler, nil
}

// initTokenizationKeyRepository creates the tokenization key repository.
func (c *Container) initTokenizationKeyRepository() (tokenizationUseCase.TokenizationKeyRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization key repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return tokenizationPostgreSQL.NewPostgreSQLTokenizationKeyRepository(db), nil
	case "mysql":
		return tokenizationMySQL.NewMySQLTokenizationKeyRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTokenizationTokenRepository creates the tokenization token repository.
func (c *Container) initTokenizationTokenRepository() (tokenizationUseCase.TokenRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization token repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return tokenizationPostgreSQL.NewPostgreSQLTokenRepository(db), nil
	case "mysql":
		return tokenizationMySQL.NewMySQLTokenRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTokenizationDekRepository creates the DEK repository for tokenization use case.
func (c *Container) initTokenizationDekRepository() (tokenizationUseCase.DekRepository, error) {
	db, err := c.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization dek repository: %w", err)
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

// initTokenizationKeyUseCase creates the tokenization key use case.
func (c *Container) initTokenizationKeyUseCase() (tokenizationUseCase.TokenizationKeyUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization key use case: %w", err)
	}

	tokenizationKeyRepository, err := c.TokenizationKeyRepository()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get tokenization key repository for tokenization key use case: %w",
			err,
		)
	}

	dekRepository, err := c.TokenizationDekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for tokenization key use case: %w", err)
	}

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for tokenization key use case: %w", err)
	}

	keyManager := c.KeyManager()

	return tokenizationUseCase.NewTokenizationKeyUseCase(
		txManager,
		tokenizationKeyRepository,
		dekRepository,
		keyManager,
		kekChain,
	), nil
}

// initTokenizationUseCase creates the tokenization use case.
func (c *Container) initTokenizationUseCase() (tokenizationUseCase.TokenizationUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization use case: %w", err)
	}

	tokenizationKeyRepository, err := c.TokenizationKeyRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization key repository for tokenization use case: %w", err)
	}

	tokenRepository, err := c.TokenizationTokenRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for tokenization use case: %w", err)
	}

	dekRepository, err := c.TokenizationDekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for tokenization use case: %w", err)
	}

	aeadManager := c.AEADManager()

	keyManager := c.KeyManager()

	hashService := tokenizationUseCase.NewSHA256HashService()

	kekChain, err := c.loadKekChain()
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for tokenization use case: %w", err)
	}

	baseUseCase := tokenizationUseCase.NewTokenizationUseCase(
		txManager,
		tokenizationKeyRepository,
		tokenRepository,
		dekRepository,
		aeadManager,
		keyManager,
		hashService,
		kekChain,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for tokenization use case: %w", err)
		}
		return tokenizationUseCase.NewTokenizationUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTokenizationKeyHandler creates the tokenization key HTTP handler.
func (c *Container) initTokenizationKeyHandler() (*tokenizationHTTP.TokenizationKeyHandler, error) {
	tokenizationKeyUseCase, err := c.TokenizationKeyUseCase()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get tokenization key use case for tokenization key handler: %w",
			err,
		)
	}

	logger := c.Logger()

	return tokenizationHTTP.NewTokenizationKeyHandler(tokenizationKeyUseCase, logger), nil
}

// initTokenizationHandler creates the tokenization HTTP handler.
func (c *Container) initTokenizationHandler() (*tokenizationHTTP.TokenizationHandler, error) {
	tokenizationUseCase, err := c.TokenizationUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization use case for tokenization handler: %w", err)
	}

	logger := c.Logger()

	return tokenizationHTTP.NewTokenizationHandler(tokenizationUseCase, logger), nil
}
