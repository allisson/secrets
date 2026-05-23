package usecase_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/keyring"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	"github.com/allisson/secrets/internal/transit/usecase"
	"github.com/allisson/secrets/internal/transit/usecase/mocks"
)

// noopTxManager runs the function with no real transaction.
type noopTxManager struct{}

func (noopTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newTransitKeyUseCase(
	t *testing.T,
) (usecase.TransitKeyUseCase, *keyring.Fake, *mocks.MockTransitKeyRepository) {
	t.Helper()
	fake := keyring.NewFake()
	repo := mocks.NewMockTransitKeyRepository(t)
	uc := usecase.NewTransitKeyUseCase(noopTxManager{}, repo, fake)
	return uc, fake, repo
}

// allocateDekForTest seeds the keyring Fake with a DekID and returns it.
func allocateDekForTest(t *testing.T, fake *keyring.Fake) uuid.UUID {
	t.Helper()
	handle, err := fake.AllocateDek(context.Background(), keyring.AESGCM)
	require.NoError(t, err)
	return handle.DekID
}

func TestTransitKeyUseCase_Create(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTransitKeyUseCase(t)
		repo.EXPECT().
			GetByNameAndVersion(ctx, "k", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound)
		repo.EXPECT().
			Create(ctx, mock.MatchedBy(func(k *transitDomain.TransitKey) bool {
				return k.Name == "k" && k.Version == 1
			})).
			Return(nil)

		key, err := uc.Create(ctx, "k", cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.Equal(t, "k", key.Name)
		assert.EqualValues(t, 1, key.Version)
		assert.NotEqual(t, uuid.Nil, key.DekID)
	})

	t.Run("Error_AlreadyExists", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTransitKeyUseCase(t)
		repo.EXPECT().
			GetByNameAndVersion(ctx, "dup", uint(1)).
			Return(&transitDomain.TransitKey{}, nil)

		_, err := uc.Create(ctx, "dup", cryptoDomain.AESGCM)
		assert.ErrorIs(t, err, transitDomain.ErrTransitKeyAlreadyExists)
	})
}

func TestTransitKeyUseCase_Rotate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Success_IncrementsVersion", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTransitKeyUseCase(t)
		repo.EXPECT().GetByName(ctx, "k").Return(&transitDomain.TransitKey{
			Name:    "k",
			Version: 2,
		}, nil)
		repo.EXPECT().
			Create(ctx, mock.MatchedBy(func(k *transitDomain.TransitKey) bool {
				return k.Version == 3
			})).
			Return(nil)

		key, err := uc.Rotate(ctx, "k", cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.EqualValues(t, 3, key.Version)
	})

	t.Run("Success_CreatesFirstVersionWhenAbsent", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTransitKeyUseCase(t)
		repo.EXPECT().GetByName(ctx, "new").Return(nil, transitDomain.ErrTransitKeyNotFound)
		repo.EXPECT().
			GetByNameAndVersion(ctx, "new", uint(1)).
			Return(nil, transitDomain.ErrTransitKeyNotFound)
		repo.EXPECT().
			Create(ctx, mock.MatchedBy(func(k *transitDomain.TransitKey) bool {
				return k.Version == 1
			})).
			Return(nil)

		key, err := uc.Rotate(ctx, "new", cryptoDomain.AESGCM)
		require.NoError(t, err)
		assert.EqualValues(t, 1, key.Version)
	})
}

func TestTransitKeyUseCase_EncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	uc, fake, repo := newTransitKeyUseCase(t)
	dekID := allocateDekForTest(t, fake)

	transitKey := &transitDomain.TransitKey{
		Name:    "k",
		Version: 1,
		DekID:   dekID,
	}
	plaintext := []byte("payload")
	aad := []byte("context")

	repo.EXPECT().GetByName(ctx, "k").Return(transitKey, nil)

	encBlob, err := uc.Encrypt(ctx, "k", plaintext, aad)
	require.NoError(t, err)
	require.NotEmpty(t, encBlob.Ciphertext)

	// Roundtrip via the wire format.
	wire := fmt.Sprintf("%d:%s", encBlob.Version, base64.StdEncoding.EncodeToString(encBlob.Ciphertext))

	repo.EXPECT().GetByNameAndVersion(ctx, "k", uint(1)).Return(transitKey, nil)

	decBlob, err := uc.Decrypt(ctx, "k", wire, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decBlob.Plaintext)
}

func TestTransitKeyUseCase_Decrypt_BadFormat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, _, _ := newTransitKeyUseCase(t)

	_, err := uc.Decrypt(ctx, "k", "not-a-valid-wire-format", nil)
	assert.Error(t, err)
}

func TestTransitKeyUseCase_Decrypt_CiphertextTooShort(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, _, repo := newTransitKeyUseCase(t)

	repo.EXPECT().
		GetByNameAndVersion(ctx, "k", uint(1)).
		Return(&transitDomain.TransitKey{Name: "k", Version: 1, DekID: uuid.New()}, nil)

	// 5 bytes < 12-byte nonce
	wire := fmt.Sprintf("1:%s", base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4, 5}))
	_, err := uc.Decrypt(ctx, "k", wire, nil)
	assert.ErrorIs(t, err, cryptoDomain.ErrDecryptionFailed)
}

func TestTransitKeyUseCase_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, _, repo := newTransitKeyUseCase(t)

	want := &transitDomain.TransitKey{Name: "k", Version: 1}
	repo.EXPECT().GetTransitKey(ctx, "k", uint(0)).Return(want, cryptoDomain.AESGCM, nil)

	got, alg, err := uc.Get(ctx, "k", 0)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, cryptoDomain.AESGCM, alg)
}

func TestTransitKeyUseCase_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, _, repo := newTransitKeyUseCase(t)

	repo.EXPECT().Delete(ctx, "k").Return(nil)
	assert.NoError(t, uc.Delete(ctx, "k"))
}

func TestTransitKeyUseCase_PurgeDeleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Error_NegativeDays", func(t *testing.T) {
		t.Parallel()
		uc, _, _ := newTransitKeyUseCase(t)
		_, err := uc.PurgeDeleted(ctx, -1, false)
		assert.Error(t, err)
	})

	t.Run("Success_DryRun", func(t *testing.T) {
		t.Parallel()
		uc, _, repo := newTransitKeyUseCase(t)
		repo.EXPECT().HardDelete(ctx, mock.Anything, true).Return(int64(2), nil)

		n, err := uc.PurgeDeleted(ctx, 30, true)
		require.NoError(t, err)
		assert.EqualValues(t, 2, n)
	})
}
