// Package testing provides shared test utilities for tokenization module tests.
package testing

import (
	"github.com/allisson/secrets/internal/keyring"
)

// CreateMasterKey creates a test master key with a random 32-byte key.
func CreateMasterKey() *keyring.MasterKey {
	return &keyring.MasterKey{
		ID:  "test-master-key",
		Key: make([]byte, 32),
	}
}
