package domain

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
)

// MasterKey represents a cryptographic master key used to encrypt Key Encryption Keys (KEKs).
//
// Master keys are the root of the envelope encryption hierarchy and should be stored
// securely in a Key Management Service (KMS), Hardware Security Module (HSM), or
// loaded from environment variables in development/test environments.
//
// Security considerations:
//   - Master keys should be 32 bytes (256 bits) for AES-256 compatibility
//   - Keys should be generated using cryptographically secure random generators
//   - Keys should be rotated periodically according to security policies
//   - Multiple master keys can be maintained for key rotation scenarios
//
// Fields:
//   - ID: Unique identifier for the master key (e.g., "prod-master-key-2025")
//   - Key: The raw 32-byte master key material
type MasterKey struct {
	ID  string
	Key []byte
}

// MasterKeyChain manages a collection of master keys with one designated as active.
//
// The keychain allows for key rotation by maintaining multiple master keys
// simultaneously. Old keys remain available to decrypt existing KEKs while new
// KEKs are encrypted with the active key.
//
// Key rotation workflow:
//  1. Add a new master key to the keychain
//  2. Set the new key as active
//  3. New KEKs will be encrypted with the new active key
//  4. Old KEKs can still be decrypted using their original master keys
//  5. Gradually re-encrypt old KEKs with the new master key
//
// Thread safety: The keychain uses sync.Map internally for concurrent access.
//
// Fields:
//   - activeID: ID of the master key to use for encrypting new KEKs
//   - keys: Thread-safe map of master key ID to MasterKey instances
type MasterKeyChain struct {
	activeID string
	keys     sync.Map
}

// ActiveMasterKeyID returns the ID of the currently active master key.
//
// The active master key is used to encrypt new Key Encryption Keys (KEKs).
// This ID corresponds to the ACTIVE_MASTER_KEY_ID environment variable.
func (m *MasterKeyChain) ActiveMasterKeyID() string {
	return m.activeID
}

// Get retrieves a master key from the keychain by its ID.
//
// This method is used to obtain the appropriate master key for decrypting
// KEKs that were encrypted with different master keys (useful during key rotation).
//
// Parameters:
//   - id: The unique identifier of the master key to retrieve
//
// Returns:
//   - The MasterKey if found
//   - A boolean indicating whether the key was found in the keychain
func (m *MasterKeyChain) Get(id string) (*MasterKey, bool) {
	if masterKey, ok := m.keys.Load(id); ok {
		return masterKey.(*MasterKey), ok
	}

	return nil, false
}

// Close securely clears all master keys from memory and resets the keychain.
//
// This method should be called when the keychain is no longer needed (e.g.,
// during application shutdown or when reloading configuration). It ensures
// sensitive key material is removed from memory.
//
// Note: Individual key bytes are zeroed when keys are loaded, but the keychain
// structure itself is cleared here for complete cleanup.
func (m *MasterKeyChain) Close() {
	m.activeID = ""
	m.keys.Clear()
}

// LoadMasterKeyChainFromEnv loads master keys from environment variables.
//
// This function reads master key configuration from two environment variables:
//   - MASTER_KEYS: Comma-separated list of key entries in format "id:base64key"
//   - ACTIVE_MASTER_KEY_ID: ID of the master key to use for encrypting new KEKs
//
// Format example:
//
//	MASTER_KEYS="key1:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3OA==,key2:MTIzNDU2Nzg5MGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eA=="
//	ACTIVE_MASTER_KEY_ID="key2"
//
// Each master key must be:
//   - Exactly 32 bytes when base64-decoded
//   - Uniquely identified by its ID
//   - Base64-encoded using standard encoding
//
// Security notes:
//   - Temporary decoded key bytes are zeroed after being stored in the keychain
//   - On error, the keychain is closed to prevent partial initialization
//   - In production, consider using a proper KMS instead of environment variables
//
// Returns:
//   - A fully initialized MasterKeyChain ready for use
//   - ErrMasterKeysNotSet if MASTER_KEYS is not configured
//   - ErrActiveMasterKeyIDNotSet if ACTIVE_MASTER_KEY_ID is not configured
//   - ErrInvalidMasterKeysFormat if the format is invalid
//   - ErrInvalidMasterKeyBase64 if base64 decoding fails
//   - ErrInvalidKeySize if a key is not exactly 32 bytes
//   - ErrActiveMasterKeyNotFound if the active key ID is not in the keychain
func LoadMasterKeyChainFromEnv() (*MasterKeyChain, error) {
	raw := os.Getenv("MASTER_KEYS")
	if raw == "" {
		return nil, ErrMasterKeysNotSet
	}

	active := os.Getenv("ACTIVE_MASTER_KEY_ID")
	if active == "" {
		return nil, ErrActiveMasterKeyIDNotSet
	}

	mkc := &MasterKeyChain{activeID: active}

	parts := strings.SplitSeq(raw, ",")
	for part := range parts {
		p := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(p) != 2 {
			mkc.Close()
			return nil, fmt.Errorf("%w: %q", ErrInvalidMasterKeysFormat, part)
		}
		id := p[0]
		key, err := base64.StdEncoding.DecodeString(p[1])
		if err != nil {
			mkc.Close()
			return nil, fmt.Errorf("%w for %s: %v", ErrInvalidMasterKeyBase64, id, err)
		}
		if len(key) != 32 {
			zero(key)
			mkc.Close()
			return nil, fmt.Errorf(
				"%w: master key %s must be 32 bytes, got %d",
				ErrInvalidKeySize,
				id,
				len(key),
			)
		}
		mkc.keys.Store(id, &MasterKey{ID: id, Key: key})
		zero(key)
	}

	if _, ok := mkc.Get(active); !ok {
		mkc.Close()
		return nil, fmt.Errorf("%w: ACTIVE_MASTER_KEY_ID=%s", ErrActiveMasterKeyNotFound, active)
	}

	return mkc, nil
}
