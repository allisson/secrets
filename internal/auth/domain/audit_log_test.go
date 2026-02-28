package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditLog_HasValidSignature(t *testing.T) {
	kekID := uuid.New()

	tests := []struct {
		name     string
		log      AuditLog
		expected bool
	}{
		{
			name: "Valid signature",
			log: AuditLog{
				IsSigned:  true,
				KekID:     &kekID,
				Signature: make([]byte, 32),
			},
			expected: true,
		},
		{
			name: "Not signed",
			log: AuditLog{
				IsSigned:  false,
				KekID:     &kekID,
				Signature: make([]byte, 32),
			},
			expected: false,
		},
		{
			name: "Nil KekID",
			log: AuditLog{
				IsSigned:  true,
				KekID:     nil,
				Signature: make([]byte, 32),
			},
			expected: false,
		},
		{
			name: "Invalid signature length",
			log: AuditLog{
				IsSigned:  true,
				KekID:     &kekID,
				Signature: make([]byte, 31),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.log.HasValidSignature())
		})
	}
}

func TestAuditLog_IsLegacy(t *testing.T) {
	kekID := uuid.New()

	tests := []struct {
		name     string
		log      AuditLog
		expected bool
	}{
		{
			name: "Legacy log",
			log: AuditLog{
				IsSigned:  false,
				KekID:     nil,
				Signature: nil,
			},
			expected: true,
		},
		{
			name: "Signed log",
			log: AuditLog{
				IsSigned:  true,
				KekID:     &kekID,
				Signature: make([]byte, 32),
			},
			expected: false,
		},
		{
			name: "Mixed state (not legacy)",
			log: AuditLog{
				IsSigned:  false,
				KekID:     &kekID,
				Signature: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.log.IsLegacy())
		})
	}
}
