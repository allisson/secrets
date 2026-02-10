// Package usecase implements business logic orchestration for transit encryption operations.
//
// This package provides the use case layer (application layer) for managing transit
// encryption keys following Clean Architecture principles. Use cases coordinate between
// services (cryptographic operations) and repositories (data persistence), implementing
// business rules and transaction management.
//
// # Key Components
//
// The package includes:
//   - TransitKeyUseCase: Manages transit key lifecycle and encryption/decryption operations
//   - Interfaces: Defines contracts for repositories and dependencies
//
// # Business Rules
//
// The use cases enforce business logic such as:
//   - Automatic versioning for key rotation
//   - Latest version selection for encryption operations
//   - Version-specific decryption from encrypted blob metadata
//   - Transactional consistency for multi-step operations
//
// # Transit Encryption
//
// Transit encryption allows clients to encrypt and decrypt data without exposing key
// material. The key hierarchy is:
//
//	Master Key → KEK → DEK → Transit Key (named, versioned)
//	                           ↓
//	                    Encrypt/Decrypt user data
//
// Each transit key version has its own DEK for cryptographic isolation, enabling
// secure key rotation without re-encrypting existing data.
//
// # Transaction Management
//
// All use cases use TxManager to ensure atomic operations:
//   - Key rotation updates are atomic (create new version)
//   - Failed operations roll back automatically
//   - Consistent state guaranteed across operations
//
// # Usage Example
//
//	// Create use case
//	transitKeyUC := usecase.NewTransitKeyUseCase(
//	    txManager, transitRepo, dekRepo, keyManager, aeadManager, kekChain,
//	)
//
//	// Create a new transit key
//	key, err := transitKeyUC.Create(ctx, "payment-key", cryptoDomain.AESGCM)
//
//	// Encrypt data
//	blob, err := transitKeyUC.Encrypt(ctx, "payment-key", []byte("sensitive data"))
//	fmt.Println(blob.String()) // "1:base64-ciphertext"
//
//	// Decrypt data
//	blob, err = transitKeyUC.Decrypt(ctx, "payment-key", []byte(blob.String()))
//	fmt.Println(string(blob.Plaintext))
//
//	// Rotate key to new version
//	newKey, err := transitKeyUC.Rotate(ctx, "payment-key", cryptoDomain.AESGCM)
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// transitKeyUseCase implements the TransitKeyUseCase interface for managing transit keys.
//
// This use case orchestrates transit key lifecycle operations including creation, rotation,
// deletion, and encryption/decryption. It coordinates between the key manager service for
// cryptographic operations, repositories for persistence, and the KEK chain for accessing
// encryption keys.
//
// The use case follows Clean Architecture principles by depending on abstractions
// (interfaces) rather than concrete implementations, enabling testability and
// flexibility in choosing different storage or cryptographic backends.
type transitKeyUseCase struct {
	txManager   database.TxManager
	transitRepo TransitKeyRepository
	dekRepo     DekRepository
	keyManager  cryptoService.KeyManager
	aeadManager cryptoService.AEADManager
	kekChain    *cryptoDomain.KekChain
}

// getKek retrieves a KEK from the chain by its ID.
//
// This helper method provides a consistent way to access KEKs with proper error
// handling. It's used internally by methods that need to decrypt DEKs.
//
// Parameters:
//   - kekID: The unique identifier of the KEK to retrieve
//
// Returns:
//   - The Kek if found
//   - An error if the KEK ID is not found in the keychain
func (t *transitKeyUseCase) getKek(kekID uuid.UUID) (*cryptoDomain.Kek, error) {
	kek, ok := t.kekChain.Get(kekID)
	if !ok {
		return nil, cryptoDomain.ErrKekNotFound
	}
	return kek, nil
}

