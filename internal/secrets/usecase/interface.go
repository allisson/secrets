// Package usecase defines the interfaces and implementations for secret management use cases.
//
// This package provides the business logic layer for managing encrypted secrets
// with automatic versioning. Use cases orchestrate operations between repositories
// (data persistence) and services (encryption/decryption), implementing business
// rules and transaction management.
//
// # Key Components
//
// The package includes:
//   - SecretUseCase: Main interface for secret operations (create, update, get, delete)
//   - SecretRepository: Interface for secret persistence
//   - DekRepository: Interface for DEK retrieval and management
//
// # Automatic Versioning
//
// The use case layer implements automatic versioning for secrets:
//   - First creation sets version to 1
//   - Updates increment version number (2, 3, 4, ...)
//   - Each version gets a new UUID and DEK for cryptographic isolation
//   - Previous versions remain unchanged for audit trail
//
// # Transaction Management
//
// All operations use TxManager to ensure atomic consistency:
//   - Create/Update operations atomically create DEK and secret
//   - Failed operations roll back automatically
//   - No partial state left in database
//
// # Usage Example
//
//	// Create use case
//	secretUseCase := usecase.NewSecretUseCase(txManager, secretRepo, dekRepo, keyManager, kekChain)
//
//	// Create or update a secret (automatic versioning)
//	secret, err := secretUseCase.CreateOrUpdate(ctx, "/app/api-key", []byte("secret-value"))
//
//	// Retrieve and decrypt the latest version
//	secret, err = secretUseCase.Get(ctx, "/app/api-key")
//	fmt.Println(string(secret.Plaintext))
//
//	// Soft delete the current version
//	err = secretUseCase.Delete(ctx, "/app/api-key")
package usecase

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// DekRepository defines the interface for Data Encryption Key persistence operations.
//
// This interface is used by SecretUseCase to create and retrieve DEKs for encrypting
// and decrypting secrets. Each secret version has its own DEK for cryptographic
// isolation.
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

// SecretRepository defines the interface for Secret persistence operations.
//
// This interface handles storing and retrieving encrypted secrets with automatic
// versioning support. Each secret version is stored as a separate database record
// with its own UUID and DEK reference.
//
// Methods:
//   - Create: Persists a new secret or secret version to the database
//   - GetByPath: Retrieves the latest version of a secret by its path
//   - Delete: Performs a soft delete on a secret version
//
// Implementation notes:
//   - Must support transaction-aware operations via context
//   - GetByPath should return the highest version number (current version)
//   - Should return apperrors.ErrNotFound when secret doesn't exist
//   - Delete performs soft deletion by setting deleted_at timestamp
type SecretRepository interface {
	Create(ctx context.Context, secret *secretsDomain.Secret) error
	Delete(ctx context.Context, secretID uuid.UUID) error
	GetByPath(ctx context.Context, path string) (*secretsDomain.Secret, error)
	GetByPathAndVersion(ctx context.Context, path string, version uint) (*secretsDomain.Secret, error)
}

// SecretUseCase defines the interface for secret management business logic.
//
// This interface provides high-level operations for managing encrypted secrets
// with automatic versioning and envelope encryption. All operations handle the
// complete encryption/decryption workflow using the KEK chain.
//
// Methods:
//   - CreateOrUpdate: Creates a new secret or new version with automatic versioning
//   - Get: Retrieves and decrypts the latest version of a secret
//   - Delete: Soft deletes the current version of a secret
//
// Behavior:
//   - CreateOrUpdate automatically detects if it's a new secret or update
//   - New secrets start at version 1
//   - Updates increment version number automatically
//   - Each version gets a new DEK for cryptographic isolation
//   - Get returns the secret with Plaintext field populated
//   - Delete marks only the current version as deleted
//
// Error handling:
//   - Returns apperrors.ErrNotFound when secret doesn't exist
//   - Returns domain errors for encryption/decryption failures
//   - All errors are wrapped with context for debugging
type SecretUseCase interface {
	CreateOrUpdate(ctx context.Context, path string, value []byte) (*secretsDomain.Secret, error)
	Get(ctx context.Context, path string) (*secretsDomain.Secret, error)
	Delete(ctx context.Context, path string) error
}
