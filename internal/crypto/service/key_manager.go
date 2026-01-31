package service

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// KeyManagerService implements the KeyManager interface for envelope encryption.
//
// This service manages the lifecycle of Key Encryption Keys (KEKs) and Data Encryption Keys (DEKs)
// in a two-tier envelope encryption scheme:
//   - KEKs are encrypted with a master key
//   - DEKs are encrypted with KEKs
//   - Actual data is encrypted with DEKs
//
// This approach provides efficient key rotation and separation of concerns.
// The service uses AEADManager to create cipher instances, following dependency
// injection principles and separating concerns.
type KeyManagerService struct {
	aeadManager AEADManager
}

// NewKeyManager creates a new KeyManagerService instance with the provided AEADManager.
//
// The AEADManager is used to create cipher instances for encrypting and decrypting
// KEKs and DEKs. This dependency injection approach allows for better testability
// and separation of concerns.
//
// Parameters:
//   - aeadManager: The AEADManager used to create cipher instances
//
// Returns:
//   - A new KeyManagerService instance
func NewKeyManager(aeadManager AEADManager) *KeyManagerService {
	return &KeyManagerService{
		aeadManager: aeadManager,
	}
}

// CreateKek creates a new Key Encryption Key encrypted with the provided master key.
//
// The KEK is generated as a random 32-byte (256-bit) key and then encrypted using
// the master key with the specified algorithm. The encrypted KEK can be safely stored
// in a database or other persistent storage.
//
// The master key's ID is stored in the KEK for tracking which master key was used during
// encryption. This enables proper key rotation workflows where multiple master keys are
// maintained simultaneously.
//
// Parameters:
//   - masterKey: The MasterKey used to encrypt the KEK (contains ID and 32-byte key material)
//   - alg: The encryption algorithm to use (AESGCM or ChaCha20)
//
// Returns:
//   - A Kek struct containing the encrypted key, nonce, master key ID reference, and metadata
//   - An error if the algorithm is unsupported or encryption fails
func (km *KeyManagerService) CreateKek(
	masterKey *cryptoDomain.MasterKey,
	alg cryptoDomain.Algorithm,
) (cryptoDomain.Kek, error) {
	// Generate a random 32-byte KEK
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		return cryptoDomain.Kek{}, fmt.Errorf("failed to generate KEK: %w", err)
	}

	// Create cipher using AEADManager
	aead, err := km.aeadManager.CreateCipher(masterKey.Key, alg)
	if err != nil {
		return cryptoDomain.Kek{}, err
	}

	// Encrypt the KEK with the master key
	encryptedKey, nonce, err := aead.Encrypt(kekKey, nil)
	if err != nil {
		return cryptoDomain.Kek{}, fmt.Errorf("failed to encrypt KEK: %w", err)
	}

	kek := cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  masterKey.ID,
		Algorithm:    alg,
		EncryptedKey: encryptedKey,
		Key:          kekKey,
		Nonce:        nonce,
		Version:      1,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	return kek, nil
}

// CreateDek creates a new Data Encryption Key encrypted with the provided KEK.
//
// The DEK is generated as a random 32-byte (256-bit) key and then encrypted using
// the KEK with the specified algorithm. The encrypted DEK should be stored alongside
// the data it encrypts.
//
// Parameters:
//   - kek: The Key Encryption Key used to encrypt the DEK (must have Key field populated)
//   - alg: The encryption algorithm to use for the DEK (AESGCM or ChaCha20)
//
// Returns:
//   - A Dek struct containing the encrypted key, nonce, and metadata
//   - An error if the algorithm is unsupported or encryption fails
func (km *KeyManagerService) CreateDek(
	kek cryptoDomain.Kek,
	alg cryptoDomain.Algorithm,
) (cryptoDomain.Dek, error) {
	// Generate a random 32-byte DEK
	dekKey := make([]byte, 32)
	if _, err := rand.Read(dekKey); err != nil {
		return cryptoDomain.Dek{}, fmt.Errorf("failed to generate DEK: %w", err)
	}

	// Create cipher using AEADManager with KEK's algorithm
	aead, err := km.aeadManager.CreateCipher(kek.Key, kek.Algorithm)
	if err != nil {
		return cryptoDomain.Dek{}, err
	}

	// Encrypt the DEK with the KEK
	encryptedKey, nonce, err := aead.Encrypt(dekKey, nil)
	if err != nil {
		return cryptoDomain.Dek{}, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	dek := cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kek.ID,
		Algorithm:    alg,
		EncryptedKey: encryptedKey,
		Nonce:        nonce,
		CreatedAt:    time.Now().UTC(),
	}

	return dek, nil
}

// DecryptDek decrypts a Data Encryption Key using the provided Key Encryption Key.
//
// This method is used to recover the plaintext DEK so it can be used to decrypt
// application data. The decrypted DEK should be kept in memory only and never
// persisted in plaintext form.
//
// Parameters:
//   - dek: The encrypted Data Encryption Key to decrypt
//   - kek: The Key Encryption Key used to decrypt the DEK (must have Key field populated)
//
// Returns:
//   - The decrypted DEK as a byte slice (32 bytes)
//   - An error if decryption fails or authentication check fails
func (km *KeyManagerService) DecryptDek(
	dek cryptoDomain.Dek,
	kek cryptoDomain.Kek,
) ([]byte, error) {
	// Create cipher using AEADManager with KEK's algorithm
	aead, err := km.aeadManager.CreateCipher(kek.Key, kek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Decrypt the DEK with the KEK
	dekKey, err := aead.Decrypt(dek.EncryptedKey, dek.Nonce, nil)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	return dekKey, nil
}
