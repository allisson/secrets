package app

import (
	"context"
	"fmt"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	tokenizationRepository "github.com/allisson/secrets/internal/tokenization/repository"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

func (c *Container) TokenizationKeyRepository(
	ctx context.Context,
) (tokenizationDomain.TokenizationKeyRepository, error) {
	return c.tokenizationKeyRepository.get(func() (tokenizationDomain.TokenizationKeyRepository, error) {
		return c.initTokenizationKeyRepository(ctx)
	})
}

func (c *Container) TokenizationTokenRepository(
	ctx context.Context,
) (tokenizationDomain.TokenRepository, error) {
	return c.tokenizationTokenRepository.get(func() (tokenizationDomain.TokenRepository, error) {
		return c.initTokenizationTokenRepository(ctx)
	})
}

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

func (c *Container) TokenizationKeyHandler(
	ctx context.Context,
) (*tokenizationHTTP.TokenizationKeyHandler, error) {
	return c.tokenizationKeyHandler.get(func() (*tokenizationHTTP.TokenizationKeyHandler, error) {
		return c.initTokenizationKeyHandler(ctx)
	})
}

func (c *Container) TokenizationHandler(ctx context.Context) (*tokenizationHTTP.TokenizationHandler, error) {
	return c.tokenizationHandler.get(func() (*tokenizationHTTP.TokenizationHandler, error) {
		return c.initTokenizationHandler(ctx)
	})
}

func (c *Container) initTokenizationKeyRepository(
	ctx context.Context,
) (tokenizationDomain.TokenizationKeyRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for tokenization key repository: %w", err)
	}

	return tokenizationRepository.NewTokenizationKeyRepository(db), nil
}

func (c *Container) initTokenizationTokenRepository(
	ctx context.Context,
) (tokenizationDomain.TokenRepository, error) {
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

	return tokenizationUseCase.NewTokenizationKeyUseCase(txManager, tokenizationKeyRepository, kr), nil
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

	return tokenizationUseCase.NewTokenizationUseCase(
		txManager,
		tokenizationKeyRepository,
		tokenRepository,
		kr,
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
