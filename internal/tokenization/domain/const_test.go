package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatType_Validate(t *testing.T) {
	tests := []struct {
		name        string
		formatType  FormatType
		expectError bool
	}{
		{
			name:        "Valid_UUID",
			formatType:  FormatUUID,
			expectError: false,
		},
		{
			name:        "Valid_Numeric",
			formatType:  FormatNumeric,
			expectError: false,
		},
		{
			name:        "Valid_LuhnPreserving",
			formatType:  FormatLuhnPreserving,
			expectError: false,
		},
		{
			name:        "Valid_Alphanumeric",
			formatType:  FormatAlphanumeric,
			expectError: false,
		},
		{
			name:        "Invalid_UnknownFormat",
			formatType:  FormatType("unknown"),
			expectError: true,
		},
		{
			name:        "Invalid_EmptyString",
			formatType:  FormatType(""),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formatType.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatType_String(t *testing.T) {
	tests := []struct {
		name       string
		formatType FormatType
		expected   string
	}{
		{
			name:       "UUID",
			formatType: FormatUUID,
			expected:   "uuid",
		},
		{
			name:       "Numeric",
			formatType: FormatNumeric,
			expected:   "numeric",
		},
		{
			name:       "LuhnPreserving",
			formatType: FormatLuhnPreserving,
			expected:   "luhn-preserving",
		},
		{
			name:       "Alphanumeric",
			formatType: FormatAlphanumeric,
			expected:   "alphanumeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.formatType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
