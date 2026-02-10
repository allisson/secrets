package domain

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID        uuid.UUID
	TokenHash string
	ClientID  uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
