package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecret_IsDeleted(t *testing.T) {
	t.Run("Not deleted", func(t *testing.T) {
		s := &Secret{
			DeletedAt: nil,
		}
		assert.False(t, s.IsDeleted())
	})

	t.Run("Deleted", func(t *testing.T) {
		now := time.Now()
		s := &Secret{
			DeletedAt: &now,
		}
		assert.True(t, s.IsDeleted())
	})
}
