// Package usecase implements business logic orchestration for cryptographic operations.
//
// This package provides the use case layer (application layer) for managing
// cryptographic keys following Clean Architecture principles. Use cases coordinate
// between services (cryptographic operations) and repositories (data persistence),
// implementing business rules and transaction management.
//
// # Key Components
//
// The package includes:
//   - KekUseCase: Manages KEK lifecycle including creation, rotation, and unwrapping
//   - Interfaces: Defines contracts for repositories and dependencies
//
// # Business Rules
//
// The use cases enforce business logic such as:
//   - Active master key selection from keychains
//   - Transactional consistency for multi-step operations
//   - Key version management and rotation workflows
//   - Error handling and propagation
//
// # Transaction Management
//
// All use cases use TxManager to ensure atomic operations:
//   - KEK rotation updates old and new KEKs atomically
//   - Failed operations roll back automatically
//   - Consistent state guaranteed across operations
//
// # Usage Example
//
//	// Create use case
//	kekUseCase := usecase.NewKekUseCase(txManager, kekRepo, keyManager)
//
//	// Create initial KEK
//	err := kekUseCase.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)
//
//	// Rotate KEK to new master key
//	err = kekUseCase.Rotate(ctx, oldKekID, newMasterKeyChain, cryptoDomain.AESGCM)
package usecase

import (
	"context"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
)

// kekUseCase implements the KekUseCase interface for managing Key Encryption Keys.
//
// This use case orchestrates KEK lifecycle operations including creation, rotation,
// and decryption (unwrapping). It coordinates between the key manager service for
// cryptographic operations and the repository for persistence.
//
// The use case follows Clean Architecture principles by depending on abstractions
// (interfaces) rather than concrete implementations, enabling testability and
// flexibility in choosing different storage or cryptographic backends.
type kekUseCase struct {
	txManager  database.TxManager
	kekRepo    KekRepository
	keyManager cryptoService.KeyManager
}

// getMasterKey retrieves a master key from the chain by its ID.
//
// This helper method provides a consistent way to access master keys with
// proper error handling. It's used internally by Create, Rotate, and Unwrap
// methods to obtain the appropriate master key for KEK operations.
//
// Parameters:
//   - masterKeyChain: The keychain containing available master keys
//   - id: The unique identifier of the master key to retrieve
//
// Returns:
//   - The MasterKey if found
//   - An error if the master key ID is not found in the keychain
func (k *kekUseCase) getMasterKey(
	masterKeyChain *cryptoDomain.MasterKeyChain, id string,
) (*cryptoDomain.MasterKey, error) {
	masterKey, ok := masterKeyChain.Get(id)
	if !ok {
		return nil, cryptoDomain.ErrMasterKeyNotFound
	}
	return masterKey, nil
}

// Create generates and persists a new Key Encryption Key.
//
// This method creates the initial KEK for the system using the active master key
// from the provided keychain. The generated KEK is encrypted with the master key
// using the specified algorithm and stored in the database.
//
// The newly created KEK will have:
//   - Version: 1
//   - Algorithm: The specified algorithm (AESGCM or ChaCha20)
//   - MasterKeyID: The active master key's ID
//
// This method should be called once during system initialization. For subsequent
// KEK updates, use the Rotate method instead.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - masterKeyChain: The keychain containing the active master key
//   - alg: The encryption algorithm to use for the KEK
//
// Returns:
//   - An error if the master key is not found, KEK generation fails,
//     or database persistence fails
//
// Example:
//
//	masterKeyChain, _ := cryptoDomain.LoadMasterKeyChainFromEnv()
//	defer masterKeyChain.Close()
//
//	err := kekUseCase.Create(ctx, masterKeyChain, cryptoDomain.AESGCM)
//	if err != nil {
//	    log.Fatalf("Failed to create KEK: %v", err)
//	}
func (k *kekUseCase) Create(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	alg cryptoDomain.Algorithm,
) error {
	masterKey, err := k.getMasterKey(masterKeyChain, masterKeyChain.ActiveMasterKeyID())
	if err != nil {
		return err
	}

	kek, err := k.keyManager.CreateKek(masterKey, alg)
	if err != nil {
		return err
	}

	return k.kekRepo.Create(ctx, &kek)
}

