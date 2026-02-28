package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestKekChain(t *testing.T) {
	kek1 := &Kek{ID: uuid.New(), Key: []byte("key1-data-1234567890123456789012")}
	kek2 := &Kek{ID: uuid.New(), Key: []byte("key2-data-1234567890123456789012")}

	t.Run("NewKekChain and ActiveKekID", func(t *testing.T) {
		keks := []*Kek{kek1, kek2}
		kc := NewKekChain(keks)
		assert.Equal(t, kek1.ID, kc.ActiveKekID())
	})

	t.Run("Get KEK", func(t *testing.T) {
		kc := NewKekChain([]*Kek{kek1, kek2})

		k, ok := kc.Get(kek1.ID)
		assert.True(t, ok)
		assert.Equal(t, kek1, k)

		k, ok = kc.Get(kek2.ID)
		assert.True(t, ok)
		assert.Equal(t, kek2, k)

		k, ok = kc.Get(uuid.New())
		assert.False(t, ok)
		assert.Nil(t, k)
	})

	t.Run("Close zeros all keys", func(t *testing.T) {
		k1Data := make([]byte, 32)
		copy(k1Data, []byte("key1-data-1234567890123456789012"))
		k2Data := make([]byte, 32)
		copy(k2Data, []byte("key2-data-1234567890123456789012"))

		k1 := &Kek{ID: uuid.New(), Key: k1Data}
		k2 := &Kek{ID: uuid.New(), Key: k2Data}

		kc := NewKekChain([]*Kek{k1, k2})
		kc.Close()

		assert.Equal(t, uuid.Nil, kc.ActiveKekID())
		_, ok := kc.Get(k1.ID)
		assert.False(t, ok)

		expectedZero := make([]byte, 32)
		assert.Equal(t, expectedZero, k1.Key)
		assert.Equal(t, expectedZero, k2.Key)
	})

	t.Run("NewKekChain with empty slice", func(t *testing.T) {
		kc := NewKekChain([]*Kek{})
		assert.Equal(t, uuid.Nil, kc.ActiveKekID())
		_, ok := kc.Get(uuid.New())
		assert.False(t, ok)
	})
}
