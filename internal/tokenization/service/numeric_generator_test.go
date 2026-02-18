package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumericGenerator_Generate(t *testing.T) {
	gen := NewNumericGenerator()

	tests := []struct {
		name          string
		length        int
		expectError   bool
		validateToken bool
	}{
		{
			name:          "Success_Length1",
			length:        1,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Success_Length16",
			length:        16,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Success_Length32",
			length:        32,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Success_Length255",
			length:        255,
			expectError:   false,
			validateToken: true,
		},
		{
			name:        "Error_LengthZero",
			length:      0,
			expectError: true,
		},
		{
			name:        "Error_NegativeLength",
			length:      -1,
			expectError: true,
		},
		{
			name:        "Error_LengthTooLarge",
			length:      256,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := gen.Generate(tt.length)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, token, tt.length)

			if tt.validateToken {
				// Verify all characters are digits
				for _, c := range token {
					assert.True(t, c >= '0' && c <= '9', "character %c is not a digit", c)
				}
			}
		})
	}
}

func TestNumericGenerator_Validate(t *testing.T) {
	gen := NewNumericGenerator()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid_SingleDigit",
			token:       "5",
			expectError: false,
		},
		{
			name:        "Valid_MultipleDigits",
			token:       "1234567890",
			expectError: false,
		},
		{
			name:        "Valid_LeadingZeros",
			token:       "0001234",
			expectError: false,
		},
		{
			name:        "Invalid_Empty",
			token:       "",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsLetters",
			token:       "123abc456",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsSpecialChars",
			token:       "123-456",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsSpaces",
			token:       "123 456",
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

func TestNumericGenerator_Randomness(t *testing.T) {
	gen := NewNumericGenerator()

	// Generate multiple tokens and ensure they're different (probabilistic test)
	tokens := make(map[string]bool)
	length := 16

	for i := 0; i < 100; i++ {
		token, err := gen.Generate(length)
		assert.NoError(t, err)
		tokens[token] = true
	}

	// With 16-digit tokens, we should have 100 unique values
	assert.Equal(t, 100, len(tokens), "expected all tokens to be unique")
}
