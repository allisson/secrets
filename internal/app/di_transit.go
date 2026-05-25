package app

import (
	"context"
	"fmt"

	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitRepository "github.com/allisson/secrets/internal/transit/repository"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
)

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

func (c *Container) initTransitKeyRepository(
	ctx context.Context,
) (transitUseCase.TransitKeyRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for transit key repository: %w", err)
	}

	return transitRepository.NewTransitKeyRepository(db), nil
}

func (c *Container) initTransitKeyUseCase(ctx context.Context) (transitUseCase.TransitKeyUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for transit key use case: %w", err)
	}

	transitKeyRepository, err := c.TransitKeyRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key repository for transit key use case: %w", err)
	}

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for transit key use case: %w", err)
	}

	inner := transitUseCase.NewTransitKeyUseCase(txManager, transitKeyRepository, kr)
	if !c.config.MetricsEnabled {
		return inner, nil
	}
	bm, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics for transit key use case: %w", err)
	}
	return transitUseCase.NewMetricsTransitKeyUseCase(inner, bm, "transit"), nil
}

func (c *Container) initTransitKeyHandler(ctx context.Context) (*transitHTTP.TransitKeyHandler, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for transit key handler: %w", err)
	}

	return transitHTTP.NewTransitKeyHandler(transitKeyUseCase, c.Logger()), nil
}

func (c *Container) initCryptoHandler(ctx context.Context) (*transitHTTP.CryptoHandler, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for crypto handler: %w", err)
	}

	return transitHTTP.NewCryptoHandler(transitKeyUseCase, c.Logger()), nil
}
