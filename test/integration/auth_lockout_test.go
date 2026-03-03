//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authDTO "github.com/allisson/secrets/internal/auth/http/dto"
)

// TestIntegration_AccountLockout_CompleteFlow tests the full lockout → unlock → re-auth cycle.
// Validates account lockout enforcement and admin unlock capability against both databases.
func TestIntegration_AccountLockout_CompleteFlow(t *testing.T) {
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
			// Setup with lockout: 3 max attempts, 5 minute lockout duration
			ctx := setupIntegrationTestWithLockout(t, tc.dbDriver, 3, 5*time.Minute)
			defer teardownIntegrationTest(t, ctx)

			var (
				victimClientID     string
				victimClientSecret string
			)

			// [1/5] Create a victim client that will be locked out
			t.Run("01_CreateVictimClient", func(t *testing.T) {
				requestBody := authDTO.CreateClientRequest{
					Name:     "Victim Client",
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

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/clients", requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.CreateClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.NotEmpty(t, response.Secret)

				victimClientID = response.ID
				victimClientSecret = response.Secret
			})

			// [2/5] Exhaust failed attempts (3×401) — each returns 401, 3rd attempt sets lock
			t.Run("02_FailedAttempts_Accumulate", func(t *testing.T) {
				for range 3 {
					requestBody := authDTO.IssueTokenRequest{
						ClientID:     victimClientID,
						ClientSecret: "wrong-secret",
					}

					resp, _ := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				}
			})

			// [3/5] Next attempt triggers lockout — 423 with client_locked error
			t.Run("03_LockedAttempt", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     victimClientID,
					ClientSecret: "wrong-secret",
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusLocked, resp.StatusCode)

				var response map[string]string
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "client_locked", response["error"])
			})

			// [4/5] Admin unlocks the victim client
			t.Run("04_UnlockClient", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/clients/"+victimClientID+"/unlock",
					nil,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, victimClientID, response.ID)
			})

			// [5/5] Victim can authenticate again after unlock
			t.Run("05_AuthAfterUnlock", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     victimClientID,
					ClientSecret: victimClientSecret,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.IssueTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)
			})

			t.Logf("All 5 account lockout tests passed for %s", tc.dbDriver)
		})
	}
}
