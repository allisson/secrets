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

// IssueTokenInput contains client credentials for token issuance requests.
// Used during authentication to verify client identity before generating tokens.
type IssueTokenInput struct {
	ClientID     uuid.UUID
	ClientSecret string
}

// IssueTokenOutput contains the newly issued authentication token and expiration.
// The PlainToken is only returned once and must be transmitted securely to the client.
type IssueTokenOutput struct {
	PlainToken string
	ExpiresAt  time.Time
}
