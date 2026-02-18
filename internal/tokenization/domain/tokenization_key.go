package domain

import (
	"time"

	"github.com/google/uuid"
)

// TokenizationKey represents a versioned tokenization key configuration.
// Each key defines the token format and deterministic behavior for tokenization operations.
type TokenizationKey struct {
	ID              uuid.UUID
	Name            string
	Version         uint
	FormatType      FormatType
	IsDeterministic bool
	DekID           uuid.UUID
	CreatedAt       time.Time
	DeletedAt       *time.Time
}
