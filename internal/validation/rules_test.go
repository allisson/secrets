package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotBlank(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "valid string",
			input:     "validstring",
			shouldErr: false,
		},
		{
			name:      "only spaces",
			input:     "   ",
			shouldErr: true,
		},
		{
			name:      "only tabs",
			input:     "\t\t",
			shouldErr: true,
		},
		{
			name:      "only newlines",
			input:     "\n\n",
			shouldErr: true,
		},
		{
			name:      "mixed whitespace",
			input:     " \t\n ",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NotBlank.Validate(tt.input)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWrapValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error returns nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "wraps validation error",
			err:      assert.AnError,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapValidationError(tt.err)
			if tt.expected {
				assert.Error(t, result)
				assert.Contains(t, result.Error(), "invalid input")
			} else {
				assert.NoError(t, result)
			}
		})
	}
}
