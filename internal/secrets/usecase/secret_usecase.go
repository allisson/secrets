// Package usecase implements business logic orchestration for secret management.
//
// This package provides the use case layer (application layer) for managing
// encrypted secrets with automatic versioning, following Clean Architecture
// principles. Use cases coordinate between cryptographic services, repositories,
// and domain logic to implement secure secret storage and retrieval.
//
// # Key Components
//
// The package includes:
//   - SecretUseCase: Manages secret lifecycle with automatic versioning
//   - Interfaces: Defines contracts for repositories and dependencies
//
// # Automatic Versioning
//
// The use case implements automatic version management:
//   - CreateOrUpdate creates version 1 for new secrets
//   - CreateOrUpdate creates version N+1 for existing secrets
//   - Each version is a separate database row with its own DEK
//   - Complete audit trail maintained automatically
//
// # Encryption Flow
//
// Secret encryption follows envelope encryption pattern:
//
//  1. Generate new DEK for this secret version
//  2. Encrypt secret data with DEK (AEAD encryption)
//  3. Store encrypted secret with DEK reference
//  4. DEK is itself encrypted by KEK (handled by crypto layer)
//
// # Business Rules
//
// The use cases enforce business logic such as:
//   - Automatic version incrementing on updates
//   - Transactional consistency for multi-step operations
//   - Complete encryption/decryption workflow orchestration
//   - Error handling and propagation
//   - Soft deletion with timestamp tracking
//
// # Transaction Management
//
// All use cases use TxManager to ensure atomic operations:
//   - Secret creation with DEK generation is atomic
//   - Failed operations roll back automatically
//   - Consistent state guaranteed across operations
//
// # Usage Example
//
//	// Create use case
//	secretUseCase := usecase.NewSecretUseCase(
//	    txManager,
//	    secretRepo,
//	    dekRepo,
//	    keyManager,
//	    aeadManager,
//	)
//
//	// Create or update a secret (automatic versioning)
//	secret, err := secretUseCase.CreateOrUpdate(ctx, "/app/api-key", []byte("secret-value"))
//	// First call: Creates version 1
//	// Second call: Creates version 2
//	// Third call: Creates version 3
//
//	// Retrieve latest version
//	secret, err := secretUseCase.Get(ctx, "/app/api-key")
//
//	// Soft delete
//	err = secretUseCase.Delete(ctx, "/app/api-key")
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// secretUseCase implements the SecretUseCase interface for managing secrets.
//
// This use case orchestrates secret lifecycle operations including creation, updates,
// retrieval, and deletion. It coordinates between the DEK repository, secret repository,
// key management services, and KEK chain for cryptographic operations.
//
// The use case follows Clean Architecture principles by depending on abstractions
// (interfaces) rather than concrete implementations, enabling testability and
// flexibility in choosing different storage or cryptographic backends.
//
// Key operations:
//   - CreateOrUpdate: Creates a new secret or a new version of an existing secret
//   - Get: Retrieves and decrypts a secret by its path
//   - Delete: Soft deletes a secret by its path
type secretUseCase struct {
	txManager    database.TxManager
	dekRepo      DekRepository
	secretRepo   SecretRepository
	kekChain     *cryptoDomain.KekChain
	aeadManager  cryptoService.AEADManager
	keyManager   cryptoService.KeyManager
	dekAlgorithm cryptoDomain.Algorithm
}

// CreateOrUpdate creates a new secret or creates a new version of an existing secret.
//
// This method implements versioned secret management. When a secret at the given path
// already exists, a new version is created with an incremented version number, preserving
// the old version in the database. This approach maintains a complete audit trail of
// secret changes.
//
// The encryption process:
//  1. Attempts to retrieve an existing secret by path
//  2. If found, increments the version number for the new secret
//  3. Creates a new Data Encryption Key (DEK) encrypted with the active KEK
//  4. Encrypts the plaintext value with the DEK
//  5. Persists both the DEK and the encrypted secret to the database
//
// All operations are performed atomically within a database transaction to ensure
// consistency. If any step fails, the entire operation is rolled back.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - path: The secret path identifier (e.g., "/app/database/password")
//   - value: The plaintext secret value to encrypt and store
//
// Returns:
//   - The newly created Secret with encrypted data (Plaintext field is cleared)
//   - An error if the active KEK is not found, DEK creation fails,
//     encryption fails, or database persistence fails
//
// Example:
//
//	// Create a new secret
//	secret, err := secretUseCase.CreateOrUpdate(ctx, "/app/api-key", []byte("secret-value"))
//	if err != nil {
//	    log.Fatalf("Failed to create secret: %v", err)
//	}
//
//	// Update the secret (creates version 2)
//	secret, err = secretUseCase.CreateOrUpdate(ctx, "/app/api-key", []byte("new-value"))
//	if err != nil {
//	    log.Fatalf("Failed to update secret: %v", err)
//	}
func (s *secretUseCase) CreateOrUpdate(
	ctx context.Context,
	path string,
	value []byte,
) (*secretsDomain.Secret, error) {
	activeKek, found := s.kekChain.Get(s.kekChain.ActiveKekID())
	if !found {
		return nil, cryptoDomain.ErrKekNotFound
	}

	return s.createOrUpdateSecret(ctx, path, value, activeKek)
}

