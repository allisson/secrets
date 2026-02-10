// Package usecase defines the interfaces and implementations for secret management use cases.
// Use cases orchestrate operations between repositories and services to implement business
// logic for managing encrypted secrets with automatic versioning.
package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// DekRepository defines the interface for Data Encryption Key persistence operations.
type DekRepository interface {
	Create(ctx context.Context, dek *cryptoDomain.Dek) error
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

// SecretRepository defines the interface for Secret persistence operations.
type SecretRepository interface {
	Create(ctx context.Context, secret *secretsDomain.Secret) error
	Delete(ctx context.Context, secretID uuid.UUID) error
	GetByPath(ctx context.Context, path string) (*secretsDomain.Secret, error)
	GetByPathAndVersion(ctx context.Context, path string, version uint) (*secretsDomain.Secret, error)
}

// SecretUseCase defines the interface for secret management business logic.
type SecretUseCase interface {
	CreateOrUpdate(ctx context.Context, path string, value []byte) (*secretsDomain.Secret, error)
	Get(ctx context.Context, path string) (*secretsDomain.Secret, error)
	Delete(ctx context.Context, path string) error
}
