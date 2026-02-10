// Package domain defines the core cryptographic domain models for envelope encryption.
//
// It implements a three-tier key hierarchy: Master Key → KEK → DEK → Data.
// KEKs encrypt Data Encryption Keys, enabling efficient key rotation without
// re-encrypting all data. Supports AESGCM and ChaCha20 algorithms with 256-bit keys.
package domain

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Kek represents a Key Encryption Key used to encrypt Data Encryption Keys.
// It is itself encrypted with a master key and stored in the database.
type Kek struct {
	ID           uuid.UUID // Unique identifier (UUIDv7)
	MasterKeyID  string    // ID of the master key used to encrypt this KEK
	Algorithm    Algorithm // Encryption algorithm (AESGCM or ChaCha20)
	EncryptedKey []byte    // The KEK encrypted with the master key
	Key          []byte    // Plaintext KEK (populated after decryption, never persisted)
	Nonce        []byte    // Unique nonce for encrypting the KEK
	Version      uint      // Version number for rotation tracking
	CreatedAt    time.Time
}

// KekChain manages a collection of Key Encryption Keys with thread-safe access.
// The active KEK (highest version) is used for encrypting new DEKs.
type KekChain struct {
	activeID uuid.UUID // UUID of the currently active KEK
	keys     sync.Map  // Thread-safe map of KEK ID to KEK instances
}

// ActiveKekID returns the UUID of the currently active Key Encryption Key.
func (k *KekChain) ActiveKekID() uuid.UUID {
	return k.activeID
}

// Get retrieves a Key Encryption Key from the chain by its UUID.
func (k *KekChain) Get(id uuid.UUID) (*Kek, bool) {
	if kek, ok := k.keys.Load(id); ok {
		return kek.(*Kek), ok
	}

	return nil, false
}

// Close securely clears all KEKs from the chain and resets the active ID.
func (k *KekChain) Close() {
	// Zero all KEK keys before clearing
	k.keys.Range(func(key, value interface{}) bool {
		if kek, ok := value.(*Kek); ok {
			Zero(kek.Key)
		}
		return true
	})
	k.activeID = uuid.Nil
	k.keys.Clear()
}

// NewKekChain creates a new KekChain with the first KEK as active.
// KEKs must be ordered by version descending (newest first).
func NewKekChain(keks []*Kek) *KekChain {
	kc := &KekChain{
		activeID: keks[0].ID,
	}

	for _, kek := range keks {
		kc.keys.Store(kek.ID, kek)
	}

	return kc
}
