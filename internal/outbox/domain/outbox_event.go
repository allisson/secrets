// Package domain defines the core outbox domain entities and types.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// OutboxEventStatus represents the status of an outbox event
type OutboxEventStatus string

const (
	OutboxEventStatusPending   OutboxEventStatus = "pending"
	OutboxEventStatusProcessed OutboxEventStatus = "processed"
	OutboxEventStatusFailed    OutboxEventStatus = "failed"
)

// OutboxEvent represents an event in the transactional outbox pattern
type OutboxEvent struct {
	ID          uuid.UUID
	EventType   string
	Payload     string
	Status      OutboxEventStatus
	Retries     int
	LastError   *string
	ProcessedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
