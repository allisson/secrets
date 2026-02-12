package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID
	RequestID  uuid.UUID
	ClientID   uuid.UUID
	Capability Capability
	Path       string
	Metadata   map[string]any
	CreatedAt  time.Time
}
