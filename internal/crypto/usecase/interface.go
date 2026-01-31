// Package usecase defines the business logic interfaces for cryptographic operations.
//
// This package contains interface definitions for repositories and use cases
// related to envelope encryption and key management. Implementations of these
// interfaces handle KEK and DEK management, key rotation, and encryption/decryption.
package usecase

import (
	"context"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// KekRepository defines the interface for Key Encryption Key persistence.
//
// This interface abstracts KEK storage operations, allowing different
// implementations for PostgreSQL, MySQL, or other data stores. It supports
// transaction-aware operations through context propagation, enabling atomic
// key rotation workflows.
//
// Implementation requirements:
//   - Support both direct database operations and transactional operations
//   - Handle UUID marshaling/unmarshaling for database compatibility
//   - Return KEKs ordered by version descending (newest first) from List()
//   - Be thread-safe for concurrent access
//
// Available implementations:
//   - PostgreSQLKekRepository: Uses native UUID and BYTEA types
//   - MySQLKekRepository: Uses BINARY(16) for UUIDs and BLOB for binary data
//
// Example usage:
//
//	// Outside transaction
//	err := repo.Create(ctx, kek)
//
//	// Within transaction (atomic rotation)
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    if err := repo.Update(txCtx, oldKek); err != nil {
//	        return err
//	    }
//	    return repo.Create(txCtx, newKek)
//	})
type KekRepository interface {
	// Create stores a new KEK in the repository.
	//
	// This method inserts a new Key Encryption Key into persistent storage.
	// The KEK should have all required fields populated: ID, MasterKeyID,
	// Algorithm, EncryptedKey, Nonce, Version, IsActive, and CreatedAt.
	//
	// The method supports transaction context. If the context contains a
	// transaction (via database.GetTx), the operation will participate in
	// that transaction.
	//
	// Parameters:
	//   - ctx: Context for cancellation, timeouts, and transaction propagation
	//   - kek: The KEK to store (must be fully populated)
	//
	// Returns:
	//   - An error if the operation fails (e.g., duplicate key, constraint violation)
	Create(ctx context.Context, kek *cryptoDomain.Kek) error

	// Update modifies an existing KEK in the repository.
	//
	// This method updates all mutable fields of an existing KEK. It's typically
	// used to deactivate old KEKs during key rotation by setting IsActive to false.
	// The KEK is identified by its ID field, which must match an existing record.
	//
	// The method supports transaction context. If the context contains a
	// transaction, the operation will participate in that transaction, enabling
	// atomic rotation operations.
	//
	// Parameters:
	//   - ctx: Context for cancellation, timeouts, and transaction propagation
	//   - kek: The KEK with updated field values (ID must match existing record)
	//
	// Returns:
	//   - An error if the operation fails (e.g., KEK not found, constraint violation)
	Update(ctx context.Context, kek *cryptoDomain.Kek) error

	// List retrieves all KEKs ordered by version descending.
	//
	// This method returns all KEKs (both active and inactive) from the repository,
	// sorted by version in descending order. The newest KEK (highest version) appears
	// first in the slice. This ordering is critical for creating a KekChain where
	// the first KEK becomes the active one.
	//
	// The method supports transaction context for consistent reads within a transaction.
	//
	// Parameters:
	//   - ctx: Context for cancellation, timeouts, and transaction propagation
	//
	// Returns:
	//   - A slice of KEK pointers ordered by version descending (newest first)
	//   - An error if the query fails
	List(ctx context.Context) ([]*cryptoDomain.Kek, error)
}

// KekUseCase defines the interface for Key Encryption Key business logic operations.
//
// This interface orchestrates KEK lifecycle management including creation, rotation,
// and decryption (unwrapping). It coordinates between the key manager service for
// cryptographic operations and the repository for persistence, implementing the
// business rules for envelope encryption key management.
//
// Key responsibilities:
//   - Create new KEKs encrypted with master keys
//   - Perform atomic KEK rotation (deactivate old, create new)
//   - Decrypt (unwrap) KEKs to create a KekChain for in-memory use
//   - Validate key relationships and ensure data integrity
//
// Implementation: kekUseCase (internal to this package)
//
// Example usage:
//
//	// Initialize dependencies
//	txManager := database.NewTxManager(db)
//	kekRepo := repository.NewPostgreSQLKekRepository(db)
//	aeadManager := service.NewAEADManager()
//	keyManager := service.NewKeyManager(aeadManager)
//	kekUseCase := NewKekUseCase(txManager, kekRepo, keyManager)
//
//	// Load master keys
//	masterKeyChain, err := cryptoDomain.LoadMasterKeyChainFromEnv()
//	if err != nil {
//	    return err
//	}
//	defer masterKeyChain.Close()
//
//	// Create initial KEK
//	err = kekUseCase.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)
//	if err != nil {
//	    return err
//	}
//
//	// Later, rotate to a new KEK
//	err = kekUseCase.Rotate(ctx, masterKeyChain, cryptoDomain.ChaCha20)
//	if err != nil {
//	    return err
//	}
//
//	// Load KEKs into memory for use
//	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
//	if err != nil {
//	    return err
//	}
//	defer kekChain.Close()
type KekUseCase interface {
	// Create generates and persists a new Key Encryption Key.
	//
	// This method creates the initial KEK for the system using the active master key
	// from the provided keychain. The generated KEK is encrypted with the master key
	// using the specified algorithm and stored in the database.
	//
	// The newly created KEK will have Version=1, IsActive=true, and will reference
	// the active master key ID from the chain.
	//
	// This method should be called once during system initialization. For subsequent
	// KEK updates, use Rotate() instead.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - masterKeyChain: The keychain containing the active master key
	//   - alg: The encryption algorithm to use (AESGCM or ChaCha20)
	//
	// Returns:
	//   - An error if the master key is not found, KEK generation fails,
	//     or database persistence fails
	Create(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error

	// Rotate performs a KEK rotation by creating a new KEK and deactivating the current one.
	//
	// Key rotation is a critical security operation that limits the exposure window if
	// a KEK is compromised. This method performs the rotation atomically within a
	// database transaction, ensuring that either both operations succeed or both fail.
	//
	// The rotation process:
	//  1. Retrieves the current active KEK (highest version)
	//  2. Marks the current KEK as inactive (IsActive = false)
	//  3. Generates a new KEK with version = current + 1
	//  4. Marks the new KEK as active (IsActive = true)
	//  5. Persists both changes atomically
	//
	// After rotation, new DEKs will be encrypted with the new KEK, while old DEKs
	// can still be decrypted using the old KEK until they are re-encrypted.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - masterKeyChain: The keychain containing the active master key
	//   - alg: The encryption algorithm for the new KEK (can differ from old KEK)
	//
	// Returns:
	//   - An error if the master key is not found, no current KEK exists,
	//     KEK generation fails, or the transaction fails
	Rotate(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain, alg cryptoDomain.Algorithm) error

	// Unwrap decrypts all KEKs from the database and returns them in a KekChain.
	//
	// This method retrieves all stored KEKs (both active and inactive) from the
	// repository and decrypts them using their corresponding master keys from the
	// provided keychain. The decrypted KEKs are assembled into a KekChain structure
	// for efficient in-memory access.
	//
	// The KekChain provides:
	//   - Quick access to the active KEK for encrypting new DEKs
	//   - Access to older KEKs for decrypting existing DEKs
	//   - Thread-safe concurrent access to all KEKs
	//
	// This method is typically called during application startup to load the KEK
	// chain into memory, avoiding repeated database queries and decryption operations.
	//
	// Important: The returned KekChain contains plaintext KEKs in memory. Ensure
	// proper memory protection and call Close() on the chain when it's no longer needed.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - masterKeyChain: The keychain containing master keys for decryption
	//
	// Returns:
	//   - A KekChain containing all decrypted KEKs with the active KEK identified
	//   - An error if database retrieval fails, a master key is missing,
	//     or KEK decryption fails
	Unwrap(ctx context.Context, masterKeyChain *cryptoDomain.MasterKeyChain) (*cryptoDomain.KekChain, error)
}
