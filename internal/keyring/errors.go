package keyring

import (
	"errors"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// Sentinel errors returned by the keyring package. Callers should use errors.Is to test them.
var (
	// ErrUnsupportedAlgorithm is returned when an unknown Algorithm value is used.
	ErrUnsupportedAlgorithm = apperrors.Wrap(apperrors.ErrInvalidInput, "unsupported algorithm")

	// ErrInvalidKeySize is returned when a key has an unexpected byte length.
	ErrInvalidKeySize = apperrors.Wrap(apperrors.ErrInvalidInput, "invalid key size")

	// ErrDecryptionFailed is returned when ciphertext cannot be authenticated or decrypted.
	ErrDecryptionFailed = apperrors.Wrap(apperrors.ErrInvalidInput, "decryption failed")

	// ErrMasterKeysNotSet is returned when the MASTER_KEYS environment variable is empty.
	ErrMasterKeysNotSet = apperrors.Wrap(apperrors.ErrInvalidInput, "MASTER_KEYS not set")

	// ErrActiveMasterKeyIDNotSet is returned when ACTIVE_MASTER_KEY_ID is not configured.
	ErrActiveMasterKeyIDNotSet = apperrors.Wrap(apperrors.ErrInvalidInput, "ACTIVE_MASTER_KEY_ID not set")

	// ErrInvalidMasterKeysFormat is returned when a MASTER_KEYS entry cannot be parsed.
	ErrInvalidMasterKeysFormat = apperrors.Wrap(apperrors.ErrInvalidInput, "invalid MASTER_KEYS format")

	// ErrInvalidMasterKeyBase64 is returned when a master key value is not valid base64.
	ErrInvalidMasterKeyBase64 = apperrors.Wrap(apperrors.ErrInvalidInput, "invalid master key base64")

	// ErrActiveMasterKeyNotFound is returned when ACTIVE_MASTER_KEY_ID is not present in the chain.
	ErrActiveMasterKeyNotFound = apperrors.Wrap(apperrors.ErrInvalidInput, "active master key not found")

	// ErrMasterKeyNotFound is returned when a KEK references a master key absent from the chain.
	ErrMasterKeyNotFound = apperrors.Wrap(apperrors.ErrNotFound, "master key not found")

	// ErrDekNotFound is returned when a DEK lookup by ID yields no row.
	ErrDekNotFound = apperrors.Wrap(apperrors.ErrNotFound, "dek not found")

	// ErrKekNotFound is returned when no KEK exists for the given ID, or no KEKs at all.
	ErrKekNotFound = apperrors.Wrap(apperrors.ErrNotFound, "kek not found")

	// ErrKMSProviderNotSet is returned when KMS_PROVIDER is empty.
	ErrKMSProviderNotSet = apperrors.Wrap(
		apperrors.ErrInvalidInput,
		"KMS_PROVIDER is required but not configured (use 'localsecrets' for local development)",
	)

	// ErrKMSKeyURINotSet is returned when KMS_KEY_URI is empty.
	ErrKMSKeyURINotSet = apperrors.Wrap(
		apperrors.ErrInvalidInput,
		"KMS_KEY_URI is required but not configured",
	)

	// ErrKMSDecryptionFailed is returned when the KMS cannot decrypt a master key.
	ErrKMSDecryptionFailed = apperrors.Wrap(apperrors.ErrInvalidInput, "KMS decryption failed")

	// ErrKMSOpenKeeperFailed is returned when the KMS keeper cannot be opened.
	ErrKMSOpenKeeperFailed = apperrors.Wrap(apperrors.ErrInvalidInput, "failed to open KMS keeper")

	// ErrSignatureInvalid is returned by VerifyWithKey when the HMAC does not match.
	ErrSignatureInvalid = errors.New("keyring: signature invalid")
)
