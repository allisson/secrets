package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// DekRepository defines the interface for Data Encryption Key persistence operations.
//
// This interface is used by TransitKeyUseCase to create and retrieve DEKs for
// transit key operations. Each transit key references a DEK that encrypts/decrypts
// the actual user data.
//
// Methods:
//   - Create: Persists a new encrypted DEK to the database
//   - Get: Retrieves an encrypted DEK by its UUID for decryption operations
//
// Implementation notes:
//   - Must support transaction-aware operations via context
//   - Should return cryptoDomain.ErrDekNotFound when DEK doesn't exist
//   - DEKs are stored encrypted by KEKs (envelope encryption)
type DekRepository interface {
	Create(ctx context.Context, dek *cryptoDomain.Dek) error
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

// TransitKeyRepository defines the interface for transit key persistence operations.
//
// This interface handles storing and retrieving transit encryption keys with versioning
// support. Transit keys are named keys that enable encryption/decryption operations
// without exposing key material to clients.
//
// Methods:
//   - Create: Persists a new transit key version to the database
//   - Delete: Performs a soft delete on a transit key
//   - GetByName: Retrieves the latest version of a transit key by name
//   - GetByNameAndVersion: Retrieves a specific version of a transit key
//
// Implementation notes:
//   - Must support transaction-aware operations via context
//   - GetByName returns the highest version (current version)
//   - Should return transitDomain.ErrTransitKeyNotFound when key doesn't exist
//   - Delete performs soft deletion by setting deleted_at timestamp
type TransitKeyRepository interface {
	Create(ctx context.Context, transitKey *transitDomain.TransitKey) error
	Delete(ctx context.Context, transitKeyID uuid.UUID) error
	GetByName(ctx context.Context, name string) (*transitDomain.TransitKey, error)
	GetByNameAndVersion(ctx context.Context, name string, version uint) (*transitDomain.TransitKey, error)
}

// TransitKeyUseCase defines the interface for transit encryption business logic.
//
// This interface provides high-level operations for managing transit encryption keys
// with automatic versioning and envelope encryption. Transit keys enable clients to
// encrypt/decrypt data without exposing the actual key material.
//
// Methods:
//   - Create: Creates a new transit key with version 1
//   - Rotate: Creates a new version of an existing transit key
//   - Delete: Soft deletes a transit key
//   - Encrypt: Encrypts plaintext using the latest version of a named key
//   - Decrypt: Decrypts ciphertext using the version specified in the encrypted blob
//
// Behavior:
//   - Create initializes a new named key at version 1
//   - Rotate increments the version number for key rotation
//   - Encrypt always uses the latest version of the key
//   - Decrypt uses the version embedded in the ciphertext
//   - Each transit key version has its own DEK for cryptographic isolation
//
// Error handling:
//   - Returns transitDomain.ErrTransitKeyNotFound when key doesn't exist
//   - Returns domain errors for encryption/decryption failures
//   - All errors are wrapped with context for debugging
type TransitKeyUseCase interface {
	Create(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)
	Rotate(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)
	Delete(ctx context.Context, transitKeyID uuid.UUID) error
	Encrypt(ctx context.Context, name string, plaintext []byte) (*transitDomain.EncryptedBlob, error)
	Decrypt(ctx context.Context, name string, ciphertext []byte) (*transitDomain.EncryptedBlob, error)
}
