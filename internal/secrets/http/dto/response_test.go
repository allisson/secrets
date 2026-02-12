package dto

import (
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
		assert.Equal(t, plaintext, response.Value) // Value should be included
		assert.Equal(t, now, response.CreatedAt)
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

		assert.Equal(t, plaintext, response.Value)
		assert.Len(t, response.Value, 10000)
	})
}
