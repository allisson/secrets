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
		{
			name: "EdgeCase_EmptySubpathAfterWildcard",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "secret/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "secret/",
			capability: ReadCapability,
			expected:   true,
		},
		{
			name: "EdgeCase_MidPathWildcard_TooFewSegments",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "/v1/keys/rotate",
			capability: RotateCapability,
			expected:   false,
		},
		{
			name: "EdgeCase_MidPathWildcard_TooManySegments",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "/v1/keys/payment/extra/rotate",
			capability: RotateCapability,
			expected:   false,
		},
		{
			name: "EdgeCase_MidPathWildcard_ExactMatch",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "/v1/keys/payment/rotate",
			capability: RotateCapability,
			expected:   true,
		},
		{
			name: "EdgeCase_MultipleWildcards",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/*/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/rotate",
			capability: RotateCapability,
			expected:   true,
		},
		{
			name: "EdgeCase_MultipleWildcards_MissingSegment",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/*/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "/v1/transit/keys/rotate",
			capability: RotateCapability,
			expected:   false,
		},
		{
			name: "EdgeCase_WildcardAtBeginning",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "anything/rotate",
			capability: RotateCapability,
			expected:   true,
		},
		{
			name: "EdgeCase_WildcardAtBeginning_TooManySegments",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "anything/extra/rotate",
			capability: RotateCapability,
			expected:   false,
		},
		{
			name: "EdgeCase_TrailingSlashInRequestPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/secrets/app/",
			capability: DecryptCapability,
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
func TestClient_IsAllowed_PolicyTemplates(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
		comment    string
	}{
		// Template #1: Read-only service
		{
			name: "PolicyTemplate_ReadOnlyService_AllowDecrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/secrets/app/db/password",
			capability: DecryptCapability,
			expected:   true,
			comment:    "Read-only service can decrypt secrets",
		},
		{
			name: "PolicyTemplate_ReadOnlyService_DenyEncrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/secrets/app/db/password",
			capability: EncryptCapability,
			expected:   false,
			comment:    "Read-only service cannot encrypt (write) secrets",
		},

		// Template #2: CI writer
		{
			name: "PolicyTemplate_CIWriter_AllowEncrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/secrets/ci/deploy/key",
			capability: EncryptCapability,
			expected:   true,
			comment:    "CI writer can encrypt (write) secrets",
		},
		{
			name: "PolicyTemplate_CIWriter_DenyDecrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/secrets/ci/deploy/key",
			capability: DecryptCapability,
			expected:   false,
			comment:    "CI writer cannot decrypt (read) secrets",
		},

		// Template #3: Transit encrypt-only service
		{
			name: "PolicyTemplate_TransitEncryptOnly_AllowEncrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/payment/encrypt",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/encrypt",
			capability: EncryptCapability,
			expected:   true,
			comment:    "Transit encrypt-only can encrypt with specific key",
		},
		{
			name: "PolicyTemplate_TransitEncryptOnly_DenyDecrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/payment/encrypt",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/decrypt",
			capability: DecryptCapability,
			expected:   false,
			comment:    "Transit encrypt-only cannot decrypt",
		},
		{
			name: "PolicyTemplate_TransitEncryptOnly_DenyDifferentKey",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/payment/encrypt",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/transit/keys/user/encrypt",
			capability: EncryptCapability,
			expected:   false,
			comment:    "Transit encrypt-only scoped to specific key name",
		},

		// Template #4: Transit decrypt-only service
		{
			name: "PolicyTemplate_TransitDecryptOnly_AllowDecrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/payment/decrypt",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/decrypt",
			capability: DecryptCapability,
			expected:   true,
			comment:    "Transit decrypt-only can decrypt with specific key",
		},
		{
			name: "PolicyTemplate_TransitDecryptOnly_DenyEncrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/payment/decrypt",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/encrypt",
			capability: EncryptCapability,
			expected:   false,
			comment:    "Transit decrypt-only cannot encrypt",
		},

		// Template #5: Audit log reader
		{
			name: "PolicyTemplate_AuditLogReader_AllowRead",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/audit-logs",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "/v1/audit-logs",
			capability: ReadCapability,
			expected:   true,
			comment:    "Audit log reader can read audit logs",
		},
		{
			name: "PolicyTemplate_AuditLogReader_DenyWrite",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/audit-logs",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "/v1/audit-logs",
			capability: WriteCapability,
			expected:   false,
			comment:    "Audit log reader cannot write",
		},
		{
			name: "PolicyTemplate_AuditLogReader_DenyNestedPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/audit-logs",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "/v1/audit-logs/123",
			capability: ReadCapability,
			expected:   false,
			comment:    "Exact match only - no nested paths",
		},

		// Template #6: Break-glass admin (already covered in RealWorldScenarios, but adding for completeness)
		{
			name: "PolicyTemplate_BreakGlassAdmin_AllowSecretsDecrypt",
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
			path:       "/v1/secrets/critical/prod",
			capability: DecryptCapability,
			expected:   true,
			comment:    "Break-glass admin has full access",
		},
		{
			name: "PolicyTemplate_BreakGlassAdmin_AllowTransitRotate",
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
			path:       "/v1/transit/keys/payment/rotate",
			capability: RotateCapability,
			expected:   true,
			comment:    "Break-glass admin can rotate keys",
		},

		// Template #7: Key operator (CRITICAL - tests mid-path wildcards!)
		{
			name: "PolicyTemplate_KeyOperator_AllowCreateKey",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys",
			capability: WriteCapability,
			expected:   true,
			comment:    "Key operator can create keys",
		},
		{
			name: "PolicyTemplate_KeyOperator_AllowRotateWithMidPathWildcard",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/rotate",
			capability: RotateCapability,
			expected:   true,
			comment:    "Mid-path wildcard matches single segment for rotate",
		},
		{
			name: "PolicyTemplate_KeyOperator_AllowRotateDifferentKey",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys/user/rotate",
			capability: RotateCapability,
			expected:   true,
			comment:    "Mid-path wildcard works for any key name",
		},
		{
			name: "PolicyTemplate_KeyOperator_AllowDeleteKey",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys/payment",
			capability: DeleteCapability,
			expected:   true,
			comment:    "Trailing wildcard allows delete of any key",
		},
		{
			name: "PolicyTemplate_KeyOperator_AllowDeleteNestedPath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/something",
			capability: DeleteCapability,
			expected:   true,
			comment:    "Trailing wildcard matches nested paths",
		},
		{
			name: "PolicyTemplate_KeyOperator_DenyEncrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/encrypt",
			capability: EncryptCapability,
			expected:   false,
			comment:    "Key operator cannot encrypt (different capability needed)",
		},
		{
			name: "PolicyTemplate_KeyOperator_AllowDeleteOnRotatePath",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/rotate",
			capability: DeleteCapability,
			expected:   true,
			comment:    "Trailing wildcard /* also matches rotate paths (greedy)",
		},

		// Template #8: Tokenization operator (complex multi-policy)
		{
			name: "PolicyTemplate_TokenizationOperator_AllowCreateKey",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/keys",
			capability: WriteCapability,
			expected:   true,
			comment:    "Tokenization operator can create keys",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_AllowRotate",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/keys/pci/rotate",
			capability: RotateCapability,
			expected:   true,
			comment:    "Mid-path wildcard allows rotate for any key",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_AllowTokenize",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/keys/pci/tokenize",
			capability: EncryptCapability,
			expected:   true,
			comment:    "Mid-path wildcard allows tokenize with any key",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_AllowDetokenize",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/detokenize",
			capability: DecryptCapability,
			expected:   true,
			comment:    "Exact path match for detokenize",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_AllowValidate",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/validate",
			capability: ReadCapability,
			expected:   true,
			comment:    "Exact path match for validate",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_AllowRevoke",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/revoke",
			capability: DeleteCapability,
			expected:   true,
			comment:    "Exact path match for revoke",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_DenyDetokenizeWithWrongCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/detokenize",
			capability: ReadCapability,
			expected:   false,
			comment:    "Detokenize requires decrypt, not read",
		},
		{
			name: "PolicyTemplate_TokenizationOperator_DenyTokenizeWithWrongCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys",
					Capabilities: []Capability{WriteCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/tokenization/validate",
					Capabilities: []Capability{ReadCapability},
				},
				{
					Path:         "/v1/tokenization/revoke",
					Capabilities: []Capability{DeleteCapability},
				},
			}),
			path:       "/v1/tokenization/keys/pci/tokenize",
			capability: RotateCapability,
			expected:   false,
			comment:    "Tokenize path requires encrypt, not rotate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result, tt.comment)
		})
	}
}

