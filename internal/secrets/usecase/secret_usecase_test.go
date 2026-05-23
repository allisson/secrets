package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/keyring"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	"github.com/allisson/secrets/internal/secrets/usecase"
	"github.com/allisson/secrets/internal/secrets/usecase/mocks"
)

// noopTxManager runs the function with no real transaction. The secrets
// use case writes nothing the in-memory keyring Fake cares about, so the
// outer transaction is not load-bearing for these unit tests.
type noopTxManager struct{}

func (noopTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// newSecretUseCase builds a SecretUseCase wired to a Fake keyring and a
// mocked SecretRepository.
func newSecretUseCase(
	t *testing.T,
	sizeLimit int,
) (usecase.SecretUseCase, *keyring.Fake, *mocks.MockSecretRepository) {
	t.Helper()
	fake := keyring.NewFake()
	repo := mocks.NewMockSecretRepository(t)
	uc := usecase.NewSecretUseCase(noopTxManager{}, fake, repo, sizeLimit, nil)
	return uc, fake, repo
}

// =============================================================================
// CreateOrUpdate
// =============================================================================

func TestSecretUseCase_CreateOrUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_CreateNewSecret", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)
		path := "app/db-password"
		value := []byte("super-secret")

		repo.EXPECT().
			GetByPath(ctx, path).
			Return(nil, secretsDomain.ErrSecretNotFound)
		repo.EXPECT().
			Create(ctx, mock.MatchedBy(func(s *secretsDomain.Secret) bool {
				return s.Path == path && s.Version == 1 && len(s.Ciphertext) > 0
			})).
			Return(nil)

		got, err := uc.CreateOrUpdate(ctx, path, value)
		require.NoError(t, err)
		assert.Equal(t, path, got.Path)
		assert.EqualValues(t, 1, got.Version)
		assert.NotEqual(t, uuid.Nil, got.DekID)

		// Round-trip: decrypting via the same fake should give back value.
		plaintext, err := fake.Decrypt(ctx, keyring.Envelope{
			DekID:      got.DekID,
			Ciphertext: got.Ciphertext,
			Nonce:      got.Nonce,
		})
		require.NoError(t, err)
		assert.Equal(t, value, plaintext)
	})

	t.Run("Success_UpdateExistingSecret_VersionIncrements", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		path := "app/api-key"
		existing := &secretsDomain.Secret{Path: path, Version: 3}

		repo.EXPECT().GetByPath(ctx, path).Return(existing, nil)
		repo.EXPECT().Create(ctx, mock.MatchedBy(func(s *secretsDomain.Secret) bool {
			return s.Version == 4
		})).Return(nil)

		got, err := uc.CreateOrUpdate(ctx, path, []byte("v4"))
		require.NoError(t, err)
		assert.EqualValues(t, 4, got.Version)
	})

	t.Run("Error_KeyringEncryptFails", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)
		boom := errors.New("kms unavailable")
		fake.FailEncrypt = boom

		repo.EXPECT().
			GetByPath(ctx, "p").
			Return(nil, secretsDomain.ErrSecretNotFound)

		_, err := uc.CreateOrUpdate(ctx, "p", []byte("x"))
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Error_SecretRepoGetByPathFails", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		boom := errors.New("db down")
		repo.EXPECT().GetByPath(ctx, "p").Return(nil, boom)

		_, err := uc.CreateOrUpdate(ctx, "p", []byte("x"))
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Error_SecretRepoCreateFails", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		boom := errors.New("insert failed")

		repo.EXPECT().GetByPath(ctx, "p").Return(nil, secretsDomain.ErrSecretNotFound)
		repo.EXPECT().Create(ctx, mock.Anything).Return(boom)

		_, err := uc.CreateOrUpdate(ctx, "p", []byte("x"))
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Error_SecretValueTooLarge", func(t *testing.T) {
		t.Parallel()
		uc, _, _ := newSecretUseCase(t, 4)

		_, err := uc.CreateOrUpdate(ctx, "p", []byte("too-long-value"))
		assert.ErrorIs(t, err, secretsDomain.ErrSecretValueTooLarge)
	})

	t.Run("Error_InvalidPath", func(t *testing.T) {
		t.Parallel()
		uc, _, _ := newSecretUseCase(t, 1024)

		_, err := uc.CreateOrUpdate(ctx, "/leading-slash", []byte("x"))
		assert.ErrorIs(t, err, secretsDomain.ErrInvalidSecretPath)
	})
}

// =============================================================================
// Get / GetByVersion
// =============================================================================

