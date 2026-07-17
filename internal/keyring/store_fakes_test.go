package keyring

import (
	"context"
	"crypto/rand"
	"sort"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// newMemDekStore returns an empty in-memory dekStore (type defined in impl_test.go).
func newMemDekStore() *memDekStore {
	return &memDekStore{deks: make(map[uuid.UUID]*dek)}
}

// memKekStore is an in-memory kekStore for unit tests. list returns rows ordered
// by version descending, matching the production repository's ORDER BY.
type memKekStore struct {
	mu   sync.Mutex
	keks map[string]*kek
}

func newMemKekStore() *memKekStore {
	return &memKekStore{keks: make(map[string]*kek)}
}

func (m *memKekStore) create(_ context.Context, k *kek) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *k
	m.keks[k.id.String()] = &cp
	return nil
}

func (m *memKekStore) update(_ context.Context, k *kek) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.keks[k.id.String()]; !ok {
		return ErrKekNotFound
	}
	cp := *k
	m.keks[k.id.String()] = &cp
	return nil
}

func (m *memKekStore) list(_ context.Context) ([]*kek, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*kek, 0, len(m.keks))
	for _, k := range m.keks {
		cp := *k
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version > out[j].version })
	return out, nil
}

// fakeTxManager runs the callback inline with no real transaction, so use-case
// orchestration can be tested without a database.
type fakeTxManager struct{}

func (fakeTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// newTestMasterKeyChain returns a chain holding a single random 32-byte master
// key whose ID is activeID, plus the master key itself for wrapping KEKs.
func newTestMasterKeyChain(t *testing.T, activeID string) (*MasterKeyChain, *masterKey) {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	mk := &masterKey{ID: activeID, key: key}
	chain := NewMasterKeyChain(activeID)
	chain.keys.Store(activeID, mk)
	return chain, mk
}
