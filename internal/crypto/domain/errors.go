// Package domain defines core cryptographic domain models for envelope encryption.
// Implements Master Key → KEK → DEK → Data hierarchy with AESGCM and ChaCha20 support.
package domain

import (
	"github.com/allisson/secrets/internal/errors"
)

// Cryptographic operation errors.
var (
	// ErrUnsupportedAlgorithm indicates the requested encryption algorithm is not supported.
	ErrUnsupportedAlgorithm = errors.Wrap(errors.ErrInvalidInput, "unsupported algorithm")

	// ErrInvalidKeySize indicates the cryptographic key size is invalid (must be 32 bytes).
	ErrInvalidKeySize = errors.Wrap(errors.ErrInvalidInput, "invalid key size")

	// ErrDecryptionFailed indicates decryption failed due to wrong key or corrupted data.
	ErrDecryptionFailed = errors.Wrap(errors.ErrInvalidInput, "decryption failed")

	// ErrMasterKeysNotSet indicates the MASTER_KEYS environment variable is not configured.
	ErrMasterKeysNotSet = errors.Wrap(errors.ErrInvalidInput, "MASTER_KEYS not set")

	// ErrActiveMasterKeyIDNotSet indicates the ACTIVE_MASTER_KEY_ID environment variable is not configured.
	ErrActiveMasterKeyIDNotSet = errors.Wrap(errors.ErrInvalidInput, "ACTIVE_MASTER_KEY_ID not set")

	// ErrInvalidMasterKeysFormat indicates the MASTER_KEYS format is invalid.
	ErrInvalidMasterKeysFormat = errors.Wrap(errors.ErrInvalidInput, "invalid MASTER_KEYS format")

	// ErrInvalidMasterKeyBase64 indicates a master key is not valid base64.
	ErrInvalidMasterKeyBase64 = errors.Wrap(errors.ErrInvalidInput, "invalid master key base64")

	// ErrActiveMasterKeyNotFound indicates the active master key ID was not found in the keychain.
	ErrActiveMasterKeyNotFound = errors.Wrap(errors.ErrInvalidInput, "active master key not found")

	// ErrMasterKeyNotFound indicates a master key with the specified ID was not found.
	ErrMasterKeyNotFound = errors.Wrap(errors.ErrNotFound, "master key not found")

	// ErrDekNotFound indicates a DEK with the specified ID was not found.
	ErrDekNotFound = errors.Wrap(errors.ErrNotFound, "dek not found")

	// ErrKekNotFound indicates a KEK with the specified ID was not found.
	ErrKekNotFound = errors.Wrap(errors.ErrNotFound, "kek not found")

	// ErrKMSProviderNotSet indicates the KMS_PROVIDER environment variable is not configured (required).
	ErrKMSProviderNotSet = errors.Wrap(
		errors.ErrInvalidInput,
		"KMS_PROVIDER is required but not configured (use 'localsecrets' for local development)",
	)

	// ErrKMSKeyURINotSet indicates the KMS_KEY_URI environment variable is not configured (required).
	ErrKMSKeyURINotSet = errors.Wrap(
		errors.ErrInvalidInput,
		"KMS_KEY_URI is required but not configured",
	)

	// ErrKMSDecryptionFailed indicates KMS decryption of master keys failed.
	ErrKMSDecryptionFailed = errors.Wrap(errors.ErrInvalidInput, "KMS decryption failed")

	// ErrKMSOpenKeeperFailed indicates opening KMS keeper failed.
	ErrKMSOpenKeeperFailed = errors.Wrap(errors.ErrInvalidInput, "failed to open KMS keeper")
)
