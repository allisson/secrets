package usecase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationTesting "github.com/allisson/secrets/internal/tokenization/testing"
)

func TestGetKek(t *testing.T) {
	masterKey := tokenizationTesting.CreateMasterKey()
	kekChain := tokenizationTesting.CreateKekChain(masterKey)
	defer kekChain.Close()

	activeKek := tokenizationTesting.GetActiveKek(kekChain)

	t.Run("Success_GetActiveKek", func(t *testing.T) {
		kek, err := getKek(kekChain, activeKek.ID)
		assert.NoError(t, err)
		assert.NotNil(t, kek)
		assert.Equal(t, activeKek.ID, kek.ID)
	})

	t.Run("Error_KekNotFound", func(t *testing.T) {
		randomID := uuid.Must(uuid.NewV7())
		kek, err := getKek(kekChain, randomID)
		assert.ErrorIs(t, err, cryptoDomain.ErrKekNotFound)
		assert.Nil(t, kek)
	})
}
