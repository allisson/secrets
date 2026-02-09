package domain

import (
	"time"

	"github.com/google/uuid"
)

type TransitKey struct {
	ID        uuid.UUID
	Name      string
	Version   uint
	DekID     uuid.UUID
	CreatedAt time.Time
	DeletedAt *time.Time
}
