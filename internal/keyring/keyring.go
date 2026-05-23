// Package keyring provides envelope encryption as a single deep module.
//
// Callers exchange plaintext for an Envelope (DekID + Ciphertext + Nonce) and
// back. The KEK chain, DEK lifecycle, AEAD cipher selection, and KMS-rooted
// master key chain all live behind this interface.
//
// Two encryption shapes are supported:
//
//   - Fresh-DEK envelope (Encrypt/Decrypt) — a new DEK is created for each
//     call. Used by the secrets and tokenization features.
//   - Persistent DEK (AllocateDek/EncryptWith/DecryptWith) — one DEK is
//     allocated once and reused across many encrypt/decrypt calls. Used by
//     the transit feature where a named key wraps user payloads repeatedly.
//
// All methods honor an ambient transaction propagated via context; persistence
// joins the caller's tx when one is present (see ADR-0005).
package keyring

import (
	"context"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// Algorithm is the AEAD algorithm used to wrap a DEK and to encrypt under it.
// Re-exported from internal/crypto/domain so callers do not need that import.
type Algorithm = cryptoDomain.Algorithm

const (
	// AESGCM is AES-256-GCM (optimal with AES-NI).
	AESGCM = cryptoDomain.AESGCM

	// ChaCha20 is ChaCha20-Poly1305 (optimal without AES-NI).
	ChaCha20 = cryptoDomain.ChaCha20
)

var (
	// ErrDecryptionFailed is returned when Decrypt or DecryptWith cannot recover
	// plaintext due to missing key material or cipher failure.
	// Re-exported so callers do not need to import crypto/domain.
	ErrDecryptionFailed = cryptoDomain.ErrDecryptionFailed

	// Zero overwrites b with zeros, clearing sensitive material from memory.
	// Re-exported from crypto/domain so callers do not need that import.
	Zero = cryptoDomain.Zero
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

// Keyring is the single seam features use to encrypt and decrypt data.
//
// Implementations must be safe for concurrent use.
type Keyring interface {
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
