package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

type DekRepository interface {
	Create(ctx context.Context, dek *cryptoDomain.Dek) error
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

type SecretRepository interface {
	Create(ctx context.Context, secret *secretsDomain.Secret) error
	Delete(ctx context.Context, secretID uuid.UUID) error
	GetByPath(ctx context.Context, path string) (*secretsDomain.Secret, error)
}

type SecretUseCase interface {
	CreateOrUpdate(ctx context.Context, path string, value []byte) (*secretsDomain.Secret, error)
	Get(ctx context.Context, path string) (*secretsDomain.Secret, error)
	Delete(ctx context.Context, path string) error
}
