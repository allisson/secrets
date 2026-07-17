package app

import (
	"context"
	"fmt"

	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationRepository "github.com/allisson/secrets/internal/tokenization/repository"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

func (c *Container) TokenizationKeyUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyUseCase, error) {
	return c.tokenizationKeyUseCase.get(func() (tokenizationUseCase.TokenizationKeyUseCase, error) {
		return c.initTokenizationKeyUseCase(ctx)
	})
}

func (c *Container) TokenizationUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationUseCase, error) {
	return c.tokenizationUseCase.get(func() (tokenizationUseCase.TokenizationUseCase, error) {
		return c.initTokenizationUseCase(ctx)
	})
}

func (c *Container) initTokenizationKeyUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization key use case: %w", err)
	}

	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization key use case: %w", err)
	}

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for tokenization key use case: %w", err)
	}

	return tokenizationUseCase.NewTokenizationKeyUseCase(
		txManager,
		tokenizationRepository.NewTokenizationKeyRepository(db),
		kr,
	), nil
}

func (c *Container) initTokenizationUseCase(
	ctx context.Context,
) (tokenizationUseCase.TokenizationUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization use case: %w", err)
	}

	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for tokenization use case: %w", err)
	}

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for tokenization use case: %w", err)
	}

	return tokenizationUseCase.NewTokenizationUseCase(
		txManager,
		tokenizationRepository.NewTokenizationKeyRepository(db),
		tokenizationRepository.NewTokenRepository(db),
		kr,
	), nil
}

// buildTokenizationModule assembles the tokenization Route Module: use cases →
// both handlers → module, with the shared authorizer and business metrics bound.
func (c *Container) buildTokenizationModule(ctx context.Context) (*tokenizationHTTP.Module, error) {
	tokenizationKeyUseCase, err := c.TokenizationKeyUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization key use case for tokenization module: %w", err)
	}

	tokenizationUC, err := c.TokenizationUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization use case for tokenization module: %w", err)
	}

	authz, err := c.Authorizer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorizer for tokenization module: %w", err)
	}

	bm, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics for tokenization module: %w", err)
	}

	keyHandler := tokenizationHTTP.NewTokenizationKeyHandler(tokenizationKeyUseCase, c.Logger())
	tokenizationHandler := tokenizationHTTP.NewTokenizationHandler(
		tokenizationUC,
		c.config.TokenizationBatchLimit,
		c.Logger(),
	)

	return tokenizationHTTP.NewModule(keyHandler, tokenizationHandler, authz, bm), nil
}
