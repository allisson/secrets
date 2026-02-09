package usecase

import (
	"context"

	"github.com/google/uuid"

	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

type TransitKeyRepository interface {
	Create(ctx context.Context, transitKey *transitDomain.TransitKey) error
	Delete(ctx context.Context, transitKeyID uuid.UUID) error
	GetByName(ctx context.Context, name string) (*transitDomain.TransitKey, error)
	GetByNameAndVersion(ctx context.Context, name string, version uint) (*transitDomain.TransitKey, error)
}
