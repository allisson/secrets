package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog records authorization decisions for compliance and security monitoring.
// Captures client identity, requested resource path, required capability, and metadata.
// Used to track access patterns and investigate security incidents.
type AuditLog struct {
	ID         uuid.UUID
	RequestID  uuid.UUID
	ClientID   uuid.UUID
	Capability Capability
	Path       string
	Metadata   map[string]any
	CreatedAt  time.Time
}