// Create generates and persists a new transit key with version 1.
//
// This method creates a new named transit key for encryption/decryption operations.
// It generates a DEK encrypted with the active KEK, stores the DEK, and creates a
// transit key referencing the DEK.
//
// The newly created transit key will have:
//   - Version: 1
//   - Algorithm: The specified algorithm (AESGCM or ChaCha20)
//   - DekID: Reference to the newly created DEK
//
// This method should be called once per named key. For subsequent key updates,
// use the Rotate method instead.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - name: The name identifier for the transit key (e.g., "payment-key")
//   - alg: The encryption algorithm to use for the DEK
//
// Returns:
//   - The created TransitKey with all fields populated
//   - An error if the active KEK is not found, DEK creation fails,
//     or database persistence fails
//
// Example:
//
//	transitKey, err := transitKeyUC.Create(ctx, "payment-key", cryptoDomain.AESGCM)
//	if err != nil {
//	    log.Fatalf("Failed to create transit key: %v", err)
//	}
//	fmt.Printf("Created transit key: %s version %d\n", transitKey.Name, transitKey.Version)
func (t *transitKeyUseCase) Create(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
	// Get active KEK from chain
	activeKek, err := t.getKek(t.kekChain.ActiveKekID())
	if err != nil {
		return nil, err
	}

	// Create DEK encrypted with active KEK
	dek, err := t.keyManager.CreateDek(activeKek, alg)
	if err != nil {
		return nil, err
	}

	// Persist DEK to database
	if err := t.dekRepo.Create(ctx, &dek); err != nil {
		return nil, err
	}

	// Create transit key with version 1
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Version:   1,
		DekID:     dek.ID,
		CreatedAt: time.Now().UTC(),
	}

	// Persist transit key
	if err := t.transitRepo.Create(ctx, transitKey); err != nil {
		return nil, err
	}

	return transitKey, nil
}

// Rotate performs a transit key rotation by creating a new version.
//
// Key rotation is a critical security operation that limits the exposure window if
// a key is compromised. This method creates a new transit key version with an
// incremented version number and a new DEK.
//
// The rotation process:
//  1. Retrieves the latest transit key version by name
//  2. If no key exists, creates the first version (delegates to Create)
//  3. Creates a new DEK encrypted with the active KEK
//  4. Creates a new transit key with version = current + 1
//  5. Persists the new DEK and transit key atomically
//
// After rotation, encryption operations will use the new version, while old
// encrypted data can still be decrypted using the old version until re-encrypted.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - name: The name of the transit key to rotate
//   - alg: The encryption algorithm for the new version (can differ from old version)
//
// Returns:
//   - The new TransitKey with incremented version
//   - An error if the active KEK is not found, DEK creation fails,
//     or the transaction fails
//
// Example:
//
//	// Rotate to a new version
//	newKey, err := transitKeyUC.Rotate(ctx, "payment-key", cryptoDomain.AESGCM)
//	if err != nil {
//	    log.Fatalf("Transit key rotation failed: %v", err)
//	}
//	fmt.Printf("Rotated to version %d\n", newKey.Version)
func (t *transitKeyUseCase) Rotate(
	ctx context.Context,
	name string,
	alg cryptoDomain.Algorithm,
) (*transitDomain.TransitKey, error) {
	var newTransitKey *transitDomain.TransitKey

	err := t.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Get latest transit key version
		currentKey, err := t.transitRepo.GetByName(txCtx, name)
		if err != nil {
			// If key doesn't exist, create first version
			if apperrors.Is(err, transitDomain.ErrTransitKeyNotFound) {
				newTransitKey, err = t.Create(txCtx, name, alg)
				return err
			}
			return err
		}

		// Get active KEK from chain
		activeKek, err := t.getKek(t.kekChain.ActiveKekID())
		if err != nil {
			return err
		}

		// Create new DEK encrypted with active KEK
		dek, err := t.keyManager.CreateDek(activeKek, alg)
		if err != nil {
			return err
		}

		// Persist new DEK
		if err := t.dekRepo.Create(txCtx, &dek); err != nil {
			return err
		}

		// Create new transit key with incremented version
		newTransitKey = &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   currentKey.Version + 1,
			DekID:     dek.ID,
			CreatedAt: time.Now().UTC(),
		}

		// Persist new transit key
		return t.transitRepo.Create(txCtx, newTransitKey)
	})

	if err != nil {
		return nil, err
	}

	return newTransitKey, nil
}

