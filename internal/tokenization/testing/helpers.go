// Package testing provides shared test utilities for tokenization module tests.
package testing

import (
	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// CreateMasterKey creates a test master key with a random 32-byte key.
func CreateMasterKey() *cryptoDomain.MasterKey {
	return &cryptoDomain.MasterKey{
		ID:  "test-master-key",
		Key: make([]byte, 32),
	}
}

// CreateKekChain creates a test KEK chain with a single active KEK.
func CreateKekChain(masterKey *cryptoDomain.MasterKey) *cryptoDomain.KekChain {
	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  masterKey.ID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: make([]byte, 32),
		Key:          make([]byte, 32),
		Nonce:        make([]byte, 12),
		Version:      1,
	}
	return cryptoDomain.NewKekChain([]*cryptoDomain.Kek{kek})
}

// GetActiveKek retrieves the active KEK from a chain.
func GetActiveKek(kekChain *cryptoDomain.KekChain) *cryptoDomain.Kek {
	activeID := kekChain.ActiveKekID()
	kek, ok := kekChain.Get(activeID)
	if !ok {
		panic("active KEK not found in chain")
	}
	return kek
}

// CreateTestDek creates a test DEK for the given KEK.
func CreateTestDek(kek *cryptoDomain.Kek) cryptoDomain.Dek {
	return cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kek.ID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek"),
		Nonce:        []byte("nonce"),
	}
}
