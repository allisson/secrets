package keyring

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// Fake is an in-memory Keyring suitable for feature unit tests.
//
// It does not perform real cryptography: ciphertext is a deterministic
// transformation of the plaintext keyed by the DekID. The point is to give
// features a real second adapter so that Keyring becomes a real seam, and to
// let feature tests assert behaviour (a value can be encrypted, persisted,
// and decrypted back) without touching the database or KMS.
//
// Concurrency-safe.
type Fake struct {
	mu      sync.Mutex
	deks    map[uuid.UUID]struct{}
	nextDek uint64

	// FailEncrypt, FailDecrypt, FailAllocate, FailRewrap, FailSign, when
	// non-nil, make the matching method return the stored error. Useful for
	// failure-path tests in callers.
	FailEncrypt  error
	FailDecrypt  error
	FailAllocate error
	FailRewrap   error
	FailSign     error
}

// NewFake constructs an empty Fake.
func NewFake() *Fake {
	return &Fake{deks: map[uuid.UUID]struct{}{}}
}

// Encrypt returns an Envelope whose Ciphertext is a reversible XOR of the
// plaintext with a DekID-derived stream. The DekID is allocated and tracked.
func (f *Fake) Encrypt(_ context.Context, plaintext []byte) (Envelope, error) {
	if f.FailEncrypt != nil {
		return Envelope{}, f.FailEncrypt
	}

	dekID := f.allocate()
	return Envelope{
		DekID:      dekID,
		Ciphertext: xorStream(plaintext, dekID),
		Nonce:      nonceFor(dekID),
	}, nil
}

// Decrypt reverses Encrypt. Returns an error if the DekID was never allocated.
func (f *Fake) Decrypt(_ context.Context, env Envelope) ([]byte, error) {
	if f.FailDecrypt != nil {
		return nil, f.FailDecrypt
	}

	if !f.knows(env.DekID) {
		return nil, errors.New("keyring.Fake: unknown DekID")
	}
	return xorStream(env.Ciphertext, env.DekID), nil
}

// AllocateDek returns a fresh handle. Algorithm is ignored.
func (f *Fake) AllocateDek(_ context.Context, _ Algorithm) (DekHandle, error) {
	if f.FailAllocate != nil {
		return DekHandle{}, f.FailAllocate
	}
	return DekHandle{DekID: f.allocate()}, nil
}

// EncryptWith XORs plaintext with a stream derived from the handle's DekID
// and the optional aad.
func (f *Fake) EncryptWith(
	_ context.Context,
	handle DekHandle,
	plaintext, aad []byte,
) (ciphertext, nonce []byte, err error) {
	if f.FailEncrypt != nil {
		return nil, nil, f.FailEncrypt
	}
	if !f.knows(handle.DekID) {
		return nil, nil, errors.New("keyring.Fake: unknown DekID")
	}
	return xorStreamAAD(plaintext, handle.DekID, aad), nonceFor(handle.DekID), nil
}

// DecryptWith reverses EncryptWith.
func (f *Fake) DecryptWith(
	_ context.Context,
	handle DekHandle,
	ciphertext, _ []byte,
	aad []byte,
) ([]byte, error) {
	if f.FailDecrypt != nil {
		return nil, f.FailDecrypt
	}
	if !f.knows(handle.DekID) {
		return nil, errors.New("keyring.Fake: unknown DekID")
	}
	return xorStreamAAD(ciphertext, handle.DekID, aad), nil
}

// Rewrap is a no-op for the Fake (there is no KEK chain) but honors
// FailRewrap and validates the DekID is known.
func (f *Fake) Rewrap(_ context.Context, dekID uuid.UUID) error {
	if f.FailRewrap != nil {
		return f.FailRewrap
	}
	if !f.knows(dekID) {
		return errors.New("keyring.Fake: unknown DekID")
	}
	return nil
}

// RewrapAll returns 0 by default for the Fake — there is no notion of a
// stale KEK in-memory. Tests that exercise the rotation worker should
// stub this behaviour by counting tracked DEKs.
func (f *Fake) RewrapAll(_ context.Context, _ int) (int, error) {
	if f.FailRewrap != nil {
		return 0, f.FailRewrap
	}
	return 0, nil
}

// ActiveKekID returns the zero UUID for the Fake; the value is only used
// by the rotation worker to verify the operator-provided KEK ID matches.
func (f *Fake) ActiveKekID() uuid.UUID {
	return uuid.Nil
}

// SignWithKey computes HMAC-SHA256 of data using a fixed zero key and returns
// uuid.Nil as the KEK ID. Intended for tests; does not use real key material.
func (f *Fake) SignWithKey(data []byte) ([]byte, uuid.UUID, error) {
	if f.FailSign != nil {
		return nil, uuid.Nil, f.FailSign
	}
	mac := hmac.New(sha256.New, make([]byte, 32))
	mac.Write(data)
	return mac.Sum(nil), uuid.Nil, nil
}

// VerifyWithKey verifies sig against data. The kekID is ignored by the Fake.
func (f *Fake) VerifyWithKey(_ uuid.UUID, data, sig []byte) error {
	if f.FailSign != nil {
		return f.FailSign
	}
	expected, _, _ := f.SignWithKey(data)
	if !hmac.Equal(sig, expected) {
		return ErrSignatureInvalid
	}
	return nil
}

func (f *Fake) allocate() uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextDek++
	var id uuid.UUID
	// Spread the counter across both halves so XOR-stream is non-zero
	// everywhere (real DekIDs are UUIDv7, with high-entropy across all bytes).
	binary.BigEndian.PutUint64(id[0:], f.nextDek^0x9e3779b97f4a7c15)
	binary.BigEndian.PutUint64(id[8:], f.nextDek)
	f.deks[id] = struct{}{}
	return id
}

func (f *Fake) knows(id uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.deks[id]
	return ok
}

func xorStream(data []byte, dekID uuid.UUID) []byte {
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ dekID[i%len(dekID)]
	}
	return out
}

func xorStreamAAD(data []byte, dekID uuid.UUID, aad []byte) []byte {
	out := xorStream(data, dekID)
	for i := range out {
		if len(aad) == 0 {
			break
		}
		out[i] ^= aad[i%len(aad)]
	}
	return out
}

func nonceFor(dekID uuid.UUID) []byte {
	n := make([]byte, 12)
	copy(n, dekID[:])
	return n
}
