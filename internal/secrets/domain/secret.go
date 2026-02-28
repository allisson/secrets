// Package domain defines the core domain models and types for secret management.
// Secrets use an immutable versioning system with envelope encryption where each
// update creates a new database row with an incremented version number.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Secret represents an encrypted secret with versioning and metadata.
type Secret struct {
	// ID is the unique identifier for this specific secret version.
	ID uuid.UUID
	// Path is the logical key used to access the secret (e.g., "/app/db-password").
	Path string
	// Version is the monotonically increasing version number for this path.
	Version uint
	// DekID references the Data Encryption Key used to encrypt this secret version.
	DekID uuid.UUID
	// Ciphertext contains the encrypted secret data.
	Ciphertext []byte
	// Plaintext holds the decrypted secret value in memory only; must be zeroed after use.
	Plaintext []byte `json:"-"`
	// Nonce is the random value used during AEAD encryption.
	Nonce []byte
	// CreatedAt is the UTC timestamp when this version was created.
	CreatedAt time.Time
	// DeletedAt marks when this secret was soft-deleted (nil if active).
	DeletedAt *time.Time
}
