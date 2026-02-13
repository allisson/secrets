package dto

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

func TestMapTransitKeyToResponse(t *testing.T) {
	t.Run("Success_ValidMapping", func(t *testing.T) {
		transitKeyID := uuid.Must(uuid.NewV7())
		dekID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		transitKey := &transitDomain.TransitKey{
			ID:        transitKeyID,
			Name:      "test-key",
			Version:   1,
			DekID:     dekID,
			CreatedAt: now,
		}

		response := MapTransitKeyToResponse(transitKey)

		assert.Equal(t, transitKeyID.String(), response.ID)
		assert.Equal(t, "test-key", response.Name)
		assert.Equal(t, uint(1), response.Version)
		assert.Equal(t, dekID.String(), response.DekID)
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_VersionTwo", func(t *testing.T) {
		transitKeyID := uuid.Must(uuid.NewV7())
		dekID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		transitKey := &transitDomain.TransitKey{
			ID:        transitKeyID,
			Name:      "test-key",
			Version:   2,
			DekID:     dekID,
			CreatedAt: now,
		}

		response := MapTransitKeyToResponse(transitKey)

		assert.Equal(t, uint(2), response.Version)
	})
}

func TestEncryptResponse(t *testing.T) {
	t.Run("Success_CreateResponse", func(t *testing.T) {
		response := EncryptResponse{
			Ciphertext: "1:ZW5jcnlwdGVkLWRhdGE=",
			Version:    1,
		}

		assert.Equal(t, "1:ZW5jcnlwdGVkLWRhdGE=", response.Ciphertext)
		assert.Equal(t, uint(1), response.Version)
	})
}

func TestDecryptResponse(t *testing.T) {
	t.Run("Success_MapPlaintextToBase64", func(t *testing.T) {
		plaintext := []byte("my secret data")
		version := uint(1)

		response := MapDecryptResponse(plaintext, version)

		assert.Equal(t, base64.StdEncoding.EncodeToString(plaintext), response.Plaintext)
		assert.Equal(t, version, response.Version)
	})

	t.Run("Success_EmptyPlaintext", func(t *testing.T) {
		plaintext := []byte{}
		version := uint(2)

		response := MapDecryptResponse(plaintext, version)

		assert.Equal(t, base64.StdEncoding.EncodeToString(plaintext), response.Plaintext)
		assert.Equal(t, version, response.Version)
	})

	t.Run("Success_BinaryData", func(t *testing.T) {
		plaintext := []byte{0x00, 0xFF, 0xAB, 0xCD, 0xEF}
		version := uint(3)

		response := MapDecryptResponse(plaintext, version)

		assert.Equal(t, base64.StdEncoding.EncodeToString(plaintext), response.Plaintext)
		assert.Equal(t, version, response.Version)
	})
}
