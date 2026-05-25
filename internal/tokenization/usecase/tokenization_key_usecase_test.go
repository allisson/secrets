package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	"github.com/allisson/secrets/internal/tokenization/usecase"
	"github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

// noopTxManager runs the function with no real transaction.
type noopTxManager struct{}

func (noopTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newTokenizationKeyUseCase(
	t *testing.T,
) (usecase.TokenizationKeyUseCase, *keyring.Fake, *mocks.MockTokenizationKeyRepository) {
	t.Helper()
	fake := keyring.NewFake()
	repo := mocks.NewMockTokenizationKeyRepository(t)
	uc := usecase.NewTokenizationKeyUseCase(noopTxManager{}, repo, fake)
	return uc, fake, repo
}

func TestTokenizationKeyUseCase_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_CreateKeyWithUUIDFormat", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)

		repo.EXPECT().
			GetByNameAndVersion(ctx, "test-key", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound)
		repo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(k *tokenizationDomain.TokenizationKey) bool {
				return k.Name == "test-key" &&
					k.FormatType == tokenizationDomain.FormatUUID &&
					k.Version == 1 &&
					!k.IsDeterministic &&
					len(k.Salt) == 32 &&
					k.DekID != [16]byte{}
			})).
			Return(nil)

		key, err := uc.Create(ctx, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.Equal(t, "test-key", key.Name)
		assert.Equal(t, uint(1), key.Version)
	})

	t.Run("Error_KeyAlreadyExists", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)
		repo.EXPECT().
			GetByNameAndVersion(ctx, "dup", uint(1)).
			Return(&tokenizationDomain.TokenizationKey{}, nil)

		_, err := uc.Create(ctx, "dup", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, tokenizationDomain.ErrTokenizationKeyAlreadyExists)
	})

	t.Run("Error_InvalidFormatType", func(t *testing.T) {
		t.Parallel()
		uc, _, _ := newTokenizationKeyUseCase(t)
		_, err := uc.Create(ctx, "k", tokenizationDomain.FormatType("nope"), false, cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, tokenizationDomain.ErrInvalidFormatType)
	})

	t.Run("Error_KeyringAllocateFails", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newTokenizationKeyUseCase(t)
		fake.FailAllocate = apperrors.Wrap(apperrors.ErrInvalidInput, "kms down")

		repo.EXPECT().
			GetByNameAndVersion(ctx, "k", uint(1)).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound)

		_, err := uc.Create(ctx, "k", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)
		assert.Error(t, err)
	})
}

func TestTokenizationKeyUseCase_Rotate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_IncrementsVersion", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)

		repo.EXPECT().
			GetByName(ctx, "k").
			Return(&tokenizationDomain.TokenizationKey{
				Name:    "k",
				Version: 2,
			}, nil)
		repo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(k *tokenizationDomain.TokenizationKey) bool {
				return k.Version == 3
			})).
			Return(nil)

		key, err := uc.Rotate(ctx, "k", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.Equal(t, uint(3), key.Version)
	})

	t.Run("Success_CreatesFirstVersionWhenAbsent", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)

		repo.EXPECT().GetByName(ctx, "new").Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound)
		repo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(k *tokenizationDomain.TokenizationKey) bool {
				return k.Version == 1
			})).
			Return(nil)

		key, err := uc.Rotate(ctx, "new", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.Equal(t, uint(1), key.Version)
	})
}

func TestTokenizationKeyUseCase_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	uc, _, repo := newTokenizationKeyUseCase(t)
	repo.EXPECT().Delete(ctx, "k").Return(nil)
	assert.NoError(t, uc.Delete(ctx, "k"))
}

func TestTokenizationKeyUseCase_GetByName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)
		want := &tokenizationDomain.TokenizationKey{Name: "k", Version: 1}
		repo.EXPECT().GetByName(ctx, "k").Return(want, nil)

		got, err := uc.GetByName(ctx, "k")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)
		repo.EXPECT().GetByName(ctx, "k").Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound)

		_, err := uc.GetByName(ctx, "k")
		assert.ErrorIs(t, err, tokenizationDomain.ErrTokenizationKeyNotFound)
	})
}

func TestTokenizationKeyUseCase_PurgeDeleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Error_NegativeDays", func(t *testing.T) {
		t.Parallel()
		uc, _, _ := newTokenizationKeyUseCase(t)
		_, err := uc.PurgeDeleted(ctx, -1, false)
		assert.Error(t, err)
	})

	t.Run("Success_DryRun", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTokenizationKeyUseCase(t)
		repo.EXPECT().HardDelete(ctx, mock.Anything, true).Return(int64(3), nil)

		n, err := uc.PurgeDeleted(ctx, 30, true)
		require.NoError(t, err)
		assert.EqualValues(t, 3, n)
	})
}
