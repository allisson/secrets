package domain

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// EncryptedBlob represents an encrypted data blob in the transit encryption system.
//
// The blob contains the transit key name, version, and encrypted ciphertext.
// It can be serialized to and deserialized from the format: "name:version:ciphertext-base64"
//
// Fields:
//   - Name: The transit key name (e.g., "payment-encryption")
//   - Version: The transit key version used for encryption
//   - Ciphertext: The encrypted data as raw bytes
type EncryptedBlob struct {
	Name       string
	Version    uint
	Ciphertext []byte
}

// NewEncryptedBlob creates an EncryptedBlob from its string representation.
//
// The input string must be in the format: "name:version:ciphertext-base64"
// where:
//   - name: non-empty string identifying the transit key
//   - version: non-negative integer (uint)
//   - ciphertext-base64: base64-encoded encrypted data (can be empty)
//
// Parameters:
//   - content: String in format "name:version:ciphertext-base64"
//
// Returns:
//   - EncryptedBlob instance if parsing succeeds
//   - ErrInvalidBlobFormat if the format is incorrect (not 3 colon-separated parts)
//   - ErrEmptyBlobName if the name field is empty
//   - ErrInvalidBlobVersion if the version cannot be parsed as uint
//   - ErrInvalidBlobBase64 if the ciphertext is not valid base64
//
// Example:
//
//	blob, err := NewEncryptedBlob("payment-key:1:SGVsbG8gV29ybGQ=")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Name: %s, Version: %d\n", blob.Name, blob.Version)
func NewEncryptedBlob(content string) (EncryptedBlob, error) {
	// Split by ":" - expect exactly 3 parts: name:version:ciphertext
	parts := strings.Split(content, ":")
	if len(parts) != 3 {
		return EncryptedBlob{}, fmt.Errorf(
			"%w: expected format 'name:version:ciphertext', got %d parts",
			ErrInvalidBlobFormat,
			len(parts),
		)
	}

	name := parts[0]
	if name == "" {
		return EncryptedBlob{}, ErrEmptyBlobName
	}

	// Parse version as uint
	version, err := strconv.ParseUint(parts[1], 10, 0)
	if err != nil {
		return EncryptedBlob{}, fmt.Errorf("%w: %v", ErrInvalidBlobVersion, err)
	}

	// Decode base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return EncryptedBlob{}, fmt.Errorf("%w: %v", ErrInvalidBlobBase64, err)
	}

	return EncryptedBlob{
		Name:       name,
		Version:    uint(version),
		Ciphertext: ciphertext,
	}, nil
}

// String serializes the EncryptedBlob to its string representation.
//
// The output format is: "name:version:ciphertext-base64"
//
// This method provides round-trip serialization with NewEncryptedBlob:
//
//	original := EncryptedBlob{Name: "key", Version: 1, Ciphertext: []byte("data")}
//	serialized := original.String()
//	parsed, _ := NewEncryptedBlob(serialized)
//	// parsed equals original
//
// Returns:
//   - String representation in format "name:version:ciphertext-base64"
func (eb EncryptedBlob) String() string {
	encodedCiphertext := base64.StdEncoding.EncodeToString(eb.Ciphertext)
	return fmt.Sprintf("%s:%d:%s", eb.Name, eb.Version, encodedCiphertext)
}