// Delete soft-deletes a transit key by setting its deleted_at timestamp.
//
// This method performs a soft delete, marking the transit key as deleted while
// preserving historical data. Deleted keys cannot be used for new operations
// but remain in the database for audit purposes.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - transitKeyID: The UUID of the transit key to soft-delete
//
// Returns:
//   - An error if the delete operation fails
//
// Example:
//
//	err := transitKeyUC.Delete(ctx, transitKeyID)
//	if err != nil {
//	    log.Fatalf("Failed to delete transit key: %v", err)
//	}
func (t *transitKeyUseCase) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	return t.transitRepo.Delete(ctx, transitKeyID)
}

// Encrypt encrypts plaintext using the latest version of a named transit key.
//
// This method retrieves the latest version of the specified transit key, decrypts
// its associated DEK, and uses it to encrypt the plaintext. The resulting encrypted
// blob includes the version number, enabling version-aware decryption.
//
// The encryption process:
//  1. Retrieves the latest transit key version by name
//  2. Retrieves the DEK associated with the transit key
//  3. Decrypts the DEK using its associated KEK
//  4. Creates an AEAD cipher with the decrypted DEK
//  5. Encrypts the plaintext with the cipher
//  6. Returns an EncryptedBlob with version and ciphertext
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - name: The name of the transit key to use for encryption
//   - plaintext: The data to encrypt
//
// Returns:
//   - An EncryptedBlob with Version and Ciphertext populated (Plaintext is nil)
//   - An error if the transit key is not found, DEK retrieval fails,
//     KEK is not found, or encryption fails
//
// Example:
//
//	blob, err := transitKeyUC.Encrypt(ctx, "payment-key", []byte("4111111111111111"))
//	if err != nil {
//	    log.Fatalf("Encryption failed: %v", err)
//	}
//	// Store blob.String() which returns "version:base64-ciphertext"
//	ciphertext := blob.String()
func (t *transitKeyUseCase) Encrypt(
	ctx context.Context,
	name string,
	plaintext []byte,
) (*transitDomain.EncryptedBlob, error) {
	// Get latest transit key version
	transitKey, err := t.transitRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get DEK by transit key's DekID
	dek, err := t.dekRepo.Get(ctx, transitKey.DekID)
	if err != nil {
		return nil, err
	}

	// Get KEK for decrypting DEK
	kek, err := t.getKek(dek.KekID)
	if err != nil {
		return nil, err
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Encrypt plaintext
	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to encrypt plaintext")
	}

	// Combine ciphertext and nonce (nonce is prepended to ciphertext by AEAD)
	// The AEAD Encrypt returns ciphertext with authentication tag, we need to store nonce separately
	//nolint:gocritic // intentionally creating new slice with combined nonce and ciphertext
	encryptedData := append(nonce, ciphertext...)

	return &transitDomain.EncryptedBlob{
		Version:    transitKey.Version,
		Ciphertext: encryptedData,
		Plaintext:  nil,
	}, nil
}

