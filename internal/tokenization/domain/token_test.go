package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToken_IsExpired(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		token     *Token
		expectExp bool
	}{
		{
			name: "NoExpiration_NotExpired",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: nil,
			},
			expectExp: false,
		},
		{
			name: "FutureExpiration_NotExpired",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: &future,
			},
			expectExp: false,
		},
		{
			name: "PastExpiration_Expired",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: &past,
			},
			expectExp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsExpired()
			assert.Equal(t, tt.expectExp, result)
		})
	}
}

func TestToken_IsRevoked(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name       string
		token      *Token
		expectRevo bool
	}{
		{
			name: "NotRevoked",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				RevokedAt: nil,
			},
			expectRevo: false,
		},
		{
			name: "Revoked",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				RevokedAt: &now,
			},
			expectRevo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsRevoked()
			assert.Equal(t, tt.expectRevo, result)
		})
	}
}

func TestToken_IsValid(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name        string
		token       *Token
		expectValid bool
	}{
		{
			name: "ValidToken_NotExpiredNotRevoked",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: nil,
				RevokedAt: nil,
			},
			expectValid: true,
		},
		{
			name: "ValidToken_FutureExpirationNotRevoked",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: &future,
				RevokedAt: nil,
			},
			expectValid: true,
		},
		{
			name: "InvalidToken_Expired",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: &past,
				RevokedAt: nil,
			},
			expectValid: false,
		},
		{
			name: "InvalidToken_Revoked",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: nil,
				RevokedAt: &now,
			},
			expectValid: false,
		},
		{
			name: "InvalidToken_ExpiredAndRevoked",
			token: &Token{
				ID:        uuid.Must(uuid.NewV7()),
				ExpiresAt: &past,
				RevokedAt: &now,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsValid()
			assert.Equal(t, tt.expectValid, result)
		})
	}
}
