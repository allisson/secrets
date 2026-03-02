package app

import (
	"context"
	"fmt"

	cryptoMySQL "github.com/allisson/secrets/internal/crypto/repository/mysql"
	cryptoPostgreSQL "github.com/allisson/secrets/internal/crypto/repository/postgresql"
	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitMySQL "github.com/allisson/secrets/internal/transit/repository/mysql"
	transitPostgreSQL "github.com/allisson/secrets/internal/transit/repository/postgresql"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
)

// TransitKeyRepository returns the transit key repository instance.
func (c *Container) TransitKeyRepository(ctx context.Context) (transitUseCase.TransitKeyRepository, error) {
	var err error
	c.transitKeyRepositoryInit.Do(func() {
		c.transitKeyRepository, err = c.initTransitKeyRepository(ctx)
		if err != nil {
			c.initErrors.Store("transitKeyRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("transitKeyRepository"); ok {
		return nil, val.(error)
	}
	return c.transitKeyRepository, nil
}

// TransitDekRepository returns the DEK repository for transit use case.
func (c *Container) TransitDekRepository(ctx context.Context) (transitUseCase.DekRepository, error) {
	var err error
	c.transitDekRepositoryInit.Do(func() {
		c.transitDekRepository, err = c.initTransitDekRepository(ctx)
		if err != nil {
			c.initErrors.Store("transitDekRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("transitDekRepository"); ok {
		return nil, val.(error)
	}
	return c.transitDekRepository, nil
}

// TransitKeyUseCase returns the transit key use case instance.
func (c *Container) TransitKeyUseCase(ctx context.Context) (transitUseCase.TransitKeyUseCase, error) {
	var err error
	c.transitKeyUseCaseInit.Do(func() {
		c.transitKeyUseCase, err = c.initTransitKeyUseCase(ctx)
		if err != nil {
			c.initErrors.Store("transitKeyUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("transitKeyUseCase"); ok {
		return nil, val.(error)
	}
	return c.transitKeyUseCase, nil
}

// TransitKeyHandler returns the transit key HTTP handler instance.
func (c *Container) TransitKeyHandler(ctx context.Context) (*transitHTTP.TransitKeyHandler, error) {
	var err error
	c.transitKeyHandlerInit.Do(func() {
		c.transitKeyHandler, err = c.initTransitKeyHandler(ctx)
		if err != nil {
			c.initErrors.Store("transitKeyHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("transitKeyHandler"); ok {
		return nil, val.(error)
	}
	return c.transitKeyHandler, nil
}

// CryptoHandler returns the crypto HTTP handler instance.
func (c *Container) CryptoHandler(ctx context.Context) (*transitHTTP.CryptoHandler, error) {
	var err error
	c.cryptoHandlerInit.Do(func() {
		c.cryptoHandler, err = c.initCryptoHandler(ctx)
		if err != nil {
			c.initErrors.Store("cryptoHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("cryptoHandler"); ok {
		return nil, val.(error)
	}
	return c.cryptoHandler, nil
}

// initTransitKeyRepository creates the transit key repository based on the database driver.
func (c *Container) initTransitKeyRepository(
	ctx context.Context,
) (transitUseCase.TransitKeyRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for transit key repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return transitPostgreSQL.NewPostgreSQLTransitKeyRepository(db), nil
	case "mysql":
		return transitMySQL.NewMySQLTransitKeyRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initTransitDekRepository creates the DEK repository for transit use case.
func (c *Container) initTransitDekRepository(ctx context.Context) (transitUseCase.DekRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for transit dek repository: %w", err)
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

// initTransitKeyUseCase creates the transit key use case with all its dependencies.
func (c *Container) initTransitKeyUseCase(ctx context.Context) (transitUseCase.TransitKeyUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for transit key use case: %w", err)
	}

	transitKeyRepository, err := c.TransitKeyRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key repository for transit key use case: %w", err)
	}

	dekRepository, err := c.TransitDekRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dek repository for transit key use case: %w", err)
	}

	kekChain, err := c.loadKekChain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for transit key use case: %w", err)
	}

	keyManager := c.KeyManager()
	aeadManager := c.AEADManager()

	baseUseCase := transitUseCase.NewTransitKeyUseCase(
		txManager,
		transitKeyRepository,
		dekRepository,
		keyManager,
		aeadManager,
		kekChain,
	)

	// Wrap with metrics if enabled
	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for transit key use case: %w", err)
		}
		return transitUseCase.NewTransitKeyUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

// initTransitKeyHandler creates the transit key HTTP handler with all its dependencies.
func (c *Container) initTransitKeyHandler(ctx context.Context) (*transitHTTP.TransitKeyHandler, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for transit key handler: %w", err)
	}

	logger := c.Logger()

	return transitHTTP.NewTransitKeyHandler(transitKeyUseCase, logger), nil
}

// initCryptoHandler creates the crypto HTTP handler with all its dependencies.
func (c *Container) initCryptoHandler(ctx context.Context) (*transitHTTP.CryptoHandler, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for crypto handler: %w", err)
	}

	logger := c.Logger()

	return transitHTTP.NewCryptoHandler(transitKeyUseCase, logger), nil
}
