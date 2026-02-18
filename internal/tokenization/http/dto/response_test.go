package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

func TestMapTokenizationKeyToResponse(t *testing.T) {
	t.Run("Success_MapAllFields_UUID", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		key := &tokenizationDomain.TokenizationKey{
			ID:              id,
			Name:            "test-key",
			Version:         1,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       now,
		}

		response := MapTokenizationKeyToResponse(key)

		assert.Equal(t, id.String(), response.ID)
		assert.Equal(t, "test-key", response.Name)
		assert.Equal(t, uint(1), response.Version)
		assert.Equal(t, "uuid", response.FormatType)
		assert.False(t, response.IsDeterministic)
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_MapAllFields_Numeric", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		key := &tokenizationDomain.TokenizationKey{
			ID:              id,
			Name:            "numeric-key",
			Version:         2,
			FormatType:      tokenizationDomain.FormatNumeric,
			IsDeterministic: true,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       now,
		}

		response := MapTokenizationKeyToResponse(key)

		assert.Equal(t, id.String(), response.ID)
		assert.Equal(t, "numeric-key", response.Name)
		assert.Equal(t, uint(2), response.Version)
		assert.Equal(t, "numeric", response.FormatType)
		assert.True(t, response.IsDeterministic)
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_MapAllFields_LuhnPreserving", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		key := &tokenizationDomain.TokenizationKey{
			ID:              id,
			Name:            "luhn-key",
			Version:         3,
			FormatType:      tokenizationDomain.FormatLuhnPreserving,
			IsDeterministic: true,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       now,
		}

		response := MapTokenizationKeyToResponse(key)

		assert.Equal(t, id.String(), response.ID)
		assert.Equal(t, "luhn-key", response.Name)
		assert.Equal(t, uint(3), response.Version)
		assert.Equal(t, "luhn-preserving", response.FormatType)
		assert.True(t, response.IsDeterministic)
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_MapAllFields_Alphanumeric", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		key := &tokenizationDomain.TokenizationKey{
			ID:              id,
			Name:            "alpha-key",
			Version:         4,
			FormatType:      tokenizationDomain.FormatAlphanumeric,
			IsDeterministic: false,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       now,
		}

		response := MapTokenizationKeyToResponse(key)

		assert.Equal(t, id.String(), response.ID)
		assert.Equal(t, "alpha-key", response.Name)
		assert.Equal(t, uint(4), response.Version)
		assert.Equal(t, "alphanumeric", response.FormatType)
		assert.False(t, response.IsDeterministic)
		assert.Equal(t, now, response.CreatedAt)
	})
}

func TestMapTokenToTokenizeResponse(t *testing.T) {
	t.Run("Success_WithMetadataAndExpiration", func(t *testing.T) {
		now := time.Now().UTC()
		expiresAt := now.Add(time.Hour)
		valueHash := "hash"

		token := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             "tok_1234567890",
			Ciphertext:        []byte("encrypted"),
			Nonce:             []byte("nonce"),
			ValueHash:         &valueHash,
			Metadata:          map[string]any{"key": "value", "count": 42},
			CreatedAt:         now,
			ExpiresAt:         &expiresAt,
			RevokedAt:         nil,
		}

		response := MapTokenToTokenizeResponse(token)

		assert.Equal(t, "tok_1234567890", response.Token)
		assert.NotNil(t, response.Metadata)
		assert.Equal(t, "value", response.Metadata["key"])
		assert.Equal(t, 42, response.Metadata["count"])
		assert.Equal(t, now, response.CreatedAt)
		assert.NotNil(t, response.ExpiresAt)
		assert.Equal(t, expiresAt, *response.ExpiresAt)
	})

	t.Run("Success_WithoutMetadata", func(t *testing.T) {
		now := time.Now().UTC()
		expiresAt := now.Add(time.Hour)
		valueHash := "hash"

		token := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             "tok_9876543210",
			Ciphertext:        []byte("encrypted"),
			Nonce:             []byte("nonce"),
			ValueHash:         &valueHash,
			Metadata:          nil,
			CreatedAt:         now,
			ExpiresAt:         &expiresAt,
			RevokedAt:         nil,
		}

		response := MapTokenToTokenizeResponse(token)

		assert.Equal(t, "tok_9876543210", response.Token)
		assert.Nil(t, response.Metadata)
		assert.Equal(t, now, response.CreatedAt)
		assert.NotNil(t, response.ExpiresAt)
		assert.Equal(t, expiresAt, *response.ExpiresAt)
	})

	t.Run("Success_WithoutExpiration", func(t *testing.T) {
		now := time.Now().UTC()
		valueHash := "hash"

		token := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             "tok_permanent",
			Ciphertext:        []byte("encrypted"),
			Nonce:             []byte("nonce"),
			ValueHash:         &valueHash,
			Metadata:          map[string]any{"type": "permanent"},
			CreatedAt:         now,
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		response := MapTokenToTokenizeResponse(token)

		assert.Equal(t, "tok_permanent", response.Token)
		assert.NotNil(t, response.Metadata)
		assert.Equal(t, "permanent", response.Metadata["type"])
		assert.Equal(t, now, response.CreatedAt)
		assert.Nil(t, response.ExpiresAt)
	})

	t.Run("Success_EmptyMetadataMap", func(t *testing.T) {
		now := time.Now().UTC()
		valueHash := "hash"

		token := &tokenizationDomain.Token{
			ID:                uuid.Must(uuid.NewV7()),
			TokenizationKeyID: uuid.Must(uuid.NewV7()),
			Token:             "tok_empty_metadata",
			Ciphertext:        []byte("encrypted"),
			Nonce:             []byte("nonce"),
			ValueHash:         &valueHash,
			Metadata:          map[string]any{},
			CreatedAt:         now,
			ExpiresAt:         nil,
			RevokedAt:         nil,
		}

		response := MapTokenToTokenizeResponse(token)

		assert.Equal(t, "tok_empty_metadata", response.Token)
		assert.NotNil(t, response.Metadata)
		assert.Empty(t, response.Metadata)
		assert.Equal(t, now, response.CreatedAt)
		assert.Nil(t, response.ExpiresAt)
	})
}

func TestDetokenizeResponse(t *testing.T) {
	t.Run("Success_CreateResponse_WithMetadata", func(t *testing.T) {
		response := DetokenizeResponse{
			Plaintext: "SGVsbG8gV29ybGQ=",
			Metadata:  map[string]any{"key": "value"},
		}

		assert.Equal(t, "SGVsbG8gV29ybGQ=", response.Plaintext)
		assert.NotNil(t, response.Metadata)
		assert.Equal(t, "value", response.Metadata["key"])
	})

	t.Run("Success_CreateResponse_WithoutMetadata", func(t *testing.T) {
		response := DetokenizeResponse{
			Plaintext: "SGVsbG8gV29ybGQ=",
			Metadata:  nil,
		}

		assert.Equal(t, "SGVsbG8gV29ybGQ=", response.Plaintext)
		assert.Nil(t, response.Metadata)
	})
}

func TestValidateTokenResponse(t *testing.T) {
	t.Run("Success_ValidToken", func(t *testing.T) {
		response := ValidateTokenResponse{
			Valid: true,
		}

		assert.True(t, response.Valid)
	})

	t.Run("Success_InvalidToken", func(t *testing.T) {
		response := ValidateTokenResponse{
			Valid: false,
		}

		assert.False(t, response.Valid)
	})
}
