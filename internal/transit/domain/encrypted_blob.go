package domain

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// EncryptedBlob represents an encrypted data blob with version and ciphertext.
// Format: "version:ciphertext-base64"
type EncryptedBlob struct {
	Version    uint   // Transit key version used for encryption
	Ciphertext []byte // Encrypted data
	Plaintext  []byte // In memory only
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
