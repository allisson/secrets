package domain

import (
	"time"

	"github.com/google/uuid"
)

// Dek represents a Data Encryption Key used in envelope encryption.
//
// A DEK is a cryptographic key used to encrypt actual application data.
// The DEK is encrypted with a Key Encryption Key (KEK) and stored alongside
// the encrypted data. To decrypt data, the DEK must first be decrypted using
// the KEK (which requires the master key).
//
// In envelope encryption hierarchy:
//   - Master Key → decrypts → KEK → decrypts → DEK → decrypts → Application Data
//
// This multi-layer approach provides several benefits:
//   - Fast key rotation: Only the KEK needs to be rotated, not all DEKs
//   - Performance: DEKs can be cached in memory after decryption
//   - Security: Master key is never directly used to encrypt data
//   - Crypto shredding: Delete a DEK to make its data unrecoverable
//   - Per-record encryption: Each data record can have its own DEK
//
// Best practices:
//   - Create a unique DEK for each piece of data to encrypt
//   - Store the encrypted DEK alongside the encrypted data (e.g., same DB record)
//   - Cache decrypted DEKs in memory with expiration for performance
//   - Never persist the plaintext DEK to disk or database
//
// Fields:
//   - ID: Unique identifier for the DEK (UUIDv7 for time-based ordering)
//   - KekID: Reference to the KEK used to encrypt this DEK
//   - Algorithm: Encryption algorithm for the actual data (AES-GCM or ChaCha20-Poly1305)
//   - EncryptedKey: The DEK encrypted with the KEK (safe to store in DB)
//   - Nonce: Unique nonce used for encrypting the DEK with the KEK
//   - CreatedAt: Timestamp when the DEK was created
type Dek struct {
	ID           uuid.UUID
	KekID        uuid.UUID
	Algorithm    Algorithm
	EncryptedKey []byte
	Nonce        []byte
	CreatedAt    time.Time
}
