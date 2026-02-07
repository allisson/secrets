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

	// ErrMasterKeysNotSet indicates the MASTER_KEYS environment variable is not configured.
	//
	// This error is returned when attempting to load master keys from environment
	// but the MASTER_KEYS variable is empty or not set.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrMasterKeysNotSet = errors.Wrap(errors.ErrInvalidInput, "MASTER_KEYS not set")

	// ErrActiveMasterKeyIDNotSet indicates the ACTIVE_MASTER_KEY_ID environment variable is not configured.
	//
	// This error is returned when attempting to load master keys from environment
	// but the ACTIVE_MASTER_KEY_ID variable is empty or not set.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrActiveMasterKeyIDNotSet = errors.Wrap(errors.ErrInvalidInput, "ACTIVE_MASTER_KEY_ID not set")

	// ErrInvalidMasterKeysFormat indicates the MASTER_KEYS format is invalid.
	//
	// The expected format is: "id1:base64key1,id2:base64key2"
	// Each entry must have an ID and base64-encoded key separated by a colon.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrInvalidMasterKeysFormat = errors.Wrap(errors.ErrInvalidInput, "invalid MASTER_KEYS format")

	// ErrInvalidMasterKeyBase64 indicates a master key is not valid base64.
	//
	// All master keys must be base64-encoded strings that decode to exactly 32 bytes.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrInvalidMasterKeyBase64 = errors.Wrap(errors.ErrInvalidInput, "invalid master key base64")

	// ErrActiveMasterKeyNotFound indicates the active master key ID was not found in the keychain.
	//
	// This error is returned when ACTIVE_MASTER_KEY_ID references a key ID
	// that doesn't exist in the MASTER_KEYS configuration.
	//
	// HTTP Status: 422 Unprocessable Entity
	ErrActiveMasterKeyNotFound = errors.Wrap(errors.ErrInvalidInput, "active master key not found")

	// ErrMasterKeyNotFound indicates a master key with the specified ID was not found.
	//
	// This error is returned when attempting to retrieve a master key by ID
	// but the key doesn't exist in the master key chain.
	//
	// HTTP Status: 404 Not Found
	ErrMasterKeyNotFound = errors.Wrap(errors.ErrNotFound, "master key not found")

	// ErrDekNotFound indicates a DEK with the specified ID was not found.
	//
	// This error is returned when attempting to retrieve a Data Encryption Key by ID
	// but the key doesn't exist in the database.
	//
	// HTTP Status: 404 Not Found
	ErrDekNotFound = errors.Wrap(errors.ErrNotFound, "dek not found")

	// ErrKekNotFound indicates a KEK with the specified ID was not found.
	//
	// This error is returned when attempting to retrieve a Key Encryption Key by ID
	// but the key doesn't exist in the KEK chain.
	//
	// HTTP Status: 404 Not Found
	ErrKekNotFound = errors.Wrap(errors.ErrNotFound, "kek not found")
)
