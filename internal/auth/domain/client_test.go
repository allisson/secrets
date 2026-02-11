package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// createTestClient creates a Client instance with the given policies for testing.
func createTestClient(policies []PolicyDocument) *Client {
	return &Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		Policies:  policies,
		CreatedAt: time.Now(),
	}
}

func TestClient_IsAllowed_WildcardPatterns(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Success_WildcardMatchesAnyPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "*",
					Capabilities: []Capability{ReadCapability, WriteCapability},
				},
			}),
			path:       "any/path/here",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Failure_WildcardWithWrongCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "any/path/here",
			capability: WriteCapability,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsAllowed_PrefixPatterns(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Success_PrefixMatchesNestedPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_PrefixMatchesMultipleSubPaths",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app1/db/password",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_NestedPrefixMatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_DeepNestedPrefixMatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app/db/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_PrefixMatchesMultipleLevels",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/api/key",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Failure_PrefixDoesNotMatchExactPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name: "Failure_PrefixWithWrongCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: WriteCapability,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsAllowed_ExactMatches(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Success_ExactMatchSimplePath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_ExactMatchNestedPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_ExactMatchWithCorrectCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app/db/password",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Failure_ExactMatchDoesNotMatchPrefix",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name: "Failure_ExactMatchWithWrongCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret",
			capability: WriteCapability,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsAllowed_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Failure_EmptyPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name: "Failure_EmptyCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app",
			capability: "",
			expected:   false,
		},
		{
			name: "Failure_EmptyPathAndCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "",
			capability: "",
			expected:   false,
		},
		{
			name: "Failure_NoMatchingPolicy",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "config/app",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name: "Failure_PathMatchesButCapabilityMissing",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app",
			capability: DeleteCapability,
			expected:   false,
		},
		{
			name:       "Failure_EmptyPoliciesList",
			client:     createTestClient([]PolicyDocument{}),
			path:       "secret/app",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name:       "Failure_NilPoliciesList",
			client:     createTestClient(nil),
			path:       "secret/app",
			capability: ReadCapability,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsAllowed_MultiplePolicies(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Success_FirstMatchingPolicyWins",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "secret/app/*",
					Capabilities: []Capability{WriteCapability},
				},
			}),
			path:       "secret/app/db",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_SecondPolicyMatches",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "config/*",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "secret/*",
					Capabilities: []Capability{WriteCapability},
				},
			}),
			path:       "secret/app",
			capability: WriteCapability,
			expected:   true,
		},
		{
			name: "Success_MultipleCapabilitiesInSinglePolicy",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability, WriteCapability, DeleteCapability},
				},
			}),
			path:       "secret/app",
			capability: DeleteCapability,
			expected:   true,
		},
		{
			name: "Failure_MultiplePoliciesNoneMatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "config/*",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "secret/*",
					Capabilities: []Capability{WriteCapability},
				},
			}),
			path:       "data/app",
			capability: ReadCapability,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsAllowed_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Failure_PathCaseMismatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "Secret",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name: "Failure_NestedPathCaseMismatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "Secret/App",
			capability: ReadCapability,
			expected:   false,
		},
		{
			name: "Success_ExactCaseMatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/app",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app",
			capability: ReadCapability,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_IsAllowed_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
	}{
		{
			name: "Success_ReadOnlyAccess",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "Success_AdminAccess",
			client: createTestClient([]PolicyDocument{
				{
					Path: "*",
					Capabilities: []Capability{
						ReadCapability,
						WriteCapability,
						DeleteCapability,
						EncryptCapability,
						DecryptCapability,
						RotateCapability,
					},
				},
			}),
			path:       "any/path/anywhere",
			capability: RotateCapability,
			expected:   true,
		},
		{
			name: "Success_MultiplePathsWithDifferentCapabilities",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "config/*",
					Capabilities: []Capability{ReadCapability, WriteCapability},
				},
			}),
			path:       "config/app/settings",
			capability: WriteCapability,
			expected:   true,
		},
		{
			name: "Failure_WriteAttemptOnReadOnlyPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/app/db/password",
			capability: WriteCapability,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}
