package app

import (
	"context"
	"fmt"

	cryptoMySQL "github.com/allisson/secrets/internal/crypto/repository/mysql"
	cryptoPostgreSQL "github.com/allisson/secrets/internal/crypto/repository/postgresql"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationMySQL "github.com/allisson/secrets/internal/tokenization/repository/mysql"
	tokenizationPostgreSQL "github.com/allisson/secrets/internal/tokenization/repository/postgresql"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// TokenizationKeyRepository returns the tokenization key repository.
func (c *Container) TokenizationKeyRepository(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyRepository, error) {
	var err error
	c.tokenizationKeyRepositoryInit.Do(func() {
		c.tokenizationKeyRepository, err = c.initTokenizationKeyRepository(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationKeyRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationKeyRepository"); ok {
		return nil, val.(error)
	}
	return c.tokenizationKeyRepository, nil
}

// TokenizationTokenRepository returns the tokenization token repository.
func (c *Container) TokenizationTokenRepository(
	ctx context.Context,
) (tokenizationUseCase.TokenRepository, error) {
	var err error
	c.tokenizationTokenRepositoryInit.Do(func() {
		c.tokenizationTokenRepository, err = c.initTokenizationTokenRepository(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationTokenRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationTokenRepository"); ok {
		return nil, val.(error)
	}
	return c.tokenizationTokenRepository, nil
}

// TokenizationDekRepository returns the DEK repository for tokenization use case.
func (c *Container) TokenizationDekRepository(
	ctx context.Context,
) (tokenizationUseCase.DekRepository, error) {
	var err error
	c.tokenizationDekRepositoryInit.Do(func() {
		c.tokenizationDekRepository, err = c.initTokenizationDekRepository(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationDekRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationDekRepository"); ok {
		return nil, val.(error)
	}
	return c.tokenizationDekRepository, nil
}

// TokenizationKeyUseCase returns the tokenization key use case.
func (c *Container) TokenizationKeyUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyUseCase, error) {
	var err error
	c.tokenizationKeyUseCaseInit.Do(func() {
		c.tokenizationKeyUseCase, err = c.initTokenizationKeyUseCase(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationKeyUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationKeyUseCase"); ok {
		return nil, val.(error)
	}
	return c.tokenizationKeyUseCase, nil
}

// TokenizationUseCase returns the tokenization use case.
func (c *Container) TokenizationUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationUseCase, error) {
	var err error
	c.tokenizationUseCaseInit.Do(func() {
		c.tokenizationUseCase, err = c.initTokenizationUseCase(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationUseCase"); ok {
		return nil, val.(error)
	}
	return c.tokenizationUseCase, nil
}

// TokenizationKeyHandler returns the tokenization key HTTP handler.
func (c *Container) TokenizationKeyHandler(
	ctx context.Context,
) (*tokenizationHTTP.TokenizationKeyHandler, error) {
	var err error
	c.tokenizationKeyHandlerInit.Do(func() {
		c.tokenizationKeyHandler, err = c.initTokenizationKeyHandler(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationKeyHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationKeyHandler"); ok {
		return nil, val.(error)
	}
	return c.tokenizationKeyHandler, nil
}

// TokenizationHandler returns the tokenization HTTP handler.
func (c *Container) TokenizationHandler(ctx context.Context) (*tokenizationHTTP.TokenizationHandler, error) {
	var err error
	c.tokenizationHandlerInit.Do(func() {
		c.tokenizationHandler, err = c.initTokenizationHandler(ctx)
		if err != nil {
			c.initErrors.Store("tokenizationHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("tokenizationHandler"); ok {
		return nil, val.(error)
	}
	return c.tokenizationHandler, nil
}

// initTokenizationKeyRepository creates the tokenization key repository.
func (c *Container) initTokenizationKeyRepository(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyRepository, error) {
	db, err := c.DB(ctx)
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
func (c *Container) initTokenizationTokenRepository(
	ctx context.Context,
) (tokenizationUseCase.TokenRepository, error) {
	db, err := c.DB(ctx)
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
func (c *Container) initTokenizationDekRepository(
	ctx context.Context,
) (tokenizationUseCase.DekRepository, error) {
	db, err := c.DB(ctx)
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
func (c *Container) initTokenizationKeyUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization key use case: %w", err)
	}

	tokenizationKeyRepository, err := c.TokenizationKeyRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get tokenization key repository for tokenization key use case: %w",
			err,
		)
	}

	dekRepository, err := c.TokenizationDekRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for tokenization key use case: %w", err)
	}

	kekChain, err := c.loadKekChain(ctx)
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
func (c *Container) initTokenizationUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization use case: %w", err)
	}

	tokenizationKeyRepository, err := c.TokenizationKeyRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get tokenization key repository for tokenization use case: %w",
			err,
		)
	}

	tokenRepository, err := c.TokenizationTokenRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token repository for tokenization use case: %w", err)
	}

	dekRepository, err := c.TokenizationDekRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for tokenization use case: %w", err)
	}

	aeadManager := c.AEADManager()

	keyManager := c.KeyManager()

	hashService := tokenizationUseCase.NewSHA256HashService()

	kekChain, err := c.loadKekChain(ctx)
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
		businessMetrics, err := c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for tokenization use case: %w", err)
		}
		return tokenizationUseCase.NewTokenizationUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTokenizationKeyHandler creates the tokenization key HTTP handler.
func (c *Container) initTokenizationKeyHandler(
	ctx context.Context,
) (*tokenizationHTTP.TokenizationKeyHandler, error) {
	tokenizationKeyUseCase, err := c.TokenizationKeyUseCase(ctx)
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
func (c *Container) initTokenizationHandler(
	ctx context.Context,
) (*tokenizationHTTP.TokenizationHandler, error) {
	tokenizationUseCase, err := c.TokenizationUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization use case for tokenization handler: %w", err)
	}

	logger := c.Logger()

	return tokenizationHTTP.NewTokenizationHandler(
		tokenizationUseCase,
		c.config.TokenizationBatchLimit,
		logger,
	), nil
}
