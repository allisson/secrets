// Package domain defines the core cryptographic domain models and types.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Kek represents a Key Encryption Key used in envelope encryption.
//
// A KEK is a cryptographic key used to encrypt Data Encryption Keys (DEKs).
// The KEK itself is encrypted with a master key and stored securely in a database.
// This approach allows for key rotation without re-encrypting all data.
//
// In envelope encryption hierarchy:
//   - Master Key (stored in KMS) → encrypts → KEK (stored in DB) → encrypts → DEK
//
// Key rotation workflow:
//  1. Create a new KEK with an incremented version number
//  2. Mark the new KEK as active (IsActive = true)
//  3. Mark the old KEK as inactive (IsActive = false)
//  4. New DEKs will be encrypted with the new KEK
//  5. Old DEKs can still be decrypted with the old KEK until they are re-encrypted
//
// Fields:
//   - ID: Unique identifier for the KEK (UUIDv7 for time-based ordering)
//   - Name: Human-readable name for identifying the KEK (e.g., "production-kek-2025")
//   - Algorithm: Encryption algorithm used (AES-GCM or ChaCha20-Poly1305)
//   - EncryptedKey: The KEK encrypted with the master key (safe to store in DB)
//   - Key: The plaintext KEK (populated after decryption, should never be persisted)
//   - Nonce: Unique nonce used for encrypting the KEK with the master key
//   - Version: Version number for tracking KEK rotations (increments with each rotation)
//   - IsActive: Whether this KEK is currently active for encrypting new DEKs
//   - CreatedAt: Timestamp when the KEK was created
type Kek struct {
	ID           uuid.UUID
	Name         string
	Algorithm    Algorithm
	EncryptedKey []byte
	Key          []byte
	Nonce        []byte
	Version      uint
	IsActive     bool
	CreatedAt    time.Time
}