func TestSecretUseCase_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_GetAndDecryptSecret", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)

		plaintext := []byte("payload")
		env, err := fake.Encrypt(ctx, plaintext)
		require.NoError(t, err)

		stored := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       "p",
			Version:    1,
			DekID:      env.DekID,
			Ciphertext: env.Ciphertext,
			Nonce:      env.Nonce,
			CreatedAt:  time.Now().UTC(),
		}
		repo.EXPECT().GetByPath(ctx, "p").Return(stored, nil)

		got, err := uc.Get(ctx, "p")
		require.NoError(t, err)
		assert.Equal(t, plaintext, got.Plaintext)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().GetByPath(ctx, "missing").Return(nil, secretsDomain.ErrSecretNotFound)

		_, err := uc.Get(ctx, "missing")
		assert.ErrorIs(t, err, secretsDomain.ErrSecretNotFound)
	})

	t.Run("Error_DekNotFound_MapsToDecryptionFailed", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)
		fake.FailDecrypt = keyring.ErrDecryptionFailed

		repo.EXPECT().GetByPath(ctx, "p").Return(&secretsDomain.Secret{
			DekID: uuid.New(),
		}, nil)

		_, err := uc.Get(ctx, "p")
		assert.ErrorIs(t, err, keyring.ErrDecryptionFailed)
	})

	t.Run("Error_KekNotFound_MapsToDecryptionFailed", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)
		fake.FailDecrypt = keyring.ErrDecryptionFailed

		repo.EXPECT().GetByPath(ctx, "p").Return(&secretsDomain.Secret{
			DekID: uuid.New(),
		}, nil)

		_, err := uc.Get(ctx, "p")
		assert.ErrorIs(t, err, keyring.ErrDecryptionFailed)
	})

	t.Run("Error_DecryptionFailed_GenericErrorWraps", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)
		fake.FailDecrypt = errors.New("AEAD tag mismatch")

		repo.EXPECT().GetByPath(ctx, "p").Return(&secretsDomain.Secret{
			DekID: uuid.New(),
		}, nil)

		_, err := uc.Get(ctx, "p")
		assert.ErrorIs(t, err, keyring.ErrDecryptionFailed)
	})
}

func TestSecretUseCase_GetByVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_GetSpecificVersion", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)

		plaintext := []byte("v7")
		env, err := fake.Encrypt(ctx, plaintext)
		require.NoError(t, err)

		stored := &secretsDomain.Secret{
			Path:       "p",
			Version:    7,
			DekID:      env.DekID,
			Ciphertext: env.Ciphertext,
			Nonce:      env.Nonce,
		}
		repo.EXPECT().GetByPathAndVersion(ctx, "p", uint(7)).Return(stored, nil)

		got, err := uc.GetByVersion(ctx, "p", 7)
		require.NoError(t, err)
		assert.Equal(t, plaintext, got.Plaintext)
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().
			GetByPathAndVersion(ctx, "p", uint(99)).
			Return(nil, secretsDomain.ErrSecretNotFound)

		_, err := uc.GetByVersion(ctx, "p", 99)
		assert.ErrorIs(t, err, secretsDomain.ErrSecretNotFound)
	})

	t.Run("Error_DecryptionFailed", func(t *testing.T) {
		t.Parallel()
		uc, fake, repo := newSecretUseCase(t, 1024)
		fake.FailDecrypt = errors.New("AEAD tag mismatch")
		repo.EXPECT().
			GetByPathAndVersion(ctx, "p", uint(1)).
			Return(&secretsDomain.Secret{DekID: uuid.New()}, nil)

		_, err := uc.GetByVersion(ctx, "p", 1)
		assert.ErrorIs(t, err, keyring.ErrDecryptionFailed)
	})
}

// =============================================================================
// Delete
// =============================================================================

func TestSecretUseCase_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_DeleteSecret", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().Delete(ctx, "p").Return(nil)

		assert.NoError(t, uc.Delete(ctx, "p"))
	})

	t.Run("Error_SecretNotFound", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().Delete(ctx, "p").Return(secretsDomain.ErrSecretNotFound)

		err := uc.Delete(ctx, "p")
		assert.ErrorIs(t, err, secretsDomain.ErrSecretNotFound)
	})

	t.Run("Error_DeleteFails", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		boom := errors.New("db error")
		repo.EXPECT().Delete(ctx, "p").Return(boom)

		assert.ErrorIs(t, uc.Delete(ctx, "p"), boom)
	})
}

// =============================================================================
// PurgeDeleted
// =============================================================================

func TestSecretUseCase_PurgeDeleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_PurgeDeletedSecrets", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().
			HardDelete(ctx, mock.MatchedBy(func(tm time.Time) bool {
				return !tm.IsZero()
			}), false).
			Return(int64(5), nil)

		n, err := uc.PurgeDeleted(ctx, 30, false)
		require.NoError(t, err)
		assert.EqualValues(t, 5, n)
	})

	t.Run("Success_DryRun", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().
			HardDelete(ctx, mock.Anything, true).
			Return(int64(3), nil)

		n, err := uc.PurgeDeleted(ctx, 30, true)
		require.NoError(t, err)
		assert.EqualValues(t, 3, n)
	})

	t.Run("Success_NoSecretsToDelete", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		repo.EXPECT().HardDelete(ctx, mock.Anything, false).Return(int64(0), nil)

		n, err := uc.PurgeDeleted(ctx, 30, false)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("Error_NegativeDays", func(t *testing.T) {
		t.Parallel()
		uc, _, _ := newSecretUseCase(t, 1024)
		_, err := uc.PurgeDeleted(ctx, -1, false)
		assert.Error(t, err)
	})

	t.Run("Error_RepositoryFails", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newSecretUseCase(t, 1024)
		boom := apperrors.Wrap(apperrors.ErrInvalidInput, "boom")
		repo.EXPECT().HardDelete(ctx, mock.Anything, false).Return(int64(0), boom)

		_, err := uc.PurgeDeleted(ctx, 30, false)
		assert.ErrorIs(t, err, boom)
	})
}
