package app

import (
	"context"
	"fmt"

	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	secretsRepository "github.com/allisson/secrets/internal/secrets/repository"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
)

// SecretUseCase returns the secret use case.
func (c *Container) SecretUseCase(ctx context.Context) (secretsUseCase.SecretUseCase, error) {
	return c.secretUseCase.get(func() (secretsUseCase.SecretUseCase, error) {
		return c.initSecretUseCase(ctx)
	})
}

func (c *Container) initSecretUseCase(ctx context.Context) (secretsUseCase.SecretUseCase, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for secret use case: %w", err)
	}

	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for secret use case: %w", err)
	}

	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring for secret use case: %w", err)
	}

	return secretsUseCase.NewSecretUseCase(
		txManager,
		kr,
		secretsRepository.NewSecretRepository(db),
		c.config.SecretValueSizeLimitBytes,
	), nil
}

// buildSecretsModule assembles the secrets Route Module: use case → handler →
// module, with the shared authorizer and business metrics bound.
func (c *Container) buildSecretsModule(ctx context.Context) (*secretsHTTP.Module, error) {
	secretUseCase, err := c.SecretUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret use case for secrets module: %w", err)
	}

	authz, err := c.Authorizer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorizer for secrets module: %w", err)
	}

	bm, err := c.BusinessMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get business metrics for secrets module: %w", err)
	}

	handler := secretsHTTP.NewSecretHandler(secretUseCase, c.Logger())

	return secretsHTTP.NewModule(handler, authz, bm), nil
}
