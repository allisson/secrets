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

func TestSecret_Validate(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid path",
			path:    "production/database/password",
			wantErr: false,
		},
		{
			name:    "Valid path with hyphens and underscores",
			path:    "app-1/prod/db_secret",
			wantErr: false,
		},
		{
			name:    "Invalid path - too long",
			path:    string(make([]byte, 256)),
			wantErr: true,
		},
		{
			name:    "Invalid path - disallowed characters",
			path:    "app 1/prod",
			wantErr: true,
		},
		{
			name:    "Invalid path - special characters",
			path:    "app@1/prod",
			wantErr: true,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Secret{
				Path: tt.path,
			}
			err := s.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidSecretPath)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
