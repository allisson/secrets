package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase64(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid base64",
			input:     "SGVsbG8gV29ybGQ=",
			shouldErr: false,
		},
		{
			name:      "empty string",
			input:     "",
			shouldErr: false,
		},
		{
			name:      "invalid base64 format",
			input:     "This is not base64!",
			shouldErr: true,
			errMsg:    "must be valid base64-encoded data",
		},
		{
			name:      "invalid type (int)",
			input:     123,
			shouldErr: true,
			errMsg:    "must be a string",
		},
		{
			name:      "invalid type (nil)",
			input:     nil,
			shouldErr: true,
			errMsg:    "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Base64.Validate(tt.input)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
