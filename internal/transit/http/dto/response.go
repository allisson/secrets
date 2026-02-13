// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"encoding/base64"
	"time"

	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// TransitKeyResponse represents a transit key in API responses.
type TransitKeyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   uint      `json:"version"`
	DekID     string    `json:"dek_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MapTransitKeyToResponse converts a domain transit key to an API response.
func MapTransitKeyToResponse(transitKey *transitDomain.TransitKey) TransitKeyResponse {
	return TransitKeyResponse{
		ID:        transitKey.ID.String(),
		Name:      transitKey.Name,
		Version:   transitKey.Version,
		DekID:     transitKey.DekID.String(),
		CreatedAt: transitKey.CreatedAt,
	}
}

// EncryptResponse contains the result of an encryption operation.
type EncryptResponse struct {
	Ciphertext string `json:"ciphertext"` // Format: "version:base64-ciphertext"
	Version    uint   `json:"version"`
}

// DecryptResponse contains the result of a decryption operation.
// SECURITY: The Plaintext field contains sensitive data and should be transmitted over HTTPS.
type DecryptResponse struct {
	Plaintext string `json:"plaintext"` // Base64-encoded plaintext
	Version   uint   `json:"version"`
}

// MapDecryptResponse converts plaintext bytes and version to an API response.
func MapDecryptResponse(plaintext []byte, version uint) DecryptResponse {
	return DecryptResponse{
		Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		Version:   version,
	}
}
