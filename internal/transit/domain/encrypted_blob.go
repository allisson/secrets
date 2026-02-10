package domain

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// EncryptedBlob represents an encrypted data blob in the transit encryption system.
//
// The blob contains the version and encrypted ciphertext.
// It can be serialized to and deserialized from the format: "version:ciphertext-base64"
//
// The transit key name is not included in the blob as it is passed separately
// in API calls (e.g., /transit/decrypt/{key_name}).
//
// Fields:
//   - Version: The transit key version used for encryption
//   - Ciphertext: The encrypted data as raw bytes
type EncryptedBlob struct {
	Version    uint
	Ciphertext []byte
	Plaintext  []byte // In memory only
}

// NewEncryptedBlob creates an EncryptedBlob from its string representation.
//
// The input string must be in the format: "version:ciphertext-base64"
// where:
//   - version: non-negative integer (uint)
//   - ciphertext-base64: base64-encoded encrypted data (can be empty)
//
// Parameters:
//   - content: String in format "version:ciphertext-base64"
//
// Returns:
//   - EncryptedBlob instance if parsing succeeds
//   - ErrInvalidBlobFormat if the format is incorrect (not 2 colon-separated parts)
//   - ErrInvalidBlobVersion if the version cannot be parsed as uint
//   - ErrInvalidBlobBase64 if the ciphertext is not valid base64
//
// Example:
//
//	blob, err := NewEncryptedBlob("1:SGVsbG8gV29ybGQ=")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Version: %d\n", blob.Version)
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

// String serializes the EncryptedBlob to its string representation.
//
// The output format is: "version:ciphertext-base64"
//
// This method provides round-trip serialization with NewEncryptedBlob:
//
//	original := EncryptedBlob{Version: 1, Ciphertext: []byte("data")}
//	serialized := original.String()
//	parsed, _ := NewEncryptedBlob(serialized)
//	// parsed equals original
//
// Returns:
//   - String representation in format "version:ciphertext-base64"
func (eb EncryptedBlob) String() string {
	encodedCiphertext := base64.StdEncoding.EncodeToString(eb.Ciphertext)
	return fmt.Sprintf("%d:%s", eb.Version, encodedCiphertext)
}
