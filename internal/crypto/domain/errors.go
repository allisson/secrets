package domain

import (
	"github.com/allisson/secrets/internal/errors"
)

// Cryptographic operation error definitions.
//
// These domain-specific errors wrap standard errors from internal/errors
// to provide context for cryptographic failures. All errors are mapped to
// appropriate HTTP status codes by the error handling layer.
var (
	// ErrUnsupportedAlgorithm indicates the requested encryption algorithm is not supported.
	//
	// Supported algorithms: AESGCM (AES-256-GCM), ChaCha20 (ChaCha20-Poly1305)
	// This error is returned when an invalid or unknown algorithm is specified
	// during KEK or DEK creation.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrUnsupportedAlgorithm = errors.Wrap(errors.ErrInvalidInput, "unsupported algorithm")

	// ErrInvalidKeySize indicates the cryptographic key size is invalid.
	//
	// All keys (master keys, KEKs, and DEKs) must be exactly 32 bytes (256 bits)
	// for both AES-256-GCM and ChaCha20-Poly1305 algorithms. This error is returned
	// when a key of incorrect length is provided.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrInvalidKeySize = errors.Wrap(errors.ErrInvalidInput, "invalid key size")

	// ErrDecryptionFailed indicates a decryption operation failed.
	//
	// This error can occur due to:
	//   - Wrong decryption key used
	//   - Ciphertext has been tampered with (authentication failure)
	//   - Invalid nonce provided
	//   - Corrupted encrypted data
	//
	// For security reasons, the specific cause is not disclosed to prevent
	// information leakage that could aid attackers.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrDecryptionFailed = errors.Wrap(errors.ErrInvalidInput, "decryption failed")
)
