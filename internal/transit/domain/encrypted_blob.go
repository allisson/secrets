package domain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// EncryptedBlob represents an encrypted data blob with version and ciphertext.
// Format: "version:ciphertext-base64"
type EncryptedBlob struct {
	Version    uint   // Transit key version used for this encryption/decryption operation
	Ciphertext []byte // Encrypted data with nonce prepended (empty after decryption)
	Plaintext  []byte // Decrypted data (only populated after decryption, should be zeroed after use)
}

// NewEncryptedBlob creates an EncryptedBlob from string format "version:ciphertext-base64".
func NewEncryptedBlob(content string) (EncryptedBlob, error) {
	// Split by ":" - expect exactly 2 parts: version:ciphertext
	parts := strings.Split(content, ":")

	if len(parts) != 2 {
		return EncryptedBlob{}, fmt.Errorf(
			"%w: expected format 'version:ciphertext', got %d parts",
			ErrInvalidBlobFormat,
			len(parts),
		)
	}

	// Parse version as uint
	version, err := strconv.ParseUint(parts[0], 10, 0)
	if err != nil {
		return EncryptedBlob{}, fmt.Errorf("%w: %v", ErrInvalidBlobVersion, err)
	}

	// Decode base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return EncryptedBlob{}, fmt.Errorf("%w: %v", ErrInvalidBlobBase64, err)
	}

	return EncryptedBlob{
		Version:    uint(version),
		Ciphertext: ciphertext,
	}, nil
}

// String serializes the EncryptedBlob to format "version:ciphertext-base64".
func (eb EncryptedBlob) String() string {
	encodedCiphertext := base64.StdEncoding.EncodeToString(eb.Ciphertext)
	return fmt.Sprintf("%d:%s", eb.Version, encodedCiphertext)
}

// Validate checks if the encrypted blob contains valid data.
// Returns an error if any field violates domain constraints.
func (eb *EncryptedBlob) Validate() error {
	if eb.Version == 0 {
		return errors.New("encrypted blob version must be greater than 0")
	}

	// Must have either ciphertext (for encryption result) or plaintext (for decryption result)
	if len(eb.Ciphertext) == 0 && len(eb.Plaintext) == 0 {
		return errors.New("encrypted blob must contain either ciphertext or plaintext")
	}

	return nil
}
