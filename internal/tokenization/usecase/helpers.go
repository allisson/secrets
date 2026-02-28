package usecase

import (
	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// getKek retrieves a KEK from the chain by its ID.
// Returns ErrKekNotFound if the KEK is not in the chain.
func getKek(kekChain *cryptoDomain.KekChain, kekID uuid.UUID) (*cryptoDomain.Kek, error) {
	kek, ok := kekChain.Get(kekID)
	if !ok {
		return nil, cryptoDomain.ErrKekNotFound
	}
	return kek, nil
}
