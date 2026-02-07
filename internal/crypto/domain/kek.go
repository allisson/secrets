// Package domain defines the core cryptographic domain models and types.
//
// This package implements the domain layer for envelope encryption, a multi-tier
// key management scheme that provides efficient key rotation and enhanced security.
// It provides the fundamental data structures and business rules for managing
// encryption keys in a hierarchical key management system.
//
// # Envelope Encryption Architecture
//
// The package implements a three-tier key hierarchy:
//
//	Master Key (KMS/Environment)
//	    ↓ encrypts
//	Key Encryption Key (KEK - stored in database)
//	    ↓ encrypts
//	Data Encryption Key (DEK - stored with data)
//	    ↓ encrypts
//	Application Data
//
// # Key Components
//
// MasterKey: Root keys stored securely in KMS or environment variables.
// Used to encrypt and decrypt KEKs. Supports key rotation through MasterKeyChain.
//
// Kek (Key Encryption Key): Intermediate keys stored in the database.
// Used to encrypt and decrypt DEKs. Supports versioning and rotation.
//
// Dek (Data Encryption Key): Per-record encryption keys stored alongside data.
// Used to encrypt actual application data. One DEK per encrypted item.
//
// # Supported Algorithms
//
// The package supports two AEAD (Authenticated Encryption with Associated Data) algorithms:
//
//   - AESGCM: AES-256-GCM, optimal on systems with AES-NI hardware acceleration
//   - ChaCha20: ChaCha20-Poly1305, optimal on mobile devices and systems without AES-NI
//
// Both algorithms provide 256-bit security and authenticated encryption.
//
// # Error Handling
//
// All domain errors wrap standard errors from internal/errors for consistent
// error handling and HTTP status code mapping. Errors follow the pattern:
//
//	ErrSpecificError = errors.Wrap(errors.ErrInvalidInput, "specific context")
//
// # Security Features
//
//   - 256-bit keys for maximum security
//   - AEAD encryption for confidentiality and authenticity
//   - Key rotation support without re-encrypting all data
//   - Secure memory zeroing for sensitive key material
//   - Per-record encryption with unique DEKs
package domain

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Kek represents a Key Encryption Key used in envelope encryption.
//
// A KEK is a cryptographic key used to encrypt Data Encryption Keys (DEKs).
// The KEK itself is encrypted with a master key and stored securely in a database.
// This approach allows for key rotation without re-encrypting all data.
//
// In envelope encryption hierarchy:
//   - Master Key (stored in KMS) → encrypts → KEK (stored in DB) → encrypts → DEK
//
// Key rotation workflow:
//  1. Create a new KEK with an incremented version number
//  2. New DEKs will be encrypted with the new KEK (highest version)
//  3. Old DEKs can still be decrypted with the old KEK until they are re-encrypted
//
// Fields:
//   - ID: Unique identifier for the KEK (UUIDv7 for time-based ordering)
//   - MasterKeyID: ID of the master key used to encrypt this KEK (for key rotation tracking)
//   - Algorithm: Encryption algorithm used (AES-GCM or ChaCha20-Poly1305)
//   - EncryptedKey: The KEK encrypted with the master key (safe to store in DB)
//   - Key: The plaintext KEK (populated after decryption, should never be persisted)
//   - Nonce: Unique nonce used for encrypting the KEK with the master key
//   - Version: Version number for tracking KEK rotations (increments with each rotation)
//   - CreatedAt: Timestamp when the KEK was created
type Kek struct {
	ID           uuid.UUID
	MasterKeyID  string
	Algorithm    Algorithm
	EncryptedKey []byte
	Key          []byte
	Nonce        []byte
	Version      uint
	CreatedAt    time.Time
}

// KekChain manages a collection of Key Encryption Keys with thread-safe access.
//
// The KekChain provides a concurrent-safe way to store and access multiple KEK versions,
// with one designated as the active KEK (highest version) for encrypting new Data Encryption
// Keys (DEKs). This supports key rotation workflows where old KEKs remain available for
// decrypting existing DEKs while new DEKs are encrypted with the latest active KEK.
//
// Key rotation workflow:
//  1. New KEK is created with an incremented version number
//  2. Old KEK remains in the chain for decrypting existing DEKs
//  3. New DEKs are encrypted with the active KEK (highest version)
//  4. Over time, old DEKs can be re-encrypted with the new KEK
//  5. Once all DEKs use the new KEK, old KEKs can be removed
//
// Thread safety: The KekChain uses sync.Map internally for concurrent access,
// making it safe to use from multiple goroutines simultaneously.
//
// Memory management: Call Close() when the chain is no longer needed to clear
// sensitive key material from memory.
//
// Fields:
//   - activeID: UUID of the currently active KEK (highest version, used for encrypting new DEKs)
//   - keys: Thread-safe map of KEK ID to KEK instances
type KekChain struct {
	activeID uuid.UUID
	keys     sync.Map
}

