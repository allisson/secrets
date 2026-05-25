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
	return c.secretRepository.get(func() (secretsUseCase.SecretRepository, error) {
		return c.initSecretRepository(ctx)
	})
}

// SecretUseCase returns the secret use case.
func (c *Container) SecretUseCase(ctx context.Context) (secretsUseCase.SecretUseCase, error) {
	return c.secretUseCase.get(func() (secretsUseCase.SecretUseCase, error) {
		return c.initSecretUseCase(ctx)
	})
}

// SecretHandler returns the HTTP handler for secret management operations.
func (c *Container) SecretHandler(ctx context.Context) (*secretsHTTP.SecretHandler, error) {
	return c.secretHandler.get(func() (*secretsHTTP.SecretHandler, error) {
		return c.initSecretHandler(ctx)
	})
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

	inner := secretsUseCase.NewSecretUseCase(
		txManager,
		kr,
		secretRepository,
		c.config.SecretValueSizeLimitBytes,
	)
	if !c.config.MetricsEnabled {
		return inner, nil
	}
	bm, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics for secret use case: %w", err)
	}
	return secretsUseCase.NewMetricsSecretUseCase(inner, bm, "secrets"), nil
}

func (c *Container) initSecretHandler(ctx context.Context) (*secretsHTTP.SecretHandler, error) {
	secretUseCase, err := c.SecretUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret use case for secret handler: %w", err)
	}

	return secretsHTTP.NewSecretHandler(secretUseCase, c.Logger()), nil
}
