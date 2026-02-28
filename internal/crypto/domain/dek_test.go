package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDek(t *testing.T) {
	t.Run("dek initialization", func(t *testing.T) {
		id := uuid.New()
		kekID := uuid.New()
		now := time.Now()
		encryptedKey := []byte("encrypted-key")
		nonce := []byte("nonce")

		dek := Dek{
			ID:           id,
			KekID:        kekID,
			Algorithm:    AESGCM,
			EncryptedKey: encryptedKey,
			Nonce:        nonce,
			CreatedAt:    now,
		}

		assert.Equal(t, id, dek.ID)
		assert.Equal(t, kekID, dek.KekID)
		assert.Equal(t, AESGCM, dek.Algorithm)
		assert.Equal(t, encryptedKey, dek.EncryptedKey)
		assert.Equal(t, nonce, dek.Nonce)
		assert.Equal(t, now, dek.CreatedAt)
	})
}
