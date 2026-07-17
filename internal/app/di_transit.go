package app

import (
	"context"
	"fmt"

	transitHTTP "github.com/allisson/secrets/internal/transit/http"
	transitRepository "github.com/allisson/secrets/internal/transit/repository"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
)

func (c *Container) TransitKeyUseCase(ctx context.Context) (transitUseCase.TransitKeyUseCase, error) {
	return c.transitKeyUseCase.get(func() (transitUseCase.TransitKeyUseCase, error) {
		return c.initTransitKeyUseCase(ctx)
	})
}

func (c *Container) initTransitKeyUseCase(ctx context.Context) (transitUseCase.TransitKeyUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for transit key use case: %w", err)
	}

	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for transit key use case: %w", err)
	}

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for transit key use case: %w", err)
	}

	return transitUseCase.NewTransitKeyUseCase(
		txManager,
		transitRepository.NewTransitKeyRepository(db),
		kr,
	), nil
}

// buildTransitModule assembles the transit Route Module: use case → both
// handlers → module, with the shared authorizer and business metrics bound.
func (c *Container) buildTransitModule(ctx context.Context) (*transitHTTP.Module, error) {
	transitKeyUseCase, err := c.TransitKeyUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit key use case for transit module: %w", err)
	}

	authz, err := c.Authorizer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorizer for transit module: %w", err)
	}

	bm, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics for transit module: %w", err)
	}

	keyHandler := transitHTTP.NewTransitKeyHandler(transitKeyUseCase, c.Logger())
	cryptoHandler := transitHTTP.NewCryptoHandler(transitKeyUseCase, c.Logger())

	return transitHTTP.NewModule(keyHandler, cryptoHandler, authz, bm), nil
}
