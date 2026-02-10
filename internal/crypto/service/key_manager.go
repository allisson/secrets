package service

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// KeyManagerService implements the KeyManager interface for envelope encryption.
type KeyManagerService struct {
	aeadManager AEADManager
}

// NewKeyManager creates a new KeyManagerService with the provided AEADManager.
func NewKeyManager(aeadManager AEADManager) *KeyManagerService {
	return &KeyManagerService{
		aeadManager: aeadManager,
	}
}

// CreateKek creates a new KEK encrypted with the master key.
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
		CreatedAt:    time.Now().UTC(),
	}

	return kek, nil
}

// DecryptKek decrypts a KEK using the master key.
func (km *KeyManagerService) DecryptKek(
	kek *cryptoDomain.Kek,
	masterKey *cryptoDomain.MasterKey,
) ([]byte, error) {
	// Create cipher using AEADManager with KEK's algorithm
	aead, err := km.aeadManager.CreateCipher(masterKey.Key, kek.Algorithm)
	if err != nil {
		return nil, err
	}

	// Decrypt the KEK with the master key
	kekKey, err := aead.Decrypt(kek.EncryptedKey, kek.Nonce, nil)
	if err != nil {
		return nil, cryptoDomain.ErrDecryptionFailed
	}

	return kekKey, nil
}

// CreateDek creates a new DEK encrypted with the KEK.
func (km *KeyManagerService) CreateDek(
	kek *cryptoDomain.Kek,
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

// DecryptDek decrypts a DEK using the KEK.
func (km *KeyManagerService) DecryptDek(
	dek *cryptoDomain.Dek,
	kek *cryptoDomain.Kek,
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