func TestClient_IsAllowed_CommonMistakes(t *testing.T) {
	tests := []struct {
		name       string
		client     *Client
		path       string
		capability Capability
		expected   bool
		mistake    string
	}{
		// Mistake #1: Wrong capability for secret reads (line 238 in policies.md)
		{
			name: "CommonMistake_SecretReadWithReadInsteadOfDecrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "/v1/secrets/app/prod/key",
			capability: DecryptCapability,
			expected:   false,
			mistake:    "Using 'read' instead of 'decrypt' for GET /v1/secrets/*path",
		},
		{
			name: "CommonMistake_SecretReadFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/secrets/app/prod/key",
			capability: DecryptCapability,
			expected:   true,
			mistake:    "Fixed: Using 'decrypt' for GET /v1/secrets/*path",
		},

		// Mistake #2: Missing rotate capability (line 239 in policies.md)
		{
			name: "CommonMistake_MissingRotateCapability",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/*",
					Capabilities: []Capability{WriteCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/rotate",
			capability: RotateCapability,
			expected:   false,
			mistake:    "Missing 'rotate' capability on /v1/transit/keys/*/rotate",
		},
		{
			name: "CommonMistake_RotateCapabilityFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/transit/keys/*/rotate",
					Capabilities: []Capability{RotateCapability},
				},
			}),
			path:       "/v1/transit/keys/payment/rotate",
			capability: RotateCapability,
			expected:   true,
			mistake:    "Fixed: Added 'rotate' on /v1/transit/keys/*/rotate",
		},

		// Mistake #3: Wrong capability for detokenize (line 240 in policies.md)
		{
			name: "CommonMistake_DetokenizeWithReadInsteadOfDecrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "/v1/tokenization/detokenize",
			capability: DecryptCapability,
			expected:   false,
			mistake:    "Using 'read' instead of 'decrypt' for detokenize",
		},
		{
			name: "CommonMistake_DetokenizeFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/tokenization/detokenize",
			capability: DecryptCapability,
			expected:   true,
			mistake:    "Fixed: Using 'decrypt' for detokenize",
		},

		// Mistake #4: Over-broad wildcard (line 241 in policies.md)
		{
			name: "CommonMistake_OverBroadWildcard_ProductionAccess",
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
			path:       "/v1/secrets/production/critical",
			capability: DecryptCapability,
			expected:   true,
			mistake:    "SECURITY RISK: Wildcard '*' grants excessive access to production",
		},
		{
			name: "CommonMistake_OverBroadWildcard_ClientDelete",
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
			path:       "/v1/clients",
			capability: DeleteCapability,
			expected:   true,
			mistake:    "SECURITY RISK: Wildcard '*' allows deleting clients",
		},
		{
			name: "CommonMistake_ScopedPathFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/app/prod/*",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/secrets/app/prod/db",
			capability: DecryptCapability,
			expected:   true,
			mistake:    "Fixed: Scoped to /v1/secrets/app/prod/* instead of '*'",
		},

		// Mistake #5: Wrong capability for secret writes (line 242 in policies.md)
		{
			name: "CommonMistake_SecretWriteWithWriteInsteadOfEncrypt",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{WriteCapability},
				},
			}),
			path:       "/v1/secrets/app/key",
			capability: EncryptCapability,
			expected:   false,
			mistake:    "Using 'write' instead of 'encrypt' for POST /v1/secrets/*path",
		},
		{
			name: "CommonMistake_SecretWriteFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/secrets/app/key",
			capability: EncryptCapability,
			expected:   true,
			mistake:    "Fixed: Using 'encrypt' for POST /v1/secrets/*path",
		},

		// Mistake #6: Insufficient tokenization path scope (line 243 in policies.md)
		{
			name: "CommonMistake_TokenizationPathScope_OnlyTokenize",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/tokenization/keys/pci/tokenize",
			capability: EncryptCapability,
			expected:   true,
			mistake:    "Can tokenize with scoped path",
		},
		{
			name: "CommonMistake_TokenizationPathScope_CannotDetokenize",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
			}),
			path:       "/v1/tokenization/detokenize",
			capability: DecryptCapability,
			expected:   false,
			mistake:    "Cannot detokenize - different path required",
		},
		{
			name: "CommonMistake_TokenizationPathScopeFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/tokenization/keys/*/tokenize",
					Capabilities: []Capability{EncryptCapability},
				},
				{
					Path:         "/v1/tokenization/detokenize",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/tokenization/detokenize",
			capability: DecryptCapability,
			expected:   true,
			mistake:    "Fixed: Added explicit /v1/tokenization/detokenize policy",
		},

		// Mistake #7: Missing audit read policy (line 244 in policies.md)
		{
			name: "CommonMistake_MissingAuditLogPolicy",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{DecryptCapability},
				},
			}),
			path:       "/v1/audit-logs",
			capability: ReadCapability,
			expected:   false,
			mistake:    "Missing explicit /v1/audit-logs policy",
		},
		{
			name: "CommonMistake_AuditLogPolicyFixed",
			client: createTestClient([]PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []Capability{DecryptCapability},
				},
				{
					Path:         "/v1/audit-logs",
					Capabilities: []Capability{ReadCapability},
				},
			}),
			path:       "/v1/audit-logs",
			capability: ReadCapability,
			expected:   true,
			mistake:    "Fixed: Added explicit /v1/audit-logs read policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.client.IsAllowed(tt.path, tt.capability)
			assert.Equal(t, tt.expected, result, tt.mistake)
		})
	}
}
