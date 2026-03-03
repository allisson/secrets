//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tokenizationDTO "github.com/allisson/secrets/internal/tokenization/http/dto"
)

// TestIntegration_Tokenization_CompleteFlow tests all tokenization endpoints in a complete lifecycle.
// This test validates tokenization functionality including deterministic/non-deterministic modes,
// token expiration, key rotation, and token lifecycle management across both database engines.
func TestIntegration_Tokenization_CompleteFlow(t *testing.T) {
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

			// Variables to store created resource IDs and tokens for later operations
			var (
				tokenizationKeyName1 = "integration-test-key-uuid"
				tokenizationKeyName2 = "integration-test-key-deterministic"
				tokenizationKeyID1   uuid.UUID
				tokenizationKeyID2   uuid.UUID
				testToken            string
				deterministicToken1  string
				deterministicToken2  string
				plaintextValue       = []byte("sensitive-credit-card-4532015112830366")
				plaintextValueBase64 = base64.StdEncoding.EncodeToString(plaintextValue)
				testMetadata         = map[string]any{"user_id": "12345", "source": "integration-test"}
			)

			// [1/12] Test POST /v1/tokenization/keys - Create UUID format tokenization key
			t.Run("01_CreateTokenizationKey_UUID", func(t *testing.T) {
				requestBody := tokenizationDTO.CreateTokenizationKeyRequest{
					Name:            tokenizationKeyName1,
					FormatType:      "uuid",
					IsDeterministic: false,
					Algorithm:       "aes-gcm",
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response tokenizationDTO.TokenizationKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, tokenizationKeyName1, response.Name)
				assert.Equal(t, uint(1), response.Version)
				assert.Equal(t, "uuid", response.FormatType)
				assert.False(t, response.IsDeterministic)
				assert.False(t, response.CreatedAt.IsZero())

				// Store ID for later operations
				parsedID, err := uuid.Parse(response.ID)
				require.NoError(t, err)
				tokenizationKeyID1 = parsedID
			})

			// [2/12] Test POST /v1/tokenization/keys/:name/tokenize - Tokenize with UUID format
			t.Run("02_Tokenize_UUID", func(t *testing.T) {
				requestBody := tokenizationDTO.TokenizeRequest{
					Plaintext: plaintextValueBase64,
					Metadata:  testMetadata,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys/"+tokenizationKeyName1+"/tokenize",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response tokenizationDTO.TokenizeResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.False(t, response.CreatedAt.IsZero())
				assert.Nil(t, response.ExpiresAt) // No TTL specified
				assert.Equal(t, testMetadata, response.Metadata)

				// Verify token is in UUID format
				_, err = uuid.Parse(response.Token)
				assert.NoError(t, err, "token should be valid UUID format")

				// Store token for detokenization
				testToken = response.Token
			})

			// [3/12] Test POST /v1/tokenization/detokenize - Detokenize UUID token
			t.Run("03_Detokenize_UUID", func(t *testing.T) {
				requestBody := tokenizationDTO.DetokenizeRequest{
					Token: testToken,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/detokenize",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response tokenizationDTO.DetokenizeResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Plaintext)
				assert.Equal(t, testMetadata, response.Metadata)

				// Verify decrypted value matches original
				assert.Equal(t, plaintextValueBase64, response.Plaintext)
			})

			// [4/12] Test POST /v1/tokenization/validate - Validate active token
			t.Run("04_ValidateToken_Valid", func(t *testing.T) {
				requestBody := tokenizationDTO.ValidateTokenRequest{
					Token: testToken,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/validate",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response tokenizationDTO.ValidateTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.True(t, response.Valid, "token should be valid")
			})

			// [5/12] Test POST /v1/tokenization/revoke - Revoke token
			t.Run("05_RevokeToken", func(t *testing.T) {
				requestBody := tokenizationDTO.RevokeTokenRequest{
					Token: testToken,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/revoke",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, body)
			})

			// [6/12] Test POST /v1/tokenization/validate - Validate revoked token
			t.Run("06_ValidateToken_Revoked", func(t *testing.T) {
				requestBody := tokenizationDTO.ValidateTokenRequest{
					Token: testToken,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/validate",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response tokenizationDTO.ValidateTokenResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.False(t, response.Valid, "revoked token should be invalid")
			})

			// [7/12] Test POST /v1/tokenization/keys - Create deterministic tokenization key
			t.Run("07_CreateTokenizationKey_Deterministic", func(t *testing.T) {
				requestBody := tokenizationDTO.CreateTokenizationKeyRequest{
					Name:            tokenizationKeyName2,
					FormatType:      "alphanumeric",
					IsDeterministic: true,
					Algorithm:       "chacha20-poly1305",
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response tokenizationDTO.TokenizationKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, tokenizationKeyName2, response.Name)
				assert.Equal(t, uint(1), response.Version)
				assert.Equal(t, "alphanumeric", response.FormatType)
				assert.True(t, response.IsDeterministic)

				// Store ID for later operations
				parsedID, err := uuid.Parse(response.ID)
				require.NoError(t, err)
				tokenizationKeyID2 = parsedID
			})

			// [8/12] Test POST /v1/tokenization/keys/:name/tokenize - Deterministic tokenization
			t.Run("08_Tokenize_Deterministic_SameValue", func(t *testing.T) {
				requestBody := tokenizationDTO.TokenizeRequest{
					Plaintext: plaintextValueBase64,
				}

				// First tokenization
				resp1, body1 := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys/"+tokenizationKeyName2+"/tokenize",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp1.StatusCode)

				var response1 tokenizationDTO.TokenizeResponse
				err := json.Unmarshal(body1, &response1)
				require.NoError(t, err)
				assert.NotEmpty(t, response1.Token)
				deterministicToken1 = response1.Token

				// Second tokenization with same plaintext
				resp2, body2 := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys/"+tokenizationKeyName2+"/tokenize",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp2.StatusCode)

				var response2 tokenizationDTO.TokenizeResponse
				err = json.Unmarshal(body2, &response2)
				require.NoError(t, err)
				assert.NotEmpty(t, response2.Token)
				deterministicToken2 = response2.Token

				// Verify both tokens are identical (deterministic behavior)
				assert.Equal(t, deterministicToken1, deterministicToken2,
					"deterministic tokenization should produce same token for same plaintext")
			})

			// [9/12] Test POST /v1/tokenization/keys/:name/tokenize - Tokenize with TTL
			t.Run("09_Tokenize_WithTTL", func(t *testing.T) {
				ttlSeconds := 60
				requestBody := tokenizationDTO.TokenizeRequest{
					Plaintext: plaintextValueBase64,
					TTL:       &ttlSeconds,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys/"+tokenizationKeyName1+"/tokenize",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response tokenizationDTO.TokenizeResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.False(t, response.CreatedAt.IsZero())
				assert.NotNil(t, response.ExpiresAt, "ExpiresAt should be set when TTL is provided")

				// Verify ExpiresAt is approximately CreatedAt + TTL
				expectedExpiry := response.CreatedAt.Add(time.Duration(ttlSeconds) * time.Second)
				assert.WithinDuration(t, expectedExpiry, *response.ExpiresAt, 2*time.Second,
					"ExpiresAt should be approximately CreatedAt + TTL")
			})

			// [10/12] Test POST /v1/tokenization/keys/:name/rotate - Rotate tokenization key
			t.Run("10_RotateTokenizationKey", func(t *testing.T) {
				requestBody := tokenizationDTO.RotateTokenizationKeyRequest{
					FormatType:      "uuid",
					IsDeterministic: false,
					Algorithm:       "chacha20-poly1305", // Rotate to different algorithm
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys/"+tokenizationKeyName1+"/rotate",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response tokenizationDTO.TokenizationKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.NotEqual(
					t,
					tokenizationKeyID1.String(),
					response.ID,
					"rotation creates new key with new ID",
				)
				assert.Equal(t, tokenizationKeyName1, response.Name, "name should remain the same")
				assert.Equal(t, uint(2), response.Version, "version should increment after rotation")
				assert.Equal(t, "uuid", response.FormatType)
				assert.False(t, response.IsDeterministic)
			})

			// [11/12] Test POST /v1/tokenization/keys/:name/tokenize - Tokenize with rotated key
			t.Run("11_Tokenize_WithRotatedKey", func(t *testing.T) {
				newPlaintext := []byte("new-data-after-rotation")
				requestBody := tokenizationDTO.TokenizeRequest{
					Plaintext: base64.StdEncoding.EncodeToString(newPlaintext),
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/keys/"+tokenizationKeyName1+"/tokenize",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response tokenizationDTO.TokenizeResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Token)

				// Verify token is in UUID format
				_, err = uuid.Parse(response.Token)
				assert.NoError(t, err, "token should be valid UUID format")

				// Verify we can detokenize with the rotated key
				detokenizeRequest := tokenizationDTO.DetokenizeRequest{
					Token: response.Token,
				}

				detokenizeResp, detokenizeBody := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/tokenization/detokenize",
					detokenizeRequest,
					true,
				)
				assert.Equal(t, http.StatusOK, detokenizeResp.StatusCode)

				var detokenizeResponse tokenizationDTO.DetokenizeResponse
				err = json.Unmarshal(detokenizeBody, &detokenizeResponse)
				require.NoError(t, err)
				assert.Equal(t, base64.StdEncoding.EncodeToString(newPlaintext), detokenizeResponse.Plaintext)
			})

			// [12/12] Test DELETE /v1/tokenization/keys/:id - Delete tokenization key
			t.Run("12_DeleteTokenizationKey", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodDelete,
					"/v1/tokenization/keys/"+tokenizationKeyID2.String(),
					nil,
					true,
				)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, body)
			})

			t.Logf("All 12 tokenization endpoint tests passed for %s", tc.dbDriver)
		})
	}
}
