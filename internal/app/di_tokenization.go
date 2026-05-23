package app

import (
	"context"
	"fmt"

	"github.com/allisson/secrets/internal/metrics"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationRepository "github.com/allisson/secrets/internal/tokenization/repository"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

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

func (c *Container) initTokenizationKeyRepository(
	ctx context.Context,
) (tokenizationUseCase.TokenizationKeyRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization key repository: %w", err)
	}

	return tokenizationRepository.NewTokenizationKeyRepository(db), nil
}

func (c *Container) initTokenizationTokenRepository(
	ctx context.Context,
) (tokenizationUseCase.TokenRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization token repository: %w", err)
	}

	return tokenizationRepository.NewTokenRepository(db), nil
}

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

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for tokenization key use case: %w", err)
	}

	var bm metrics.BusinessMetrics
	if c.config.MetricsEnabled {
		bm, err = c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for tokenization key use case: %w", err)
		}
	}

	return tokenizationUseCase.NewTokenizationKeyUseCase(
		txManager,
		tokenizationKeyRepository,
		kr,
		bm,
	), nil
}

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

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for tokenization use case: %w", err)
	}

	hashService := tokenizationUseCase.NewSHA256HashService()

	var bm metrics.BusinessMetrics
	if c.config.MetricsEnabled {
		bm, err = c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for tokenization use case: %w", err)
		}
	}

	return tokenizationUseCase.NewTokenizationUseCase(
		txManager,
		tokenizationKeyRepository,
		tokenRepository,
		hashService,
		kr,
		bm,
	), nil
}

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

	return tokenizationHTTP.NewTokenizationKeyHandler(tokenizationKeyUseCase, c.Logger()), nil
}

func (c *Container) initTokenizationHandler(
	ctx context.Context,
) (*tokenizationHTTP.TokenizationHandler, error) {
	tokenizationUC, err := c.TokenizationUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenization use case for tokenization handler: %w", err)
	}

	return tokenizationHTTP.NewTokenizationHandler(
		tokenizationUC,
		c.config.TokenizationBatchLimit,
		c.Logger(),
	), nil
}
