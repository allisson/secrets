package domain

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID        uuid.UUID
	Secret    string
	Name      string
	IsActive  bool
	CreatedAt time.Time
}
