//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authDTO "github.com/allisson/secrets/internal/auth/http/dto"
)

// TestIntegration_Auth_CompleteFlow tests authentication, client management, and audit logging.
// Validates complete client lifecycle including token issuance, CRUD operations, and audit trails.
func TestIntegration_Auth_CompleteFlow(t *testing.T) {
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

			// Variables to store created resource IDs for later operations
			var (
				newClientID uuid.UUID
			)

			// [1/8] Test POST /v1/token - Issue authentication token
			t.Run("01_IssueToken", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     ctx.rootClient.ID.String(),
					ClientSecret: ctx.rootSecret,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.IssueTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.False(t, response.ExpiresAt.IsZero())

				// Update token for subsequent requests
				ctx.rootToken = response.Token
			})

			// [2/8] Test POST /v1/clients - Create new client
			t.Run("02_CreateClient", func(t *testing.T) {
				requestBody := authDTO.CreateClientRequest{
					Name:     "Test Client",
					IsActive: true,
					Policies: []authDomain.PolicyDocument{
						{
							Path: "/v1/secrets/test/*",
							Capabilities: []authDomain.Capability{
								authDomain.ReadCapability,
								authDomain.WriteCapability,
							},
						},
					},
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/clients", requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.CreateClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.NotEmpty(t, response.Secret)

				// Store client ID for later operations
				parsedID, err := uuid.Parse(response.ID)
				require.NoError(t, err)
				newClientID = parsedID
			})

			// [3/8] Test GET /v1/clients/:id - Get client by ID
			t.Run("03_GetClient", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodGet,
					"/v1/clients/"+newClientID.String(),
					nil,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, newClientID.String(), response.ID)
				assert.Equal(t, "Test Client", response.Name)
				assert.True(t, response.IsActive)
				assert.Len(t, response.Policies, 1)
			})

			// [4/8] Test PUT /v1/clients/:id - Update client
			t.Run("04_UpdateClient", func(t *testing.T) {
				requestBody := authDTO.UpdateClientRequest{
					Name:     "Updated Test Client",
					IsActive: true,
					Policies: []authDomain.PolicyDocument{
						{
							Path: "/v1/secrets/*",
							Capabilities: []authDomain.Capability{
								authDomain.ReadCapability,
							},
						},
					},
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPut,
					"/v1/clients/"+newClientID.String(),
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "Updated Test Client", response.Name)
				assert.True(t, response.IsActive)
				assert.Len(t, response.Policies, 1)
			})

			// [5/8] Test GET /v1/clients - List clients
			t.Run("05_ListClients", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/clients", nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ListClientsResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Data)
				assert.GreaterOrEqual(t, len(response.Data), 2, "should have at least root + new client")
			})

			// [6/8] Test GET /v1/audit-logs - List audit logs
			t.Run("06_ListAuditLogs", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/audit-logs", nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ListAuditLogsResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Data)

				// Verify some audit log entries exist for our operations
				assert.GreaterOrEqual(t, len(response.Data), 5, "should have multiple audit log entries")

				// Verify audit log structure
				firstLog := response.Data[0]
				assert.NotEmpty(t, firstLog.ID)
				assert.NotEmpty(t, firstLog.ClientID)
				assert.NotEmpty(t, firstLog.Capability)
			})

			// [7/8] Test DELETE /v1/clients/:id - Delete client (soft delete)
			t.Run("07_DeleteClient", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodDelete,
					"/v1/clients/"+newClientID.String(),
					nil,
					true,
				)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, body)
			})

			// [8/8] Test GET /v1/clients/:id - Verify client is inactive after deletion
			t.Run("08_VerifyClientInactive", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodGet,
					"/v1/clients/"+newClientID.String(),
					nil,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.False(t, response.IsActive, "client should be inactive after deletion")
			})

			// [9/10] Test DELETE /v1/token - Self-revocation
			t.Run("09_SelfRevokeToken", func(t *testing.T) {
				// Create a temporary client and token
				clientRequest := authDTO.CreateClientRequest{
					Name:     "Revoke Test Client",
					IsActive: true,
					Policies: []authDomain.PolicyDocument{{Path: "*", Capabilities: []authDomain.Capability{authDomain.ReadCapability}}},
				}
				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/clients", clientRequest, true)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				var clientResponse authDTO.CreateClientResponse
				err := json.Unmarshal(body, &clientResponse)
				require.NoError(t, err)

				issueRequest := authDTO.IssueTokenRequest{
					ClientID:     clientResponse.ID,
					ClientSecret: clientResponse.Secret,
				}
				resp, body = ctx.makeRequest(t, http.MethodPost, "/v1/token", issueRequest, false)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				var tokenResponse authDTO.IssueTokenResponse
				err = json.Unmarshal(body, &tokenResponse)
				require.NoError(t, err)
				tempToken := tokenResponse.Token

				// Verify token works
				req, _ := http.NewRequest(http.MethodGet, ctx.server.URL+"/v1/clients/"+clientResponse.ID, nil)
				req.Header.Set("Authorization", "Bearer "+tempToken)
				resp, err = http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// Revoke token
				req, _ = http.NewRequest(http.MethodDelete, ctx.server.URL+"/v1/token", nil)
				req.Header.Set("Authorization", "Bearer "+tempToken)
				resp, err = http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)

				// Verify token NO LONGER works
				req, _ = http.NewRequest(http.MethodGet, ctx.server.URL+"/v1/clients/"+clientResponse.ID, nil)
				req.Header.Set("Authorization", "Bearer "+tempToken)
				resp, err = http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			// [10/10] Test DELETE /v1/clients/:id/tokens - Client-wide revocation
			t.Run("10_ClientWideRevocation", func(t *testing.T) {
				// Create a temporary client
				clientRequest := authDTO.CreateClientRequest{
					Name:     "Multi Revoke Test Client",
					IsActive: true,
					Policies: []authDomain.PolicyDocument{{Path: "*", Capabilities: []authDomain.Capability{authDomain.ReadCapability}}},
				}
				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/clients", clientRequest, true)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				var clientResponse authDTO.CreateClientResponse
				err := json.Unmarshal(body, &clientResponse)
				require.NoError(t, err)

				// Issue 2 tokens
				issueRequest := authDTO.IssueTokenRequest{
					ClientID:     clientResponse.ID,
					ClientSecret: clientResponse.Secret,
				}
				
				resp, body = ctx.makeRequest(t, http.MethodPost, "/v1/token", issueRequest, false)
				var t1 authDTO.IssueTokenResponse
				json.Unmarshal(body, &t1)
				
				resp, body = ctx.makeRequest(t, http.MethodPost, "/v1/token", issueRequest, false)
				var t2 authDTO.IssueTokenResponse
				json.Unmarshal(body, &t2)

				// Revoke all tokens for this client
				resp, _ = ctx.makeRequest(t, http.MethodDelete, "/v1/clients/"+clientResponse.ID+"/tokens", nil, true)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)

				// Verify BOTH tokens are rejected
				for _, token := range []string{t1.Token, t2.Token} {
					req, _ := http.NewRequest(http.MethodGet, ctx.server.URL+"/v1/clients/"+clientResponse.ID, nil)
					req.Header.Set("Authorization", "Bearer "+token)
					resp, err = http.DefaultClient.Do(req)
					require.NoError(t, err)
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				}
			})

			t.Logf("All 10 auth endpoint tests passed for %s", tc.dbDriver)
		})
	}
}
