package domain

import (
	"time"

	"github.com/google/uuid"
)

// Dek represents a Data Encryption Key used to encrypt application data.
// It is encrypted with a KEK and stored alongside the encrypted data.
type Dek struct {
	ID           uuid.UUID // Unique identifier (UUIDv7)
	KekID        uuid.UUID // Reference to the KEK used to encrypt this DEK
	Algorithm    Algorithm // Encryption algorithm (AESGCM or ChaCha20)
	EncryptedKey []byte    // The DEK encrypted with the KEK
	Nonce        []byte    // Unique nonce for encrypting the DEK
	CreatedAt    time.Time
}
