package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

type mockFormatter struct {
	text string
	json string
}

func (m *mockFormatter) ToText() string { return m.text }
func (m *mockFormatter) ToJSON() string { return m.json }

func TestWriteOutput(t *testing.T) {
	data := &mockFormatter{
		text: "text output",
		json: `{"output": "json"}`,
	}

	t.Run("text", func(t *testing.T) {
		var out bytes.Buffer
		WriteOutput(&out, "text", data)
		require.Equal(t, "text output\n", out.String())
	})

	t.Run("json", func(t *testing.T) {
		var out bytes.Buffer
		WriteOutput(&out, "json", data)
		require.Equal(t, "{\"output\": \"json\"}\n", out.String())
	})
}

func TestParseAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected cryptoDomain.Algorithm
		wantErr  bool
	}{
		{"aes-gcm", cryptoDomain.AESGCM, false},
		{"chacha20-poly1305", cryptoDomain.ChaCha20, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseAlgorithm(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestParseFormatType(t *testing.T) {
	tests := []struct {
		input    string
		expected tokenizationDomain.FormatType
		wantErr  bool
	}{
		{"uuid", tokenizationDomain.FormatUUID, false},
		{"numeric", tokenizationDomain.FormatNumeric, false},
		{"luhn-preserving", tokenizationDomain.FormatLuhnPreserving, false},
		{"alphanumeric", tokenizationDomain.FormatAlphanumeric, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormatType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}