// createOrUpdateSecret is a helper method that handles the secret creation/update logic.
//
// This method encapsulates the core logic for creating or updating a secret with a
// specific KEK. It's separated from CreateOrUpdate to enable better testability and
// potential reuse in key rotation scenarios.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - path: The secret path identifier
//   - value: The plaintext secret value to encrypt
//   - kek: The Key Encryption Key to use for encrypting the new DEK
//
// Returns:
//   - The newly created Secret with encrypted data
//   - An error if any step in the process fails
func (s *secretUseCase) createOrUpdateSecret(
	ctx context.Context,
	path string,
	value []byte,
	kek *cryptoDomain.Kek,
) (*secretsDomain.Secret, error) {
	var version uint = 1

	// Check if secret already exists to determine the version
	existingSecret, err := s.secretRepo.GetByPath(ctx, path)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}
	if existingSecret != nil {
		version = existingSecret.Version + 1
	}

	// Execute the creation within a transaction
	var newSecret *secretsDomain.Secret
	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Create a new DEK for this secret
		dek, err := s.keyManager.CreateDek(kek, s.dekAlgorithm)
		if err != nil {
			return err
		}

		// Persist the DEK
		if err := s.dekRepo.Create(txCtx, &dek); err != nil {
			return err
		}

		// Decrypt the DEK first to get the plaintext key
		dekKey, err := s.keyManager.DecryptDek(&dek, kek)
		if err != nil {
			return err
		}

		// Create cipher with the decrypted DEK key
		cipher, err := s.aeadManager.CreateCipher(dekKey, s.dekAlgorithm)
		if err != nil {
			return err
		}

		// Encrypt the secret value
		ciphertext, nonce, err := cipher.Encrypt(value, nil)
		if err != nil {
			return err
		}

		// Create the secret entity
		newSecret = &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    version,
			DekID:      dek.ID,
			Ciphertext: ciphertext,
			Nonce:      nonce,
			CreatedAt:  time.Now().UTC(),
		}

		// Persist the secret
		if err := s.secretRepo.Create(txCtx, newSecret); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newSecret, nil
}

// Get retrieves and decrypts a secret by its path.
//
// This method fetches the most recent version of a secret at the specified path,
// retrieves and decrypts its Data Encryption Key (DEK), and then uses the DEK to
// decrypt the secret value. The decrypted plaintext is populated in the Plaintext
// field of the returned Secret.
//
// The decryption process:
//  1. Retrieves the secret by path (gets the latest version)
//  2. Retrieves the DEK associated with the secret
//  3. Retrieves the KEK needed to decrypt the DEK
//  4. Decrypts the DEK using the KEK
//  5. Uses the decrypted DEK to decrypt the secret value
//
// Important: The returned Secret contains plaintext sensitive data in memory.
// Ensure proper handling and clearing of this data when no longer needed.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - path: The secret path identifier to retrieve
//
// Returns:
//   - The Secret with the Plaintext field populated with the decrypted value
//   - ErrNotFound if the secret doesn't exist at the specified path
//   - An error if the DEK or KEK is not found, or decryption fails
//
// Example:
//
//	secret, err := secretUseCase.Get(ctx, "/app/api-key")
//	if err != nil {
//	    if errors.Is(err, apperrors.ErrNotFound) {
//	        log.Printf("Secret not found")
//	    }
//	    return err
//	}
//	// Use secret.Plaintext...
//	// Clear sensitive data when done
//	cryptoDomain.Zero(secret.Plaintext)
func (s *secretUseCase) Get(ctx context.Context, path string) (*secretsDomain.Secret, error) {
	// Retrieve the secret by path
	secret, err := s.secretRepo.GetByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	// Retrieve the DEK
	dek, err := s.dekRepo.Get(ctx, secret.DekID)
	if err != nil {
		return nil, err
	}

	// Retrieve the KEK needed to decrypt the DEK
	kek, found := s.kekChain.Get(dek.KekID)
	if !found {
		return nil, cryptoDomain.ErrKekNotFound
	}

	// Decrypt the DEK
	dekKey, err := s.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}

	// Create cipher with the decrypted DEK key
	cipher, err := s.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Decrypt the secret value
	plaintext, err := cipher.Decrypt(secret.Ciphertext, secret.Nonce, nil)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	// Populate the plaintext field
	secret.Plaintext = plaintext

	return secret, nil
}

