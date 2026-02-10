package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// DekRepository defines the interface for DEK persistence operations.
type DekRepository interface {
	Create(ctx context.Context, dek *cryptoDomain.Dek) error
	Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error)
}

// TransitKeyRepository defines the interface for transit key persistence.
type TransitKeyRepository interface {
	Create(ctx context.Context, transitKey *transitDomain.TransitKey) error
	Delete(ctx context.Context, transitKeyID uuid.UUID) error
	GetByName(ctx context.Context, name string) (*transitDomain.TransitKey, error)
	GetByNameAndVersion(ctx context.Context, name string, version uint) (*transitDomain.TransitKey, error)
}

// TransitKeyUseCase defines the interface for transit encryption operations.
type TransitKeyUseCase interface {
	Create(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)
	Rotate(ctx context.Context, name string, alg cryptoDomain.Algorithm) (*transitDomain.TransitKey, error)
	Delete(ctx context.Context, transitKeyID uuid.UUID) error
	Encrypt(ctx context.Context, name string, plaintext []byte) (*transitDomain.EncryptedBlob, error)
	// Decrypt decrypts ciphertext using the version specified in the encrypted blob.
	//
	// Security Note: The returned EncryptedBlob contains plaintext data in the Plaintext field.
	// Callers MUST zero this data after use by calling cryptoDomain.Zero(blob.Plaintext).
	Decrypt(ctx context.Context, name string, ciphertext []byte) (*transitDomain.EncryptedBlob, error)
}
