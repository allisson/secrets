package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidCapabilities(t *testing.T) {
	caps := ValidCapabilities()
	assert.Len(t, caps, 6)
	assert.Contains(t, caps, ReadCapability)
	assert.Contains(t, caps, WriteCapability)
	assert.Contains(t, caps, DeleteCapability)
	assert.Contains(t, caps, EncryptCapability)
	assert.Contains(t, caps, DecryptCapability)
	assert.Contains(t, caps, RotateCapability)
}

func TestIsValidCapability(t *testing.T) {
	tests := []struct {
		name     string
		cap      Capability
		expected bool
	}{
		{"read is valid", ReadCapability, true},
		{"write is valid", WriteCapability, true},
		{"delete is valid", DeleteCapability, true},
		{"encrypt is valid", EncryptCapability, true},
		{"decrypt is valid", DecryptCapability, true},
		{"rotate is valid", RotateCapability, true},
		{"invalid is not valid", Capability("invalid"), false},
		{"empty is not valid", Capability(""), false},
		{"case sensitive is not valid", Capability("READ"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidCapability(tt.cap))
		})
	}
}