// Decrypt decrypts ciphertext using the version specified in the encrypted blob.
//
// This method parses the encrypted blob to extract the version number, retrieves
// the corresponding transit key version, decrypts the DEK, and uses it to decrypt
// the ciphertext. This enables decryption of data encrypted with older key versions
// after rotation.
//
// The decryption process:
//  1. Parses the ciphertext bytes as an EncryptedBlob (format: "version:base64-ciphertext")
//  2. Retrieves the transit key by name and version from the blob
//  3. Retrieves the DEK associated with that transit key version
//  4. Decrypts the DEK using its associated KEK
//  5. Creates an AEAD cipher with the decrypted DEK
//  6. Decrypts the ciphertext with the cipher
//  7. Returns an EncryptedBlob with version and plaintext
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - name: The name of the transit key used for encryption
//   - ciphertext: The encrypted data in EncryptedBlob format (from blob.String())
//
// Returns:
//   - An EncryptedBlob with Version and Plaintext populated (Ciphertext is nil)
//   - An error if the blob format is invalid, transit key is not found,
//     DEK retrieval fails, KEK is not found, or decryption fails
//
// Example:
//
//	// Decrypt data encrypted with any version of the key
//	blob, err := transitKeyUC.Decrypt(ctx, "payment-key", []byte("1:SGVsbG8gV29ybGQ="))
//	if err != nil {
//	    log.Fatalf("Decryption failed: %v", err)
//	}
//	fmt.Printf("Plaintext: %s\n", string(blob.Plaintext))
func (t *transitKeyUseCase) Decrypt(
	ctx context.Context,
	name string,
	ciphertext []byte,
) (*transitDomain.EncryptedBlob, error) {
	// Parse encrypted blob from ciphertext
	blob, err := transitDomain.NewEncryptedBlob(string(ciphertext))
	if err != nil {
		return nil, err
	}

	// Get transit key by name and version from blob
	transitKey, err := t.transitRepo.GetByNameAndVersion(ctx, name, blob.Version)
	if err != nil {
		return nil, err
	}

	// Get DEK by transit key's DekID
	dek, err := t.dekRepo.Get(ctx, transitKey.DekID)
	if err != nil {
		return nil, err
	}

	// Get KEK for decrypting DEK
	kek, err := t.getKek(dek.KekID)
	if err != nil {
		return nil, err
	}

	// Decrypt DEK with KEK
	dekKey, err := t.keyManager.DecryptDek(dek, kek)
	if err != nil {
		return nil, err
	}

	// Create AEAD cipher with decrypted DEK
	cipher, err := t.aeadManager.CreateCipher(dekKey, dek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Extract nonce and ciphertext from encrypted data
	// The nonce is prepended to the ciphertext
	nonceSize := 12 // Standard nonce size for AES-GCM and ChaCha20-Poly1305
	if len(blob.Ciphertext) < nonceSize {
		return nil, apperrors.Wrap(cryptoDomain.ErrDecryptionFailed, "ciphertext too short")
	}

	nonce := blob.Ciphertext[:nonceSize]
	encryptedData := blob.Ciphertext[nonceSize:]

	// Decrypt ciphertext
	plaintext, err := cipher.Decrypt(encryptedData, nonce, nil)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	return &transitDomain.EncryptedBlob{
		Version:    blob.Version,
		Ciphertext: nil,
		Plaintext:  plaintext,
	}, nil
}

// NewTransitKeyUseCase creates a new transit key use case instance with the provided dependencies.
//
// This constructor follows dependency injection principles, allowing different
// implementations of the required interfaces to be provided. This design enables
// testing with mocks and flexibility in choosing storage or cryptographic backends.
//
// Parameters:
//   - txManager: Transaction manager for atomic database operations
//   - transitRepo: Repository for transit key persistence (PostgreSQL or MySQL)
//   - dekRepo: Repository for DEK persistence
//   - keyManager: Service for DEK cryptographic operations
//   - aeadManager: Service for AEAD cipher creation
//   - kekChain: Chain of KEKs for accessing active and historical KEKs
//
// Returns:
//   - A fully initialized TransitKeyUseCase ready for use
//
// Example:
//
//	db, _ := sql.Open("postgres", dsn)
//	txManager := database.NewTxManager(db)
//	transitRepo := repository.NewPostgreSQLTransitKeyRepository(db)
//	dekRepo := repository.NewPostgreSQLDekRepository(db)
//	aeadManager := service.NewAEADManager()
//	keyManager := service.NewKeyManager(aeadManager)
//	kekChain, _ := kekUseCase.Unwrap(ctx, masterKeyChain)
//
//	transitKeyUC := NewTransitKeyUseCase(
//	    txManager, transitRepo, dekRepo, keyManager, aeadManager, kekChain,
//	)
func NewTransitKeyUseCase(
	txManager database.TxManager,
	transitRepo TransitKeyRepository,
	dekRepo DekRepository,
	keyManager cryptoService.KeyManager,
	aeadManager cryptoService.AEADManager,
	kekChain *cryptoDomain.KekChain,
) TransitKeyUseCase {
	return &transitKeyUseCase{
		txManager:   txManager,
		transitRepo: transitRepo,
		dekRepo:     dekRepo,
		keyManager:  keyManager,
		aeadManager: aeadManager,
		kekChain:    kekChain,
	}
}
