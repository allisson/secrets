package app

import (
	"context"
	"fmt"

	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	secretsRepository "github.com/allisson/secrets/internal/secrets/repository"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
)

// SecretRepository returns the secret repository.
func (c *Container) SecretRepository(ctx context.Context) (secretsUseCase.SecretRepository, error) {
	var err error
	c.secretRepositoryInit.Do(func() {
		c.secretRepository, err = c.initSecretRepository(ctx)
		if err != nil {
			c.initErrors.Store("secretRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("secretRepository"); ok {
		return nil, val.(error)
	}
	return c.secretRepository, nil
}

// SecretUseCase returns the secret use case.
func (c *Container) SecretUseCase(ctx context.Context) (secretsUseCase.SecretUseCase, error) {
	var err error
	c.secretUseCaseInit.Do(func() {
		c.secretUseCase, err = c.initSecretUseCase(ctx)
		if err != nil {
			c.initErrors.Store("secretUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("secretUseCase"); ok {
		return nil, val.(error)
	}
	return c.secretUseCase, nil
}

// SecretHandler returns the HTTP handler for secret management operations.
func (c *Container) SecretHandler(ctx context.Context) (*secretsHTTP.SecretHandler, error) {
	var err error
	c.secretHandlerInit.Do(func() {
		c.secretHandler, err = c.initSecretHandler(ctx)
		if err != nil {
			c.initErrors.Store("secretHandler", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("secretHandler"); ok {
		return nil, val.(error)
	}
	return c.secretHandler, nil
}

func (c *Container) initSecretRepository(ctx context.Context) (secretsUseCase.SecretRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for secret repository: %w", err)
	}

	return secretsRepository.NewSecretRepository(db), nil
}

func (c *Container) initSecretUseCase(ctx context.Context) (secretsUseCase.SecretUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for secret use case: %w", err)
	}

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for secret use case: %w", err)
	}

	secretRepository, err := c.SecretRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret repository for secret use case: %w", err)
	}

	baseUseCase := secretsUseCase.NewSecretUseCase(
		txManager,
		kr,
		secretRepository,
		c.config.SecretValueSizeLimitBytes,
	)

	if c.config.MetricsEnabled {
		businessMetrics, err := c.BusinessMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get business metrics for secret use case: %w", err)
		}
		return secretsUseCase.NewSecretUseCaseWithMetrics(baseUseCase, businessMetrics), nil
	}

	return baseUseCase, nil
}

func (c *Container) initSecretHandler(ctx context.Context) (*secretsHTTP.SecretHandler, error) {
	secretUseCase, err := c.SecretUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret use case for secret handler: %w", err)
	}

	return secretsHTTP.NewSecretHandler(secretUseCase, c.Logger()), nil
}
