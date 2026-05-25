package app

import (
	"context"
	"fmt"

	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitRepository "github.com/allisson/secrets/internal/transit/repository"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
)

func (c *Container) TransitKeyRepository(ctx context.Context) (transitUseCase.TransitKeyRepository, error) {
	return c.transitKeyRepository.get(func() (transitUseCase.TransitKeyRepository, error) {
		return c.initTransitKeyRepository(ctx)
	})
}

func (c *Container) TransitKeyUseCase(ctx context.Context) (transitUseCase.TransitKeyUseCase, error) {
	return c.transitKeyUseCase.get(func() (transitUseCase.TransitKeyUseCase, error) {
		return c.initTransitKeyUseCase(ctx)
	})
}

func (c *Container) TransitKeyHandler(ctx context.Context) (*transitHTTP.TransitKeyHandler, error) {
	return c.transitKeyHandler.get(func() (*transitHTTP.TransitKeyHandler, error) {
		return c.initTransitKeyHandler(ctx)
	})
}

func (c *Container) CryptoHandler(ctx context.Context) (*transitHTTP.CryptoHandler, error) {
	return c.cryptoHandler.get(func() (*transitHTTP.CryptoHandler, error) {
		return c.initCryptoHandler(ctx)
	})
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

	return transitUseCase.NewTransitKeyUseCase(txManager, transitKeyRepository, kr), nil
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
