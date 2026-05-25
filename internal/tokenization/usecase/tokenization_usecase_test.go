package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	"github.com/allisson/secrets/internal/tokenization/usecase"
	"github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

func newTokenizationUseCase(
	t *testing.T,
) (
	usecase.TokenizationUseCase,
	*keyring.Fake,
	*mocks.MockTokenizationKeyRepository,
	*mocks.MockTokenRepository,
	*mocks.MockHashService,
) {
	t.Helper()
	fake := keyring.NewFake()
	keyRepo := mocks.NewMockTokenizationKeyRepository(t)
	tokenRepo := mocks.NewMockTokenRepository(t)
	hashSvc := mocks.NewMockHashService(t)
	uc := usecase.NewTokenizationUseCase(noopTxManager{}, keyRepo, tokenRepo, hashSvc, fake)
	return uc, fake, keyRepo, tokenRepo, hashSvc
}

// allocateDekForTest seeds the keyring Fake with a DekID and returns it,
// mimicking a tokenization key previously created via uc.Create.
func allocateDekForTest(t *testing.T, fake *keyring.Fake) uuid.UUID {
	t.Helper()
	handle, err := fake.AllocateDek(context.Background(), keyring.AESGCM)
	require.NoError(t, err)
	return handle.DekID
}

func TestTokenizationUseCase_Tokenize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_NonDeterministic", func(t *testing.T) {
		t.Parallel()
		uc, fake, keyRepo, tokenRepo, _ := newTokenizationUseCase(t)
		dekID := allocateDekForTest(t, fake)

		key := &tokenizationDomain.TokenizationKey{
			ID:              uuid.New(),
			Name:            "k",
			Version:         1,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			DekID:           dekID,
		}
		keyRepo.EXPECT().GetByName(ctx, "k").Return(key, nil)
		tokenRepo.EXPECT().Create(ctx, mock.MatchedBy(func(tok *tokenizationDomain.Token) bool {
			return tok.TokenizationKeyID == key.ID &&
				len(tok.Token) > 0 &&
				len(tok.Ciphertext) > 0 &&
				tok.ValueHash == nil
		})).Return(nil)

		got, err := uc.Tokenize(ctx, "k", []byte("payload"), nil, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, got.Token)
	})

	t.Run("Success_Deterministic_ReturnsExistingValidToken", func(t *testing.T) {
		t.Parallel()
		uc, fake, keyRepo, tokenRepo, hashSvc := newTokenizationUseCase(t)
		dekID := allocateDekForTest(t, fake)

		key := &tokenizationDomain.TokenizationKey{
			ID:              uuid.New(),
			Name:            "k",
			Version:         1,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: true,
			Salt:            []byte("salt"),
			DekID:           dekID,
		}
		existing := &tokenizationDomain.Token{
			ID:                uuid.New(),
			TokenizationKeyID: key.ID,
			Token:             "existing-token",
		}

		keyRepo.EXPECT().GetByName(ctx, "k").Return(key, nil)
		hashSvc.EXPECT().Hash([]byte("payload"), []byte("salt")).Return("hash-value")
		tokenRepo.EXPECT().
			GetByValueHash(ctx, key.ID, "hash-value").
			Return(existing, nil)

		got, err := uc.Tokenize(ctx, "k", []byte("payload"), nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "existing-token", got.Token)
	})

	t.Run("Error_PlaintextEmpty", func(t *testing.T) {
		t.Parallel()
		uc, _, _, _, _ := newTokenizationUseCase(t)
		_, err := uc.Tokenize(ctx, "k", nil, nil, nil)
		assert.ErrorIs(t, err, tokenizationDomain.ErrPlaintextEmpty)
	})

	t.Run("Error_PlaintextTooLarge", func(t *testing.T) {
		t.Parallel()
		uc, _, _, _, _ := newTokenizationUseCase(t)
		big := make([]byte, tokenizationDomain.MaxPlaintextSize+1)
		_, err := uc.Tokenize(ctx, "k", big, nil, nil)
		assert.ErrorIs(t, err, tokenizationDomain.ErrPlaintextTooLarge)
	})

	t.Run("Error_KeyNotFound", func(t *testing.T) {
		t.Parallel()
		uc, _, keyRepo, _, _ := newTokenizationUseCase(t)
		keyRepo.EXPECT().
			GetByName(ctx, "missing").
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound)

		_, err := uc.Tokenize(ctx, "missing", []byte("x"), nil, nil)
		assert.ErrorIs(t, err, tokenizationDomain.ErrTokenizationKeyNotFound)
	})

	t.Run("Error_KeyringEncryptFails", func(t *testing.T) {
		t.Parallel()
		uc, fake, keyRepo, _, _ := newTokenizationUseCase(t)
		dekID := allocateDekForTest(t, fake)
		fake.FailEncrypt = apperrors.New("boom")

		keyRepo.EXPECT().GetByName(ctx, "k").Return(&tokenizationDomain.TokenizationKey{
			ID:         uuid.New(),
			FormatType: tokenizationDomain.FormatUUID,
			DekID:      dekID,
		}, nil)

		_, err := uc.Tokenize(ctx, "k", []byte("payload"), nil, nil)
		assert.Error(t, err)
	})
}

