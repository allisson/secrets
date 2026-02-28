package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTransitKey_Validate(t *testing.T) {
	validTransitKey := &TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "test-key",
		Version:   1,
		DekID:     uuid.Must(uuid.NewV7()),
		CreatedAt: time.Now().UTC(),
		DeletedAt: nil,
	}

	t.Run("Success_ValidTransitKey", func(t *testing.T) {
		err := validTransitKey.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyName", func(t *testing.T) {
		key := *validTransitKey
		key.Name = ""

		err := key.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("Error_NameTooLong", func(t *testing.T) {
		key := *validTransitKey
		key.Name = strings.Repeat("a", MaxTransitKeyNameLength+1)

		err := key.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("Success_NameAtMaxLength", func(t *testing.T) {
		key := *validTransitKey
		key.Name = strings.Repeat("a", MaxTransitKeyNameLength)

		err := key.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_ZeroVersion", func(t *testing.T) {
		key := *validTransitKey
		key.Version = 0

		err := key.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version must be greater than 0")
	})

	t.Run("Error_NilDekID", func(t *testing.T) {
		key := *validTransitKey
		key.DekID = uuid.Nil

		err := key.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "valid DEK ID")
	})

	t.Run("Error_ZeroCreatedAt", func(t *testing.T) {
		key := *validTransitKey
		key.CreatedAt = time.Time{}

		err := key.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "valid created_at timestamp")
	})

	t.Run("Success_WithDeletedAt", func(t *testing.T) {
		key := *validTransitKey
		now := time.Now().UTC()
		key.DeletedAt = &now

		err := key.Validate()
		assert.NoError(t, err)
	})
}
