//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	secretsDTO "github.com/allisson/secrets/internal/secrets/http/dto"
)

// TestIntegration_Secrets_CompleteFlow tests the secrets engine complete lifecycle.
// Validates secret creation, versioning, updates, version-specific reads, and deletion.
func TestIntegration_Secrets_CompleteFlow(t *testing.T) {
	// Skip if short mode (integration tests can be slow)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name     string
		dbDriver string
	}{
		{"PostgreSQL", "postgres"},
		{"MySQL", "mysql"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			ctx := setupIntegrationTest(t, tc.dbDriver)
			defer teardownIntegrationTest(t, ctx)

			// Variables to store test data
			var (
				secretPath            = "/integration-test/password"
				secretPathStored      = "integration-test/password" // API stores without leading slash
				plaintextValue1       = []byte("super-secret-value-v1")
				plaintextValue2       = []byte("super-secret-value-v2-updated")
				plaintextValue1Base64 = base64.StdEncoding.EncodeToString(plaintextValue1)
				plaintextValue2Base64 = base64.StdEncoding.EncodeToString(plaintextValue2)
			)

			// [1/6] Test POST /v1/secrets/*path - Create secret (version 1)
			t.Run("01_CreateSecret", func(t *testing.T) {
				requestBody := secretsDTO.CreateOrUpdateSecretRequest{
					Value: plaintextValue1Base64,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/secrets"+secretPath, requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(1), response.Version)
				assert.Empty(t, response.Value, "value should not be returned on create")
			})

			// [2/6] Test GET /v1/secrets/*path - Read secret
			t.Run("02_ReadSecret", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/secrets"+secretPath, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(1), response.Version)
				assert.Equal(t, plaintextValue1Base64, response.Value)

				// Verify the value decodes correctly
				decoded, err := base64.StdEncoding.DecodeString(response.Value)
				require.NoError(t, err)
				assert.Equal(t, plaintextValue1, decoded)
			})

			// [3/6] Test POST /v1/secrets/*path - Update secret (version 2)
			t.Run("03_UpdateSecret", func(t *testing.T) {
				requestBody := secretsDTO.CreateOrUpdateSecretRequest{
					Value: plaintextValue2Base64,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/secrets"+secretPath, requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(2), response.Version, "version should increment to 2")
				assert.Empty(t, response.Value, "value should not be returned on create/update")
			})

			// [4/6] Test GET /v1/secrets/*path - Read updated secret (latest version)
			t.Run("04_ReadUpdatedSecret", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/secrets"+secretPath, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(2), response.Version, "should return latest version (v2)")
				assert.Equal(t, plaintextValue2Base64, response.Value)

				// Verify the value decodes correctly
				decoded, err := base64.StdEncoding.DecodeString(response.Value)
				require.NoError(t, err)
				assert.Equal(t, plaintextValue2, decoded, "should return updated value")
			})

			// [5/6] Test GET /v1/secrets/*path?version=1 - Read specific version
			t.Run("05_ReadSecretVersion1", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodGet,
					"/v1/secrets"+secretPath+"?version=1",
					nil,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(1), response.Version, "should return version 1")
				assert.Equal(t, plaintextValue1Base64, response.Value)

				// Verify the value decodes correctly
				decoded, err := base64.StdEncoding.DecodeString(response.Value)
				require.NoError(t, err)
				assert.Equal(t, plaintextValue1, decoded, "should return original v1 value")
			})

			// [6/6] Test DELETE /v1/secrets/*path - Delete secret
			t.Run("06_DeleteSecret", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodDelete,
					"/v1/secrets"+secretPath,
					nil,
					true,
				)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, body)
			})

			t.Logf("All 6 secrets endpoint tests passed for %s", tc.dbDriver)
		})
	}
}
