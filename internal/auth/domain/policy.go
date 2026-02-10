package domain

import (
	"time"

	"github.com/google/uuid"
)

type PolicyDocument struct {
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
}

type Policy struct {
	ID        uuid.UUID
	Name      string
	Document  PolicyDocument
	CreatedAt time.Time
}
