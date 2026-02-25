package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoServiceMocks "github.com/allisson/secrets/internal/crypto/service/mocks"
	"github.com/allisson/secrets/internal/crypto/usecase"
	cryptoUsecaseMocks "github.com/allisson/secrets/internal/crypto/usecase/mocks"
	dbMocks "github.com/allisson/secrets/internal/database/mocks"
)

func TestDekUseCase_Rewrap(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		oldKekID := uuid.New()
		batchSize := 10

		oldKek := &cryptoDomain.Kek{
			ID:           oldKekID,
			MasterKeyID:  uuid.New().String(),
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("old-encrypted-key"),
			Key:          []byte("old-key"),
			Nonce:        []byte("old-nonce"),
			Version:      1,
		}

		newKek := &cryptoDomain.Kek{
			ID:           newKekID,
			MasterKeyID:  uuid.New().String(),
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("new-encrypted-key"),
			Key:          []byte("new-key"),
			Nonce:        []byte("new-nonce"),
			Version:      2,
		}

		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{newKek, oldKek})

		dek1 := &cryptoDomain.Dek{
			ID:           uuid.New(),
			KekID:        oldKekID,
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("dek1-encrypted-old"),
			Nonce:        []byte("dek1-nonce-old"),
			CreatedAt:    time.Now(),
		}

		batch := []*cryptoDomain.Dek{dek1}

		// Setup mock expectations
		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return(batch, nil)

		plainDek1Key := []byte("dek1-plaintext-key")
		keyManager.EXPECT().DecryptDek(dek1, oldKek).Return(plainDek1Key, nil)

		newEncDek1 := []byte("dek1-encrypted-new")
		newNonceDek1 := []byte("dek1-nonce-new")
		keyManager.EXPECT().EncryptDek(plainDek1Key, newKek).Return(newEncDek1, newNonceDek1, nil)

		dekRepo.EXPECT().Update(ctx, mock.MatchedBy(func(dek *cryptoDomain.Dek) bool {
			return dek.ID == dek1.ID &&
				dek.KekID == newKekID &&
				string(dek.EncryptedKey) == "dek1-encrypted-new" &&
				string(dek.Nonce) == "dek1-nonce-new"
		})).Return(nil)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.NoError(t, err)
		assert.Equal(t, 1, rewrapped)
	})

	t.Run("Zero DEKs to rewrap", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		newKek := &cryptoDomain.Kek{ID: newKekID, Key: []byte("new-key")}
		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{newKek})
		batchSize := 10

		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return([]*cryptoDomain.Dek{}, nil)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.NoError(t, err)
		assert.Equal(t, 0, rewrapped)
	})

	t.Run("New KEK not found in chain", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{{ID: uuid.New()}})
		batchSize := 10

		dek1 := &cryptoDomain.Dek{ID: uuid.New(), KekID: uuid.New()}
		batch := []*cryptoDomain.Dek{dek1}

		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return(batch, nil)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.ErrorIs(t, err, cryptoDomain.ErrKekNotFound)
		assert.Equal(t, 0, rewrapped)
	})

	t.Run("Old KEK not found in chain", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		oldKekID := uuid.New()
		batchSize := 10

		newKek := &cryptoDomain.Kek{ID: newKekID, Key: []byte("new-key")}
		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{newKek})

		dek1 := &cryptoDomain.Dek{ID: uuid.New(), KekID: oldKekID}
		batch := []*cryptoDomain.Dek{dek1}

		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return(batch, nil)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.ErrorIs(t, err, cryptoDomain.ErrKekNotFound)
		assert.Equal(t, 0, rewrapped)
	})

	t.Run("DecryptDek error", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		oldKekID := uuid.New()
		batchSize := 10

		newKek := &cryptoDomain.Kek{ID: newKekID, Key: []byte("new-key")}
		oldKek := &cryptoDomain.Kek{ID: oldKekID, Key: []byte("old-key")}
		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{newKek, oldKek})

		dek1 := &cryptoDomain.Dek{ID: uuid.New(), KekID: oldKekID}
		batch := []*cryptoDomain.Dek{dek1}

		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return(batch, nil)
		expectedErr := errors.New("decryption failed")
		keyManager.EXPECT().DecryptDek(dek1, oldKek).Return(nil, expectedErr)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 0, rewrapped)
	})

	t.Run("EncryptDek error", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		oldKekID := uuid.New()
		batchSize := 10

		newKek := &cryptoDomain.Kek{ID: newKekID, Key: []byte("new-key")}
		oldKek := &cryptoDomain.Kek{ID: oldKekID, Key: []byte("old-key")}
		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{newKek, oldKek})

		dek1 := &cryptoDomain.Dek{ID: uuid.New(), KekID: oldKekID}
		batch := []*cryptoDomain.Dek{dek1}

		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return(batch, nil)

		plainDek1Key := []byte("plain-key")
		keyManager.EXPECT().DecryptDek(dek1, oldKek).Return(plainDek1Key, nil)

		expectedErr := errors.New("encryption failed")
		keyManager.EXPECT().EncryptDek(plainDek1Key, newKek).Return(nil, nil, expectedErr)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 0, rewrapped)
	})

	t.Run("Update dek error", func(t *testing.T) {
		txManager := dbMocks.NewMockTxManager(t)
		dekRepo := cryptoUsecaseMocks.NewMockDekRepository(t)
		keyManager := cryptoServiceMocks.NewMockKeyManager(t)
		useCase := usecase.NewDekUseCase(txManager, dekRepo, keyManager)

		ctx := context.Background()
		newKekID := uuid.New()
		oldKekID := uuid.New()
		batchSize := 10

		newKek := &cryptoDomain.Kek{ID: newKekID, Key: []byte("new-key")}
		oldKek := &cryptoDomain.Kek{ID: oldKekID, Key: []byte("old-key")}
		kekChain := cryptoDomain.NewKekChain([]*cryptoDomain.Kek{newKek, oldKek})

		dek1 := &cryptoDomain.Dek{ID: uuid.New(), KekID: oldKekID}
		batch := []*cryptoDomain.Dek{dek1}

		dekRepo.EXPECT().GetBatchNotKekID(ctx, newKekID, batchSize).Return(batch, nil)

		plainDek1Key := []byte("plain-key")
		keyManager.EXPECT().DecryptDek(dek1, oldKek).Return(plainDek1Key, nil)
		keyManager.EXPECT().EncryptDek(plainDek1Key, newKek).Return([]byte("enc"), []byte("nonce"), nil)

		expectedErr := errors.New("update failed")
		dekRepo.EXPECT().Update(ctx, dek1).Return(expectedErr)

		rewrapped, err := useCase.Rewrap(ctx, kekChain, newKekID, batchSize)

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 0, rewrapped)
	})
}
