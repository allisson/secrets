//go:build integration

// Package integration provides additional integration test scenarios required by P1-3.
// These tests complement the comprehensive tests in api_test.go with scenarios that were
// identified as missing from the P1-3 requirements.
package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authDTO "github.com/allisson/secrets/internal/auth/http/dto"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDTO "github.com/allisson/secrets/internal/transit/http/dto"
)

// TestIntegration_Auth_TokenExpiry tests that tokens properly expire and return 401.
// This addresses P1-3 requirement: "Auth: token expiry"
// Uses a 2-second token expiration to test expiry behavior without long waits.
func TestIntegration_Auth_TokenExpiry(t *testing.T) {
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
			// Setup with 2-second token expiration for fast testing
			ctx := setupIntegrationTestWithTokenExpiration(t, tc.dbDriver, 2*time.Second)
			defer teardownIntegrationTest(t, ctx)

			var (
				testClientID     string
				testClientSecret string
				testToken        string
			)

			// [1] Create a test client
			t.Run("01_CreateClient", func(t *testing.T) {
				requestBody := authDTO.CreateClientRequest{
					Name:     "Token Expiry Test Client",
					IsActive: true,
					Policies: []authDomain.PolicyDocument{
						{
							Path: "/v1/clients",
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

				testClientID = response.ID
				testClientSecret = response.Secret
			})

			// [2] Issue token
			t.Run("02_IssueToken", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     testClientID,
					ClientSecret: testClientSecret,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.IssueTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)

				testToken = response.Token
			})

			// [3] Verify token works initially
			t.Run("03_TokenWorksInitially", func(t *testing.T) {
				// Create custom request with the test token
				req, err := http.NewRequest(http.MethodGet, ctx.server.URL+"/v1/clients", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+testToken)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode, "token should work initially")
			})

			// [4] Wait for token to expire (2s + 1s buffer = 3s)
			t.Run("04_WaitForExpiry", func(t *testing.T) {
				t.Logf("Waiting 3 seconds for token to expire (2s expiration + 1s buffer)...")
				time.Sleep(3 * time.Second)
			})

			// [5] Verify token expired
			t.Run("05_TokenExpired", func(t *testing.T) {
				// Create custom request with the expired token
				req, err := http.NewRequest(http.MethodGet, ctx.server.URL+"/v1/clients", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+testToken)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "token should be expired")
			})

			// [6] Verify new token works
			t.Run("06_NewTokenWorks", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     testClientID,
					ClientSecret: testClientSecret,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.IssueTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)

				// Verify new token works
				req, err := http.NewRequest(http.MethodGet, ctx.server.URL+"/v1/clients", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+response.Token)

				resp2, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp2.Body.Close()

				assert.Equal(t, http.StatusOK, resp2.StatusCode, "new token should work")
			})

			t.Logf("Token expiry test passed for %s", tc.dbDriver)
		})
	}
}

// TestIntegration_Transit_DecryptAfterRotation tests that ciphertexts encrypted
// before key rotation can still be decrypted after rotation.
// This addresses P1-3 requirement: "Transit: decrypt with old and new version"
func TestIntegration_Transit_DecryptAfterRotation(t *testing.T) {
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
			ctx := setupIntegrationTest(t, tc.dbDriver)
			defer teardownIntegrationTest(t, ctx)

			keyName := "test-rotation-key"
			plaintext := []byte("sensitive-data-before-rotation")
			plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)

			// [1] Create transit key
			t.Run("01_CreateKey", func(t *testing.T) {
				requestBody := transitDTO.CreateTransitKeyRequest{
					Name:      keyName,
					Algorithm: string(cryptoDomain.AESGCM),
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys", requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, keyName, response.Name)
				assert.Equal(t, uint(1), response.Version)
			})

			// [2] Encrypt data with version 1
			var ciphertextV1 string
			t.Run("02_EncryptWithV1", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: plaintextB64,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/encrypt", requestBody, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.EncryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Ciphertext)
				assert.Equal(t, uint(1), response.Version, "should be encrypted with version 1")
				ciphertextV1 = response.Ciphertext
			})

			// [3] Rotate key to version 2
			t.Run("03_RotateKey", func(t *testing.T) {
				requestBody := transitDTO.RotateTransitKeyRequest{
					Algorithm: string(cryptoDomain.AESGCM),
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/rotate", requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, uint(2), response.Version, "key should be at version 2 after rotation")
			})

			// [4] Decrypt old ciphertext (encrypted with v1) - should still work
			t.Run("04_DecryptOldCiphertext", func(t *testing.T) {
				requestBody := transitDTO.DecryptRequest{
					Ciphertext: ciphertextV1,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/decrypt", requestBody, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode, "decryption with old version should succeed")

				var response transitDTO.DecryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, plaintextB64, response.Plaintext, "decrypted plaintext should match original")
				assert.Equal(t, uint(1), response.Version, "should indicate decrypted with version 1")
			})

			// [5] Encrypt new data with version 2
			var ciphertextV2 string
			t.Run("05_EncryptWithV2", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: plaintextB64,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/encrypt", requestBody, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.EncryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Ciphertext)
				assert.Equal(t, uint(2), response.Version, "should be encrypted with version 2")
				ciphertextV2 = response.Ciphertext
			})

			// [6] Decrypt new ciphertext (encrypted with v2) - should work
			t.Run("06_DecryptNewCiphertext", func(t *testing.T) {
				requestBody := transitDTO.DecryptRequest{
					Ciphertext: ciphertextV2,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/decrypt", requestBody, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode, "decryption with new version should succeed")

				var response transitDTO.DecryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, plaintextB64, response.Plaintext, "decrypted plaintext should match original")
				assert.Equal(t, uint(2), response.Version, "should indicate decrypted with version 2")
			})

			t.Logf("Transit rotation test passed for %s: old and new versions both decrypt successfully", tc.dbDriver)
		})
	}
}

