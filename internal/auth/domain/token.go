package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// HashTokenPlain computes the canonical hex-encoded SHA-256 digest of a plaintext
// token. This is the single source of truth for how tokens are hashed both at
// issuance (storing the hash) and at authentication (looking up by hash).
// Drifting from this function on either side will silently invalidate every
// issued token.
func HashTokenPlain(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// mintRandomBase64 reads 32 cryptographically secure random bytes and returns
// the base64 URL encoding. Shared by MintToken and MintClientSecret.
func mintRandomBase64() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", apperrors.Wrap(err, "failed to read random bytes")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// MintToken creates a fresh authentication token and returns the plaintext value
// alongside the canonical hash that should be persisted. Tokens are 32 random
// bytes, base64-URL encoded; the hash is HashTokenPlain(plain).
func MintToken() (plain, hash string, err error) {
	plain, err = mintRandomBase64()
	if err != nil {
		return "", "", err
	}
	return plain, HashTokenPlain(plain), nil
}

// MintClientSecret creates a fresh client secret and returns the plaintext value.
// The caller is responsible for hashing it under the chosen password-hash policy
// before persistence.
func MintClientSecret() (plain string, err error) {
	return mintRandomBase64()
}

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
	ClientSecret string //nolint:gosec // authentication credential field
}

// IssueTokenOutput contains the newly issued authentication token and expiration.
// The PlainToken is only returned once and must be transmitted securely to the client.
type IssueTokenOutput struct {
	PlainToken string
	ExpiresAt  time.Time
}