// Delete performs a soft delete on a secret by its path.
//
// This method retrieves the current version of the secret at the specified path
// and marks it as deleted by setting the DeletedAt timestamp. The secret data
// is not physically removed from the database, preserving it for audit purposes
// or potential recovery.
//
// Note: This method only deletes the current (latest) version of the secret at
// the given path. Previous versions remain in the database unchanged.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - path: The secret path identifier to delete
//
// Returns:
//   - ErrNotFound if the secret doesn't exist at the specified path
//   - An error if the database operation fails
//
// Example:
//
//	err := secretUseCase.Delete(ctx, "/app/api-key")
//	if err != nil {
//	    if errors.Is(err, apperrors.ErrNotFound) {
//	        log.Printf("Secret not found")
//	    }
//	    return err
//	}
func (s *secretUseCase) Delete(ctx context.Context, path string) error {
	// Retrieve the secret by path to get its ID
	secret, err := s.secretRepo.GetByPath(ctx, path)
	if err != nil {
		return err
	}

	// Perform soft delete
	return s.secretRepo.Delete(ctx, secret.ID)
}

// NewSecretUseCase creates a new secret use case instance with the provided dependencies.
//
// This constructor follows dependency injection principles, allowing different
// implementations of TxManager, repositories, services, and key chains to be provided.
// This design enables testing with mocks and flexibility in choosing storage
// or cryptographic backends.
//
// Parameters:
//   - txManager: Transaction manager for atomic database operations
//   - dekRepo: Repository for DEK persistence
//   - secretRepo: Repository for secret persistence
//   - kekChain: Chain of Key Encryption Keys for DEK encryption/decryption
//   - aeadManager: Service for creating AEAD cipher instances
//   - keyManager: Service for key cryptographic operations
//   - dekAlgorithm: The encryption algorithm to use for new DEKs (AESGCM or ChaCha20)
//
// Returns:
//   - A fully initialized SecretUseCase ready for use
//
// Example:
//
//	db, _ := sql.Open("postgres", dsn)
//	txManager := database.NewTxManager(db)
//	dekRepo := cryptoRepository.NewPostgreSQLDekRepository(db)
//	secretRepo := secretsRepository.NewPostgreSQLSecretRepository(db)
//	aeadManager := cryptoService.NewAEADManager()
//	keyManager := cryptoService.NewKeyManager(aeadManager)
//
//	// Load KEK chain
//	masterKeyChain, _ := cryptoDomain.LoadMasterKeyChainFromEnv()
//	defer masterKeyChain.Close()
//	kekChain, _ := kekUseCase.Unwrap(ctx, masterKeyChain)
//	defer kekChain.Close()
//
//	secretUseCase := NewSecretUseCase(
//	    txManager,
//	    dekRepo,
//	    secretRepo,
//	    kekChain,
//	    aeadManager,
//	    keyManager,
//	    cryptoDomain.AESGCM,
//	)
func NewSecretUseCase(
	txManager database.TxManager,
	dekRepo DekRepository,
	secretRepo SecretRepository,
	kekChain *cryptoDomain.KekChain,
	aeadManager cryptoService.AEADManager,
	keyManager cryptoService.KeyManager,
	dekAlgorithm cryptoDomain.Algorithm,
) SecretUseCase {
	return &secretUseCase{
		txManager:    txManager,
		dekRepo:      dekRepo,
		secretRepo:   secretRepo,
		kekChain:     kekChain,
		aeadManager:  aeadManager,
		keyManager:   keyManager,
		dekAlgorithm: dekAlgorithm,
	}
}
