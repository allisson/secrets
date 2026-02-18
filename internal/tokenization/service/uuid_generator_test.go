package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUIDGenerator_Generate(t *testing.T) {
	gen := NewUUIDGenerator()

	t.Run("Success_GeneratesValidUUID", func(t *testing.T) {
		token, err := gen.Generate(0) // length parameter is ignored
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate it's a proper UUID
		_, err = uuid.Parse(token)
		assert.NoError(t, err)
	})

	t.Run("Success_GeneratesUniqueTokens", func(t *testing.T) {
		token1, err1 := gen.Generate(0)
		token2, err2 := gen.Generate(0)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, token1, token2, "tokens should be unique")
	})
}

func TestUUIDGenerator_Validate(t *testing.T) {
	gen := NewUUIDGenerator()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid_UUIDv4",
			token:       "550e8400-e29b-41d4-a716-446655440000",
			expectError: false,
		},
		{
			name:        "Valid_UUIDv7",
			token:       uuid.Must(uuid.NewV7()).String(),
			expectError: false,
		},
		{
			name:        "Invalid_NotUUID",
			token:       "not-a-uuid",
			expectError: true,
		},
		{
			name:        "Invalid_Empty",
			token:       "",
			expectError: true,
		},
		{
			name:        "Invalid_PartialUUID",
			token:       "550e8400-e29b-41d4",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.Validate(tt.token)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
