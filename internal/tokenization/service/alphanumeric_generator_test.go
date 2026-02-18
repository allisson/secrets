package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlphanumericGenerator_Generate(t *testing.T) {
	gen := NewAlphanumericGenerator()

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
			name:          "Success_Length32",
			length:        32,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Success_Length64",
			length:        64,
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
				// Verify all characters are alphanumeric
				for _, c := range token {
					isValid := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
					assert.True(t, isValid, "character %c is not alphanumeric", c)
				}
			}
		})
	}
}

func TestAlphanumericGenerator_Validate(t *testing.T) {
	gen := NewAlphanumericGenerator()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid_Uppercase",
			token:       "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			expectError: false,
		},
		{
			name:        "Valid_Lowercase",
			token:       "abcdefghijklmnopqrstuvwxyz",
			expectError: false,
		},
		{
			name:        "Valid_Digits",
			token:       "0123456789",
			expectError: false,
		},
		//nolint:gosec // test token string
		{
			name:        "Valid_Mixed",
			token:       "aB3dE5fG7h",
			expectError: false,
		},
		{
			name:        "Invalid_Empty",
			token:       "",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsHyphen",
			token:       "abc-def",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsUnderscore",
			token:       "abc_def",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsSpaces",
			token:       "abc def",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsSpecialChars",
			token:       "abc@def!",
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

func TestAlphanumericGenerator_Randomness(t *testing.T) {
	gen := NewAlphanumericGenerator()

	// Generate multiple tokens and ensure they're different
	tokens := make(map[string]bool)
	length := 32

	for i := 0; i < 100; i++ {
		token, err := gen.Generate(length)
		assert.NoError(t, err)
		tokens[token] = true
	}

	// With 32-character alphanumeric tokens, we should have 100 unique values
	assert.Equal(t, 100, len(tokens), "expected all tokens to be unique")
}

func TestAlphanumericGenerator_CharacterDistribution(t *testing.T) {
	gen := NewAlphanumericGenerator()

	// Generate a large token to check character distribution (within limit)
	length := 255
	token, err := gen.Generate(length)
	assert.NoError(t, err)

	// Count character types
	uppercaseCount := 0
	lowercaseCount := 0
	digitCount := 0

	for _, c := range token {
		switch {
		case c >= 'A' && c <= 'Z':
			uppercaseCount++
		case c >= 'a' && c <= 'z':
			lowercaseCount++
		case c >= '0' && c <= '9':
			digitCount++
		}
	}

	// Verify we have a mix of all character types (probabilistic test)
	assert.Greater(t, uppercaseCount, 0, "should contain uppercase letters")
	assert.Greater(t, lowercaseCount, 0, "should contain lowercase letters")
	assert.Greater(t, digitCount, 0, "should contain digits")

	// Verify counts sum to total length (if we have any characters)
	if uppercaseCount+lowercaseCount+digitCount > 0 {
		assert.Equal(t, length, uppercaseCount+lowercaseCount+digitCount)
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		char     rune
		expected bool
	}{
		{name: "Uppercase_A", char: 'A', expected: true},
		{name: "Uppercase_Z", char: 'Z', expected: true},
		{name: "Lowercase_a", char: 'a', expected: true},
		{name: "Lowercase_z", char: 'z', expected: true},
		{name: "Digit_0", char: '0', expected: true},
		{name: "Digit_9", char: '9', expected: true},
		{name: "Space", char: ' ', expected: false},
		{name: "Hyphen", char: '-', expected: false},
		{name: "Underscore", char: '_', expected: false},
		{name: "At", char: '@', expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlphanumeric(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}
