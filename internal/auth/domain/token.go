package domain

import (
	"time"

	"github.com/google/uuid"
)

// Token represents an authentication token with expiration and revocation support.
// Tokens are stored as hashes and associated with a client for authentication.
type Token struct {
	ID        uuid.UUID  // Unique identifier (UUIDv7)
	TokenHash string     // SHA-256 hash of the token string
	ClientID  uuid.UUID  // ID of the client that owns this token
	ExpiresAt time.Time  // Token expiration timestamp
	RevokedAt *time.Time // Token revocation timestamp (nil if active)
	CreatedAt time.Time
}

type IssueTokenInput struct {
	ClientID     uuid.UUID
	ClientSecret string
}

type IssueTokenOutput struct {
	PlainToken string
}
