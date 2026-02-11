package usecase

import (
	"context"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

type ClientRepository interface {
	Create(ctx context.Context, client *authDomain.Client) error
	Update(ctx context.Context, client *authDomain.Client) error
	Get(ctx context.Context, clientID uuid.UUID) (*authDomain.Client, error)
}

type TokenRepository interface {
	Create(ctx context.Context, token *authDomain.Token) error
	Update(ctx context.Context, token *authDomain.Token) error
	Get(ctx context.Context, tokenID uuid.UUID) (*authDomain.Token, error)
}