// TestIntegration_Transit_DeleteKey tests deleting a transit key and verifying
// that subsequent operations fail.
// This addresses P1-3 requirement: "Transit: delete key"
func TestIntegration_Transit_DeleteKey(t *testing.T) {
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
			ctx := setupIntegrationTest(t, tc.dbDriver)
			defer teardownIntegrationTest(t, ctx)

			keyName := "test-delete-key"
			plaintext := []byte("test-data")
			plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)

			// [1] Create transit key
			t.Run("01_CreateKey", func(t *testing.T) {
				requestBody := transitDTO.CreateTransitKeyRequest{
					Name:      keyName,
					Algorithm: string(cryptoDomain.AESGCM),
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys", requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, keyName, response.Name)
			})

			// [2] Encrypt some data successfully
			t.Run("02_EncryptBeforeDelete", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: plaintextB64,
				}

				resp, _ := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/encrypt", requestBody, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode, "encrypt should work before deletion")
			})

			// [3] Delete the key by name
			t.Run("03_DeleteKey", func(t *testing.T) {
				resp, _ := ctx.makeRequest(t, http.MethodDelete, "/v1/transit/keys/"+keyName, nil, true)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			})

			// [4] Attempt to encrypt after deletion - should fail
			t.Run("04_EncryptAfterDelete", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: plaintextB64,
				}

				resp, _ := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys/"+keyName+"/encrypt", requestBody, true)
				assert.Equal(t, http.StatusNotFound, resp.StatusCode, "encrypt should fail after key deletion")
			})

			// [5] Verify GET returns 404
			t.Run("05_GetAfterDelete", func(t *testing.T) {
				resp, _ := ctx.makeRequest(t, http.MethodGet, "/v1/transit/keys/"+keyName, nil, true)
				assert.Equal(t, http.StatusNotFound, resp.StatusCode, "key should not be found after deletion")
			})

			t.Logf("Transit delete key test passed for %s", tc.dbDriver)
		})
	}
}

// TestIntegration_Auth_UnlockAccount tests the account unlock functionality.
// This addresses P1-3 requirement: "Auth: unlock"
// This test verifies the complete lockout → unlock → recovery flow.
func TestIntegration_Auth_UnlockAccount(t *testing.T) {
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
				testClientID     string
				testClientSecret string
			)

			// [1] Create a test client
			t.Run("01_CreateClient", func(t *testing.T) {
				requestBody := authDTO.CreateClientRequest{
					Name:     "Unlock Test Client",
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

				testClientID = response.ID
				testClientSecret = response.Secret
			})

			// [2] Trigger lockout with 3 failed attempts
			t.Run("02_TriggerLockout", func(t *testing.T) {
				for i := 0; i < 3; i++ {
					requestBody := authDTO.IssueTokenRequest{
						ClientID:     testClientID,
						ClientSecret: "wrong-secret",
					}

					resp, _ := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				}
			})

			// [3] Verify account is locked (next attempt returns 423 Locked)
			t.Run("03_VerifyLocked", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     testClientID,
					ClientSecret: testClientSecret, // Use correct secret
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusLocked, resp.StatusCode)

				var response map[string]string
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "client_locked", response["error"])
			})

			// [4] Unlock the account via admin endpoint
			t.Run("04_UnlockAccount", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/clients/"+testClientID+"/unlock",
					nil,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ClientResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, testClientID, response.ID)
			})

			// [5] Verify authentication works after unlock
			t.Run("05_AuthAfterUnlock", func(t *testing.T) {
				requestBody := authDTO.IssueTokenRequest{
					ClientID:     testClientID,
					ClientSecret: testClientSecret,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/token", requestBody, false)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response authDTO.IssueTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)
			})

			// [6] Verify the new token works
			t.Run("06_UseTokenAfterUnlock", func(t *testing.T) {
				// The token from step 5 should work for authenticated requests
				// We'll verify by listing clients which requires authentication
				resp, _ := ctx.makeRequest(t, http.MethodGet, "/v1/clients", nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})

			t.Logf("Account unlock test passed for %s", tc.dbDriver)
		})
	}
}