func TestTokenizationUseCase_Detokenize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_RoundTrip", func(t *testing.T) {
		t.Parallel()
		uc, fake, keyRepo, tokenRepo, _ := newTokenizationUseCase(t)

		// Encrypt via the fake to get matching ciphertext/nonce/dekID.
		handle, err := fake.AllocateDek(ctx, keyring.AESGCM)
		require.NoError(t, err)
		plaintext := []byte("4111111111111111")
		ciphertext, nonce, err := fake.EncryptWith(ctx, handle, plaintext, nil)
		require.NoError(t, err)

		key := &tokenizationDomain.TokenizationKey{
			ID:    uuid.New(),
			Name:  "cards",
			DekID: handle.DekID,
		}
		tokenRec := &tokenizationDomain.Token{
			TokenizationKeyID: key.ID,
			Token:             "tok",
			Ciphertext:        ciphertext,
			Nonce:             nonce,
			CreatedAt:         time.Now().UTC(),
		}

		tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(tokenRec, nil)
		keyRepo.EXPECT().Get(ctx, key.ID).Return(key, nil)

		got, _, err := uc.Detokenize(ctx, "tok")
		require.NoError(t, err)
		assert.Equal(t, plaintext, got)
	})

	t.Run("Error_TokenExpired", func(t *testing.T) {
		t.Parallel()
		uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
		past := time.Now().Add(-time.Hour)
		tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(&tokenizationDomain.Token{
			Token:     "tok",
			ExpiresAt: &past,
		}, nil)

		_, _, err := uc.Detokenize(ctx, "tok")
		assert.ErrorIs(t, err, tokenizationDomain.ErrTokenExpired)
	})

	t.Run("Error_TokenRevoked", func(t *testing.T) {
		t.Parallel()
		uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
		now := time.Now()
		tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(&tokenizationDomain.Token{
			Token:     "tok",
			RevokedAt: &now,
		}, nil)

		_, _, err := uc.Detokenize(ctx, "tok")
		assert.ErrorIs(t, err, tokenizationDomain.ErrTokenRevoked)
	})

	t.Run("Error_DecryptFails", func(t *testing.T) {
		t.Parallel()
		uc, fake, keyRepo, tokenRepo, _ := newTokenizationUseCase(t)
		dekID := allocateDekForTest(t, fake)
		fake.FailDecrypt = apperrors.New("AEAD tag mismatch")

		tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(&tokenizationDomain.Token{
			Token:             "tok",
			TokenizationKeyID: uuid.New(),
			Ciphertext:        []byte("ct"),
			Nonce:             []byte("nonce"),
		}, nil)
		keyRepo.EXPECT().Get(ctx, mock.Anything).Return(&tokenizationDomain.TokenizationKey{
			DekID: dekID,
		}, nil)

		_, _, err := uc.Detokenize(ctx, "tok")
		assert.ErrorIs(t, err, keyring.ErrDecryptionFailed)
	})
}

func TestTokenizationUseCase_Validate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()
		uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
		tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(&tokenizationDomain.Token{
			Token:     "tok",
			CreatedAt: time.Now(),
		}, nil)

		ok, err := uc.Validate(ctx, "tok")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("NotFound_ReturnsFalseNoError", func(t *testing.T) {
		t.Parallel()
		uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
		tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(nil, tokenizationDomain.ErrTokenNotFound)

		ok, err := uc.Validate(ctx, "tok")
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestTokenizationUseCase_Revoke(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
	tokenRepo.EXPECT().GetByToken(ctx, "tok").Return(&tokenizationDomain.Token{Token: "tok"}, nil)
	tokenRepo.EXPECT().Revoke(ctx, "tok").Return(nil)

	assert.NoError(t, uc.Revoke(ctx, "tok"))
}

func TestTokenizationUseCase_CleanupExpired(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Error_NegativeDays", func(t *testing.T) {
		t.Parallel()
		uc, _, _, _, _ := newTokenizationUseCase(t)
		_, err := uc.CleanupExpired(ctx, -1, false)
		assert.Error(t, err)
	})

	t.Run("Success_DryRun", func(t *testing.T) {
		t.Parallel()
		uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
		tokenRepo.EXPECT().CountExpired(ctx, mock.Anything).Return(int64(7), nil)

		n, err := uc.CleanupExpired(ctx, 30, true)
		require.NoError(t, err)
		assert.EqualValues(t, 7, n)
	})

	t.Run("Success_Delete", func(t *testing.T) {
		t.Parallel()
		uc, _, _, tokenRepo, _ := newTokenizationUseCase(t)
		tokenRepo.EXPECT().DeleteExpired(ctx, mock.Anything).Return(int64(4), nil)

		n, err := uc.CleanupExpired(ctx, 30, false)
		require.NoError(t, err)
		assert.EqualValues(t, 4, n)
	})
}