// ActiveKekID returns the UUID of the currently active Key Encryption Key.
//
// The active KEK is the one with the highest version number and is used to encrypt
// new Data Encryption Keys (DEKs). During key rotation, this ID changes to point to
// the newest KEK version while old KEKs remain accessible in the chain for decrypting
// existing DEKs.
//
// Returns:
//   - The UUID of the active KEK
//
// Example:
//
//	activeID := kekChain.ActiveKekID()
//	activeKek, found := kekChain.Get(activeID)
//	if !found {
//	    return errors.New("active KEK not found in chain")
//	}
//	// Use activeKek to encrypt new DEKs
func (k *KekChain) ActiveKekID() uuid.UUID {
	return k.activeID
}

// Get retrieves a Key Encryption Key from the chain by its UUID.
//
// This method provides thread-safe access to KEKs stored in the chain. It's
// used to obtain KEKs for decrypting Data Encryption Keys (DEKs) that reference
// a specific KEK version. The active KEK can be retrieved by first calling
// ActiveKekID() to get its UUID.
//
// Parameters:
//   - id: The UUID of the KEK to retrieve
//
// Returns:
//   - The KEK if found in the chain
//   - A boolean indicating whether the KEK was found (true) or not (false)
//
// Example:
//
//	// Get a specific KEK for decrypting a DEK
//	kek, found := kekChain.Get(dek.KekID)
//	if !found {
//	    return errors.New("KEK not found for decrypting DEK")
//	}
//	// Use kek to decrypt the DEK
func (k *KekChain) Get(id uuid.UUID) (*Kek, bool) {
	if kek, ok := k.keys.Load(id); ok {
		return kek.(*Kek), ok
	}

	return nil, false
}

// Close securely clears all KEKs from the chain and resets the active ID.
//
// This method should be called when the KekChain is no longer needed (e.g.,
// during application shutdown, reloading configuration, or key rotation completion).
// It ensures sensitive key material is removed from memory by clearing the internal
// storage and resetting the active KEK reference.
//
// After calling Close(), the KekChain should not be used anymore. Create a new
// KekChain if KEKs are needed again.
//
// Note: Individual KEK key bytes should be zeroed separately if needed before
// calling Close(). This method only clears the chain's internal storage structure.
//
// Example:
//
//	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
//	if err != nil {
//	    return err
//	}
//	defer kekChain.Close() // Ensure cleanup on function exit
//
//	// Use kekChain...
func (k *KekChain) Close() {
	k.activeID = uuid.Nil
	k.keys.Clear()
}

// NewKekChain creates a new KekChain from a slice of KEKs.
//
// This constructor initializes a thread-safe KEK chain with the provided KEKs.
// The first KEK in the slice (index 0) is designated as the active KEK (highest
// version), which will be used for encrypting new Data Encryption Keys (DEKs).
//
// The KEKs slice must be ordered by version in descending order (newest first)
// to ensure the KEK with the highest version becomes the active one. This is
// typically the order returned by the KEK repository's List() method.
//
// Parameters:
//   - keks: Slice of KEK pointers to store in the chain (must not be empty)
//
// Returns:
//   - A new KekChain with all KEKs loaded and the first KEK set as active
//
// Panics:
//   - If the keks slice is empty (accessing keks[0] would panic)
//
// Example:
//
//	// Keks are typically loaded from repository, ordered by version DESC
//	keks, err := kekRepo.List(ctx)
//	if err != nil {
//	    return nil, err
//	}
//	if len(keks) == 0 {
//	    return nil, errors.New("no KEKs found")
//	}
//
//	// Create chain with newest KEK as active
//	kekChain := NewKekChain(keks)
//	defer kekChain.Close()
func NewKekChain(keks []*Kek) *KekChain {
	kc := &KekChain{
		activeID: keks[0].ID,
	}

	for _, kek := range keks {
		kc.keys.Store(kek.ID, kek)
	}

	return kc
}
