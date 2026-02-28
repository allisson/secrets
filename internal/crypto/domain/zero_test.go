package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZero(t *testing.T) {
	t.Run("zero non-empty slice", func(t *testing.T) {
		b := []byte{1, 2, 3, 4, 5}
		Zero(b)
		for _, v := range b {
			assert.Equal(t, byte(0), v)
		}
	})

	t.Run("zero empty slice", func(t *testing.T) {
		b := []byte{}
		Zero(b)
		assert.Equal(t, 0, len(b))
	})

	t.Run("zero nil slice", func(t *testing.T) {
		var b []byte
		assert.NotPanics(t, func() { Zero(b) })
	})

	t.Run("zero large slice", func(t *testing.T) {
		b := make([]byte, 1024)
		for i := range b {
			b[i] = byte(i % 256)
		}
		Zero(b)
		for _, v := range b {
			assert.Equal(t, byte(0), v)
		}
	})
}
