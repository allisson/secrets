// Package domain defines the core cryptographic domain models and types.
//
// This package implements the domain layer for envelope encryption, a multi-tier
// key management scheme that provides efficient key rotation and enhanced security.
//
// # Envelope Encryption Architecture
//
// The package implements a three-tier key hierarchy:
//
//	Master Key (KMS/Environment)
//	    ↓ encrypts
//	Key Encryption Key (KEK - stored in database)
//	    ↓ encrypts
//	Data Encryption Key (DEK - stored with data)
//	    ↓ encrypts
//	Application Data
//
// # Key Components
//
// MasterKey: Root keys stored securely in KMS or environment variables.
// Used to encrypt and decrypt KEKs. Supports key rotation through MasterKeyChain.
//
// Kek (Key Encryption Key): Intermediate keys stored in the database.
// Used to encrypt and decrypt DEKs. Supports versioning and rotation.
//
// Dek (Data Encryption Key): Per-record encryption keys stored alongside data.
// Used to encrypt actual application data. One DEK per encrypted item.
//
// # Supported Algorithms
//
// The package supports two AEAD (Authenticated Encryption with Associated Data) algorithms:
//
//   - AESGCM: AES-256-GCM, optimal on systems with AES-NI hardware acceleration
//   - ChaCha20: ChaCha20-Poly1305, optimal on mobile devices and systems without AES-NI
//
// Both algorithms provide 256-bit security and authenticated encryption.
//
// # Error Handling
//
// All domain errors wrap standard errors from internal/errors for consistent
// error handling and HTTP status code mapping. Errors follow the pattern:
//
//	ErrSpecificError = errors.Wrap(errors.ErrInvalidInput, "specific context")
//
// # Security Features
//
//   - 256-bit keys for maximum security
//   - AEAD encryption for confidentiality and authenticity
//   - Key rotation support without re-encrypting all data
//   - Secure memory zeroing for sensitive key material
//   - Per-record encryption with unique DEKs
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
//   - MasterKeyID: ID of the master key used to encrypt this KEK (for key rotation tracking)
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
	MasterKeyID  string
	Name         string
	Algorithm    Algorithm
	EncryptedKey []byte
	Key          []byte
	Nonce        []byte
	Version      uint
	IsActive     bool
	CreatedAt    time.Time
}
