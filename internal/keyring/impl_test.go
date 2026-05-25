package keyring

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memDekStore is an in-memory dekStore for unit tests.
type memDekStore struct {
	mu   sync.RWMutex
	deks map[uuid.UUID]*dek
}

func (m *memDekStore) create(_ context.Context, d *dek) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *d
	m.deks[d.id] = &cp
	return nil
}

func (m *memDekStore) get(_ context.Context, dekID uuid.UUID) (*dek, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.deks[dekID]
	if !ok {
		return nil, ErrDekNotFound
	}
	cp := *d
	return &cp, nil
}

func (m *memDekStore) update(_ context.Context, d *dek) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deks[d.id]; !ok {
		return ErrDekNotFound
	}
	cp := *d
	m.deks[d.id] = &cp
	return nil
}

func (m *memDekStore) getBatchNotKekID(_ context.Context, kekID uuid.UUID, limit int) ([]*dek, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*dek
	for _, d := range m.deks {
		if d.kekID != kekID {
			cp := *d
			result = append(result, &cp)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func newTestKek(t *testing.T, alg Algorithm) *kek {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return &kek{
		id:        uuid.Must(uuid.NewV7()),
		algorithm: alg,
		key:       key,
	}
}

func newTestKeyring(t *testing.T, alg Algorithm) *keyringImpl {
	t.Helper()
	kk := newTestKek(t, alg)
	aeadMgr := newAEADManager()
	return &keyringImpl{
		kekChain:     newKekChain([]*kek{kk}),
		dekStore:     &memDekStore{deks: make(map[uuid.UUID]*dek)},
		aeadManager:  aeadMgr,
		keyManager:   newKeyManager(aeadMgr),
		dekAlgorithm: alg,
	}
}

func TestImpl_Encrypt_Decrypt_RoundTrip(t *testing.T) {
	for _, alg := range []Algorithm{AESGCM, ChaCha20} {
		t.Run(string(alg), func(t *testing.T) {
			kr := newTestKeyring(t, alg)
			ctx := context.Background()
			plaintext := []byte("hello world")

			env, err := kr.Encrypt(ctx, plaintext)
			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, env.DekID)
			assert.NotEqual(t, plaintext, env.Ciphertext)

			got, err := kr.Decrypt(ctx, env)
			require.NoError(t, err)
			assert.Equal(t, plaintext, got)
		})
	}
}

func TestImpl_AllocateDek_EncryptWith_DecryptWith(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()

	handle, err := kr.AllocateDek(ctx, AESGCM)
	require.NoError(t, err)

	plaintext := []byte("payload")
	aad := []byte("additional-authenticated-data")

	ct, nonce, err := kr.EncryptWith(ctx, handle, plaintext, aad)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ct)
	assert.NotEmpty(t, nonce)

	got, err := kr.DecryptWith(ctx, handle, ct, nonce, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestImpl_DecryptWith_WrongAAD_Fails(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()

	handle, err := kr.AllocateDek(ctx, AESGCM)
	require.NoError(t, err)

	ct, nonce, err := kr.EncryptWith(ctx, handle, []byte("secret"), []byte("aad1"))
	require.NoError(t, err)

	_, err = kr.DecryptWith(ctx, handle, ct, nonce, []byte("aad2"))
	assert.Error(t, err)
}

// TestImpl_AEAD_CrossAlgorithm_Rejection verifies that a ciphertext produced by
// AES-256-GCM cannot be decrypted when the DEK is misidentified as ChaCha20-Poly1305.
func TestImpl_AEAD_CrossAlgorithm_Rejection(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()

	env, err := kr.Encrypt(ctx, []byte("aesgcm plaintext"))
	require.NoError(t, err)

	// Corrupt the stored DEK's algorithm — the raw ciphertext was sealed with
	// AES-GCM, so opening it with ChaCha20-Poly1305 must fail.
	store := kr.dekStore.(*memDekStore)
	store.mu.Lock()
	store.deks[env.DekID].algorithm = ChaCha20
	store.mu.Unlock()

	_, err = kr.Decrypt(ctx, env)
	assert.Error(t, err, "AES-GCM ciphertext must not be decryptable by ChaCha20-Poly1305")
}

func TestImpl_Rewrap_ReencryptsDEKUnderNewKEK(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()

	plaintext := []byte("rewrap me")
	env, err := kr.Encrypt(ctx, plaintext)
	require.NoError(t, err)

	oldKekID := kr.ActiveKekID()

	// Rotate: add a second KEK and make it active.
	kek2 := newTestKek(t, AESGCM)
	kr.kekChain.keys.Store(kek2.id, kek2)
	kr.kekChain.activeID = kek2.id

	require.NoError(t, kr.Rewrap(ctx, env.DekID))

	d, err := kr.dekStore.get(ctx, env.DekID)
	require.NoError(t, err)
	assert.Equal(t, kek2.id, d.kekID, "DEK must point to the new KEK after rewrap")
	assert.NotEqual(t, oldKekID, d.kekID)

	// Original envelope must remain decryptable.
	got, err := kr.Decrypt(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestImpl_Rewrap_AlreadyUnderActiveKEK_IsNoop(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()

	env, err := kr.Encrypt(ctx, []byte("already current"))
	require.NoError(t, err)

	// DEK is already under the active KEK — Rewrap must be a no-op.
	require.NoError(t, kr.Rewrap(ctx, env.DekID))

	got, err := kr.Decrypt(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, []byte("already current"), got)
}

func TestImpl_RewrapAll_Idempotency(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()

	const numDEKs = 3
	for i := range numDEKs {
		_, err := kr.Encrypt(ctx, []byte{byte(i)})
		require.NoError(t, err)
	}

	// Rotate to a second KEK.
	kek2 := newTestKek(t, AESGCM)
	kr.kekChain.keys.Store(kek2.id, kek2)
	kr.kekChain.activeID = kek2.id

	// First pass migrates all DEKs.
	count, err := kr.RewrapAll(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, numDEKs, count)

	// Second pass finds nothing to migrate.
	count, err = kr.RewrapAll(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestImpl_RewrapAll_InvalidBatchSize(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	ctx := context.Background()
	_, err := kr.RewrapAll(ctx, 0)
	assert.ErrorIs(t, err, errKeyringBadBatchSize)
}

func TestImpl_SignWithKey_VerifyWithKey_RoundTrip(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	data := []byte("audit log entry")

	sig, kekID, err := kr.SignWithKey(data)
	require.NoError(t, err)
	assert.Len(t, sig, 32)

	require.NoError(t, kr.VerifyWithKey(kekID, data, sig))
}

func TestImpl_VerifyWithKey_TamperedData(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)

	sig, kekID, err := kr.SignWithKey([]byte("original"))
	require.NoError(t, err)

	err = kr.VerifyWithKey(kekID, []byte("tampered"), sig)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestImpl_VerifyWithKey_UnknownKEK(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	err := kr.VerifyWithKey(uuid.New(), []byte("data"), make([]byte, 32))
	assert.ErrorIs(t, err, ErrKekNotFound)
}

func TestImpl_ActiveKekID_ReturnsNonNil(t *testing.T) {
	kr := newTestKeyring(t, AESGCM)
	assert.NotEqual(t, uuid.Nil, kr.ActiveKekID())
}

func TestZero_ClearsKeyMaterial(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5}
	Zero(key)
	assert.Equal(t, make([]byte, 5), key)
}
