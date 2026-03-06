package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

func TestParseCapabilities(t *testing.T) {
	tests := []struct {
		input    string
		expected []authDomain.Capability
		wantErr  bool
	}{
		{"read,write", []authDomain.Capability{"read", "write"}, false},
		{"read , write ", []authDomain.Capability{"read", "write"}, false},
		{"read,invalid", nil, true},
		{"READ", nil, true},
		{"", nil, true},
		{" , ", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCapabilities(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestPromptForPolicies(t *testing.T) {
	t.Run("success-single-policy", func(t *testing.T) {
		input := "secret/*\nread,write\nn\n"
		var output bytes.Buffer
		policies, err := PromptForPolicies(strings.NewReader(input), &output)

		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "secret/*", policies[0].Path)
		require.Equal(t, []authDomain.Capability{"read", "write"}, policies[0].Capabilities)
	})

	t.Run("success-multiple-policies", func(t *testing.T) {
		input := "secret/*\nread\ny\nother/*\nwrite\nn\n"
		var output bytes.Buffer
		policies, err := PromptForPolicies(strings.NewReader(input), &output)

		require.NoError(t, err)
		require.Len(t, policies, 2)
		require.Equal(t, "secret/*", policies[0].Path)
		require.Equal(t, "other/*", policies[1].Path)
	})

	t.Run("empty-path", func(t *testing.T) {
		input := "\n"
		var output bytes.Buffer
		_, err := PromptForPolicies(strings.NewReader(input), &output)

		require.Error(t, err)
		require.Contains(t, err.Error(), "path cannot be empty")
	})

	t.Run("invalid-capability", func(t *testing.T) {
		input := "secret/*\ninvalid\n"
		var output bytes.Buffer
		_, err := PromptForPolicies(strings.NewReader(input), &output)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid capability: 'invalid'")
	})
}
