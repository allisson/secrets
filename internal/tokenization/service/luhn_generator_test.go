package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLuhnGenerator_Generate(t *testing.T) {
	gen := NewLuhnGenerator()

	tests := []struct {
		name          string
		length        int
		expectError   bool
		validateToken bool
	}{
		{
			name:          "Success_Length2",
			length:        2,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Success_Length16_CreditCard",
			length:        16,
			expectError:   false,
			validateToken: true,
		},
		{
			name:          "Success_Length19_AmexCard",
			length:        19,
			expectError:   false,
			validateToken: true,
		},
		{
			name:        "Error_LengthOne",
			length:      1,
			expectError: true,
		},
		{
			name:        "Error_LengthZero",
			length:      0,
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
				// Verify it's numeric
				for _, c := range token {
					assert.True(t, c >= '0' && c <= '9', "character %c is not a digit", c)
				}

				// Verify it passes Luhn validation
				err := gen.Validate(token)
				assert.NoError(t, err, "generated token should pass Luhn validation")
			}
		})
	}
}

func TestLuhnGenerator_Validate(t *testing.T) {
	gen := NewLuhnGenerator()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid_KnownLuhnNumber_79927398713",
			token:       "79927398713",
			expectError: false,
		},
		{
			name:        "Valid_KnownLuhnNumber_4532015112830366",
			token:       "4532015112830366",
			expectError: false,
		},
		{
			name:        "Valid_SimpleCase_18",
			token:       "18",
			expectError: false,
		},
		{
			name:        "Invalid_KnownInvalidNumber",
			token:       "4532015112830367",
			expectError: true,
		},
		{
			name:        "Invalid_Empty",
			token:       "",
			expectError: true,
		},
		{
			name:        "Invalid_SingleDigit",
			token:       "5",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsLetters",
			token:       "453201511283036a",
			expectError: true,
		},
		{
			name:        "Invalid_ContainsSpaces",
			token:       "4532 0151 1283 0366",
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

func TestCalculateLuhnCheckDigit(t *testing.T) {
	tests := []struct {
		name          string
		digits        []int
		expectedDigit int
	}{
		{
			name:          "SimpleCase_1",
			digits:        []int{1},
			expectedDigit: 8,
		},
		{
			name:          "SimpleCase_79927398713",
			digits:        []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1},
			expectedDigit: 3,
		},
		{
			name:          "CreditCard_453201511283036",
			digits:        []int{4, 5, 3, 2, 0, 1, 5, 1, 1, 2, 8, 3, 0, 3, 6},
			expectedDigit: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkDigit := calculateLuhnCheckDigit(tt.digits)
			assert.Equal(t, tt.expectedDigit, checkDigit)
		})
	}
}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name     string
		digits   []int
		expected bool
	}{
		{
			name:     "Valid_18",
			digits:   []int{1, 8},
			expected: true,
		},
		{
			name:     "Valid_79927398713",
			digits:   []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1, 3},
			expected: true,
		},
		{
			name:     "Valid_4532015112830366",
			digits:   []int{4, 5, 3, 2, 0, 1, 5, 1, 1, 2, 8, 3, 0, 3, 6, 6},
			expected: true,
		},
		{
			name:     "Invalid_17",
			digits:   []int{1, 7},
			expected: false,
		},
		{
			name:     "Invalid_79927398712",
			digits:   []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1, 2},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateLuhn(tt.digits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLuhnGenerator_Randomness(t *testing.T) {
	gen := NewLuhnGenerator()

	// Generate multiple tokens and ensure they're different and all pass Luhn validation
	tokens := make(map[string]bool)
	length := 16

	for i := 0; i < 100; i++ {
		token, err := gen.Generate(length)
		assert.NoError(t, err)

		// Verify Luhn compliance
		err = gen.Validate(token)
		assert.NoError(t, err, "token %s should pass Luhn validation", token)

		tokens[token] = true
	}

	// With 16-digit tokens, we should have 100 unique values
	assert.Equal(t, 100, len(tokens), "expected all tokens to be unique")
}
