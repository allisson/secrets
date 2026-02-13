package dto

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

func TestMapSecretToCreateResponse(t *testing.T) {
	t.Run("Success_MapAllFields", func(t *testing.T) {
		secretID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		secret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      "database/password",
			Version:   1,
			Plaintext: []byte("secret-value"), // Should not be included
			CreatedAt: now,
		}

		response := MapSecretToCreateResponse(secret)

		assert.Equal(t, secretID.String(), response.ID)
		assert.Equal(t, "database/password", response.Path)
		assert.Equal(t, uint(1), response.Version)
		assert.Empty(t, response.Value) // Value should be empty for create response
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_NestedPath", func(t *testing.T) {
		secretID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		secret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      "my/nested/secret/path",
			Version:   5,
			CreatedAt: now,
		}

		response := MapSecretToCreateResponse(secret)

		assert.Equal(t, "my/nested/secret/path", response.Path)
		assert.Equal(t, uint(5), response.Version)
		assert.Empty(t, response.Value)
	})
}

func TestMapSecretToGetResponse(t *testing.T) {
	t.Run("Success_MapAllFieldsIncludingValue", func(t *testing.T) {
		secretID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()
		plaintext := []byte("super-secret-value")

		secret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      "database/password",
			Version:   1,
			Plaintext: plaintext,
			CreatedAt: now,
		}

		response := MapSecretToGetResponse(secret)

		assert.Equal(t, secretID.String(), response.ID)
		assert.Equal(t, "database/password", response.Path)
		assert.Equal(t, uint(1), response.Version)
		assert.Equal(
			t,
			base64.StdEncoding.EncodeToString(plaintext),
			response.Value,
		) // Value should be base64-encoded
		assert.Equal(t, now, response.CreatedAt)

		// Verify we can decode it back
		decoded, err := base64.StdEncoding.DecodeString(response.Value)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decoded)
	})

	t.Run("Success_EmptyPlaintext", func(t *testing.T) {
		secretID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		secret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      "database/password",
			Version:   2,
			Plaintext: []byte{}, // Empty plaintext
			CreatedAt: now,
		}

		response := MapSecretToGetResponse(secret)

		assert.Equal(t, secretID.String(), response.ID)
		assert.Empty(t, response.Value)
	})

	t.Run("Success_LargePlaintext", func(t *testing.T) {
		secretID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()
		plaintext := make([]byte, 10000) // 10KB value

		secret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      "large/secret",
			Version:   1,
			Plaintext: plaintext,
			CreatedAt: now,
		}

		response := MapSecretToGetResponse(secret)

		expectedBase64 := base64.StdEncoding.EncodeToString(plaintext)
		assert.Equal(t, expectedBase64, response.Value)

		// Verify decoding
		decoded, err := base64.StdEncoding.DecodeString(response.Value)
		assert.NoError(t, err)
		assert.Len(t, decoded, 10000)
		assert.Equal(t, plaintext, decoded)
	})

	t.Run("Success_BinaryData", func(t *testing.T) {
		secretID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

		secret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      "binary/secret",
			Version:   1,
			Plaintext: binaryData,
			CreatedAt: now,
		}

		response := MapSecretToGetResponse(secret)

		expectedBase64 := base64.StdEncoding.EncodeToString(binaryData)
		assert.Equal(t, expectedBase64, response.Value)

		// Verify decoding preserves binary data
		decoded, err := base64.StdEncoding.DecodeString(response.Value)
		assert.NoError(t, err)
		assert.Equal(t, binaryData, decoded)
	})
}
