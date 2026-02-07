// Package domain defines the core domain models and types for secret management.
//
// This package implements the domain layer for encrypted secret storage with
// automatic versioning. It provides the fundamental data structures and business
// rules for managing encrypted secrets in a hierarchical key management system.
//
// # Secret Versioning Model
//
// Secrets use an immutable versioning system where each update creates a new
// database row with an incremented version number:
//
//	Path: /app/api-key
//	├── Version 1 (id: uuid-1, value: "old-secret")
//	├── Version 2 (id: uuid-2, value: "updated-secret")
//	└── Version 3 (id: uuid-3, value: "latest-secret")
//
// Benefits:
//   - Complete audit trail of all changes
//   - Point-in-time recovery capability
//   - Cryptographic isolation (each version has its own DEK)
//   - Safe concurrent access patterns
//
// # Encryption Architecture
//
// Each secret version is encrypted using envelope encryption:
//
//	Master Key → KEK → DEK → Secret Data
//	     ↓         ↓      ↓        ↓
//	  KMS/Env   Database  DB    Encrypted
//
// Properties:
//   - Each version can use a different DEK
//   - DEKs are encrypted by KEKs
//   - KEKs are encrypted by master keys
//   - Efficient key rotation without re-encrypting secrets
//
// # Data Model
//
// The Secret struct represents an encrypted secret version with:
//   - ID: Unique identifier for this version (UUIDv7)
//   - Path: Logical path for secret lookup (e.g., "/app/database/password")
//   - Version: Monotonically increasing version number (1, 2, 3, ...)
//   - DekID: Reference to the DEK used for encryption
//   - Ciphertext: Encrypted secret data
//   - Nonce: Cryptographic nonce for AEAD encryption
//   - CreatedAt: Version creation timestamp
//   - DeletedAt: Soft deletion timestamp (optional)
//
// # Usage Example
//
//	// Create a new secret version
//	secret := &domain.Secret{
//	    ID:         uuid.Must(uuid.NewV7()),
//	    Path:       "/app/api-key",
//	    Version:    1,
//	    DekID:      dekID,
//	    Ciphertext: encryptedData,
//	    Nonce:      nonce,
//	    CreatedAt:  time.Now().UTC(),
//	}
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
