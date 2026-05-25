// Package keyring provides envelope encryption and audit-log signing as a
// single deep module.
//
// Callers exchange plaintext for an Envelope (DekID + Ciphertext + Nonce) and
// back. The KEK chain, DEK lifecycle, AEAD cipher selection, KMS-rooted
// master key chain, and HMAC-SHA256 signing all live behind this interface.
//
// Two encryption shapes are supported:
//
//   - Fresh-DEK envelope (Encrypt/Decrypt) — a new DEK is created for each
//     call. Used by the secrets and tokenization features.
//   - Persistent DEK (AllocateDek/EncryptWith/DecryptWith) — one DEK is
//     allocated once and reused across many encrypt/decrypt calls. Used by
//     the transit feature where a named key wraps user payloads repeatedly.
//
// Signing (SignWithKey/VerifyWithKey) uses HKDF-SHA256 to derive a purpose-
// specific key from the active KEK, keeping raw key material inside the module.
//
// All methods honor an ambient transaction propagated via context; persistence
// joins the caller's tx when one is present (see ADR-0005).
package keyring

import (
	"context"

	"github.com/google/uuid"
)

// Algorithm is the AEAD algorithm used to wrap a DEK and to encrypt under it.
type Algorithm string

const (
	// AESGCM selects AES-256-GCM. Prefer this on hardware with AES-NI acceleration.
	AESGCM Algorithm = "aes-gcm"

	// ChaCha20 selects ChaCha20-Poly1305. Prefer this on hardware without AES-NI.
	ChaCha20 Algorithm = "chacha20-poly1305"
)

// Envelope is the result of Keyring.Encrypt and the input to Keyring.Decrypt.
// Callers persist the three fields exactly as they would today; nothing about
// the KEK or DEK material is exposed.
type Envelope struct {
	DekID      uuid.UUID
	Ciphertext []byte
	Nonce      []byte
}

// DekHandle is an opaque reference to a persistent DEK allocated via
// AllocateDek. Callers store only the DekID and reload the handle on demand.
type DekHandle struct {
	DekID uuid.UUID
}

// KeySigner signs and verifies arbitrary byte payloads using KEK-derived HMAC-SHA256
// keys. Key material never leaves the keyring.
type KeySigner interface {
	// SignWithKey signs data using a key derived from the active KEK via HKDF-SHA256.
	// Returns the 32-byte HMAC-SHA256 signature and the ID of the KEK used.
	SignWithKey(data []byte) (sig []byte, kekID uuid.UUID, err error)

	// VerifyWithKey verifies data against sig using the KEK identified by kekID.
	// Returns ErrSignatureInvalid if the signature does not match.
	// Returns ErrKekNotFound if the KEK is not present in the chain.
	VerifyWithKey(kekID uuid.UUID, data, sig []byte) error
}

// Keyring is the single seam features use for encryption and signing.
// It embeds KeySigner so any Keyring implementation also satisfies signing
// without a separate interface or runtime type assertion.
//
// Implementations must be safe for concurrent use.
type Keyring interface {
	KeySigner

	// Encrypt creates a fresh DEK, persists it, and uses it to encrypt
	// plaintext exactly once. The returned Envelope contains everything a
	// future Decrypt call needs.
	Encrypt(ctx context.Context, plaintext []byte) (Envelope, error)

	// Decrypt reverses Encrypt for an Envelope produced by this Keyring.
	Decrypt(ctx context.Context, env Envelope) ([]byte, error)

	// AllocateDek creates and persists a new DEK and returns a handle to it.
	// The DEK is wrapped under the active KEK with the given algorithm.
	AllocateDek(ctx context.Context, alg Algorithm) (DekHandle, error)

	// EncryptWith encrypts plaintext under the DEK identified by handle and
	// authenticates the optional aad. Reuses the DEK across many calls.
	EncryptWith(
		ctx context.Context,
		handle DekHandle,
		plaintext, aad []byte,
	) (ciphertext, nonce []byte, err error)

	// DecryptWith reverses EncryptWith for ciphertext produced under the
	// same DekHandle with the same aad.
	DecryptWith(
		ctx context.Context,
		handle DekHandle,
		ciphertext, nonce, aad []byte,
	) ([]byte, error)

	// Rewrap re-encrypts an existing DEK under the active KEK without
	// changing the underlying key material. Used by KEK rotation.
	Rewrap(ctx context.Context, dekID uuid.UUID) error

	// RewrapAll re-encrypts, in batches, every DEK not already under the
	// active KEK. Returns the total number of DEKs rewrapped across all
	// batches. Used by the KEK rotation worker.
	RewrapAll(ctx context.Context, batchSize int) (int, error)

	// ActiveKekID returns the ID of the KEK currently used for new
	// encryption. Exposed so the rotation worker can confirm it is
	// targeting the same KEK the application is using.
	ActiveKekID() uuid.UUID
}
