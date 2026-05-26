package keyring

import "github.com/google/uuid"

// NullSigner is a no-op KeySigner for tests that do not exercise signing behaviour.
// SignWithKey returns a 32-byte zero signature and uuid.Nil; VerifyWithKey always returns nil.
type NullSigner struct{}

func (NullSigner) SignWithKey(_ []byte) ([]byte, uuid.UUID, error) {
	return make([]byte, 32), uuid.Nil, nil
}

func (NullSigner) VerifyWithKey(_ uuid.UUID, _, _ []byte) error { return nil }