// Rotate performs a KEK rotation by creating a new KEK with an incremented version.
//
// Key rotation is a critical security operation that limits the exposure window if
// a KEK is compromised. This method performs the rotation atomically within a
// database transaction, ensuring that either all operations succeed or all fail.
//
// The rotation process:
//  1. Retrieves all KEKs ordered by version (descending)
//  2. If no KEKs exist, creates the first KEK with version 1 (delegates to Create)
//  3. If KEKs exist, generates a new KEK with version = current + 1
//  4. Persists the new KEK to the database
//
// This dual behavior makes Rotate a safe operation that can be called whether or not
// KEKs have been initialized, simplifying application startup and key management workflows.
//
// After rotation, new Data Encryption Keys (DEKs) will be encrypted with the new
// KEK, while old DEKs can still be decrypted using the old KEK until they are
// re-encrypted.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - masterKeyChain: The keychain containing the active master key
//   - alg: The encryption algorithm for the new KEK (can differ from old KEK)
//
// Returns:
//   - An error if the master key is not found, KEK generation fails,
//     or the transaction fails
//
// Example:
//
//	// Rotate to a new KEK (creates first KEK if none exist)
//	err := kekUseCase.Rotate(ctx, masterKeyChain, cryptoDomain.AESGCM)
//	if err != nil {
//	    log.Fatalf("KEK rotation failed: %v", err)
//	}
//
//	// Rotate and change algorithm
//	err = kekUseCase.Rotate(ctx, masterKeyChain, cryptoDomain.ChaCha20)
func (k *kekUseCase) Rotate(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	alg cryptoDomain.Algorithm,
) error {
	masterKey, err := k.getMasterKey(masterKeyChain, masterKeyChain.ActiveMasterKeyID())
	if err != nil {
		return err
	}

	return k.txManager.WithTx(ctx, func(ctx context.Context) error {
		keks, err := k.kekRepo.List(ctx)
		if err != nil {
			return err
		}

		// We don't have any registered keks, we created a new one.
		if len(keks) == 0 {
			return k.Create(ctx, masterKeyChain, alg)
		}

		currentKek := keks[0]

		kek, err := k.keyManager.CreateKek(masterKey, alg)
		if err != nil {
			return err
		}

		kek.Version = currentKek.Version + 1
		return k.kekRepo.Create(ctx, &kek)
	})
}

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
//
// Example:
//
//	// Load KEKs at startup
//	masterKeyChain, _ := cryptoDomain.LoadMasterKeyChainFromEnv()
//	defer masterKeyChain.Close()
//
//	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
//	if err != nil {
//	    log.Fatalf("Failed to unwrap KEKs: %v", err)
//	}
//	defer kekChain.Close()
//
//	// Get active KEK for encrypting new DEKs
//	activeKek, _ := kekChain.Get(kekChain.ActiveKekID())
func (k *kekUseCase) Unwrap(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
) (*cryptoDomain.KekChain, error) {
	keks, err := k.kekRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, kek := range keks {
		masterKey, err := k.getMasterKey(masterKeyChain, kek.MasterKeyID)
		if err != nil {
			return nil, err
		}
		key, err := k.keyManager.DecryptKek(kek, masterKey)
		if err != nil {
			return nil, err
		}
		kek.Key = key
	}

	kekChain := cryptoDomain.NewKekChain(keks)

	return kekChain, nil
}

// NewKekUseCase creates a new KEK use case instance with the provided dependencies.
//
// This constructor follows dependency injection principles, allowing different
// implementations of TxManager, KekRepository, and KeyManager to be provided.
// This design enables testing with mocks and flexibility in choosing storage
// or cryptographic backends.
//
// Parameters:
//   - txManager: Transaction manager for atomic database operations
//   - kekRepo: Repository for KEK persistence (PostgreSQL or MySQL)
//   - keyManager: Service for KEK cryptographic operations
//
// Returns:
//   - A fully initialized KekUseCase ready for use
//
// Example:
//
//	db, _ := sql.Open("postgres", dsn)
//	txManager := database.NewTxManager(db)
//	kekRepo := repository.NewPostgreSQLKekRepository(db)
//	aeadManager := service.NewAEADManager()
//	keyManager := service.NewKeyManager(aeadManager)
//
//	kekUseCase := NewKekUseCase(txManager, kekRepo, keyManager)
func NewKekUseCase(
	txManager database.TxManager,
	kekRepo KekRepository,
	keyManager cryptoService.KeyManager,
) KekUseCase {
	return &kekUseCase{
		txManager:  txManager,
		kekRepo:    kekRepo,
		keyManager: keyManager,
	}
}
