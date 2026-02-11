package domain

import (
	"time"

	"github.com/google/uuid"
)

type PolicyDocument struct {
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
}

type Client struct {
	ID        uuid.UUID
	Secret    string
	Name      string
	IsActive  bool
	Policies  []PolicyDocument
	CreatedAt time.Time
}
