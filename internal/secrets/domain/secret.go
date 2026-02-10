// Package domain defines the core domain models and types for secret management.
// Secrets use an immutable versioning system with envelope encryption where each
// update creates a new database row with an incremented version number.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Secret struct {
	ID         uuid.UUID
	Path       string
	Version    uint
	DekID      uuid.UUID
	Ciphertext []byte
	Plaintext  []byte // In memory only
	Nonce      []byte
	CreatedAt  time.Time
	DeletedAt  *time.Time
}
