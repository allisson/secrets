package domain

import (
	"time"

	"github.com/google/uuid"
)

// Token represents a tokenization mapping between a token and its encrypted original value.
// Supports optional expiration, revocation, and metadata for display purposes.
type Token struct {
	ID                uuid.UUID
	TokenizationKeyID uuid.UUID
	Token             string
	ValueHash         *string
	Ciphertext        []byte
	Nonce             []byte
	Metadata          map[string]any
	CreatedAt         time.Time
	ExpiresAt         *time.Time
	RevokedAt         *time.Time
}

// IsExpired checks if the token has expired. All time comparisons use UTC.
func (t *Token) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(t.ExpiresAt.UTC())
}

// IsRevoked checks if the token has been revoked.
func (t *Token) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsValid checks if the token is valid (not expired and not revoked).
func (t *Token) IsValid() bool {
	return !t.IsExpired() && !t.IsRevoked()
}
