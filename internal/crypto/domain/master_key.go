package domain

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
)

// MasterKey represents a cryptographic master key used to encrypt KEKs.
// Should be 32 bytes (256 bits) and stored securely in KMS or environment variables.
type MasterKey struct {
	ID  string // Unique identifier for the master key
	Key []byte // The raw 32-byte master key material
}

// MasterKeyChain manages a collection of master keys with one designated as active.
// Supports key rotation by maintaining multiple keys simultaneously.
type MasterKeyChain struct {
	activeID string   // ID of the master key to use for encrypting new KEKs
	keys     sync.Map // Thread-safe map of master key ID to MasterKey instances
}

// ActiveMasterKeyID returns the ID of the currently active master key.
func (m *MasterKeyChain) ActiveMasterKeyID() string {
	return m.activeID
}

// Get retrieves a master key from the keychain by its ID.
func (m *MasterKeyChain) Get(id string) (*MasterKey, bool) {
	if masterKey, ok := m.keys.Load(id); ok {
		return masterKey.(*MasterKey), ok
	}

	return nil, false
}

// Close securely clears all master keys from memory and resets the keychain.
func (m *MasterKeyChain) Close() {
	m.activeID = ""
	m.keys.Clear()
}

// LoadMasterKeyChainFromEnv loads master keys from MASTER_KEYS and ACTIVE_MASTER_KEY_ID environment variables.
// Keys must be in format "id:base64key" (comma-separated) and exactly 32 bytes when decoded.
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
			Zero(key)
			mkc.Close()
			return nil, fmt.Errorf(
				"%w: master key %s must be 32 bytes, got %d",
				ErrInvalidKeySize,
				id,
				len(key),
			)
		}
		mkc.keys.Store(id, &MasterKey{ID: id, Key: key})
		Zero(key)
	}

	if _, ok := mkc.Get(active); !ok {
		mkc.Close()
		return nil, fmt.Errorf("%w: ACTIVE_MASTER_KEY_ID=%s", ErrActiveMasterKeyNotFound, active)
	}

	return mkc, nil
}
