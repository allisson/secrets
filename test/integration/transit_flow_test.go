//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	transitDTO "github.com/allisson/secrets/internal/transit/http/dto"
)

// TestIntegration_Transit_CompleteFlow tests all transit encryption endpoints in a complete lifecycle.
// This test validates transit key creation, encryption/decryption, key rotation, and backward
// compatibility (decrypting old ciphertexts after rotation) across both database engines.
func TestIntegration_Transit_CompleteFlow(t *testing.T) {
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

			// Variables to store created resource IDs and encrypted data for later operations
			var (
				transitKeyName = "integration-test-transit-key"
				transitKeyID   uuid.UUID
				plaintext1     = []byte("transit-test-data-1")
				plaintext2     = []byte("transit-test-data-2")
				ciphertext1    string // Encrypted with version 1
				ciphertext2    string // Encrypted with different plaintext
				ciphertextV2   string // Encrypted with version 2 (after rotation)
			)

			// [1/8] Test POST /v1/transit/keys - Create transit key
			t.Run("01_CreateTransitKey", func(t *testing.T) {
				requestBody := transitDTO.CreateTransitKeyRequest{
					Name:      transitKeyName,
					Algorithm: string(cryptoDomain.AESGCM),
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/transit/keys", requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, transitKeyName, response.Name)
				assert.Equal(t, uint(1), response.Version)
				assert.NotEmpty(t, response.DekID)
				assert.False(t, response.CreatedAt.IsZero())

				// Store transit key ID for later deletion
				parsedID, err := uuid.Parse(response.ID)
				require.NoError(t, err)
				transitKeyID = parsedID
			})

			// [2/11] Test GET /v1/transit/keys/:name - Get transit key
			t.Run("02_GetTransitKey", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/transit/keys/"+transitKeyName, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.TransitKeyMetadataResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, transitKeyName, response.Name)
				assert.Equal(t, "aes-gcm", response.Type)
				assert.Equal(t, uint(1), response.Version)
				assert.False(t, response.CreatedAt.IsZero())
			})

			// [3/11] Test POST /v1/transit/keys/:name/encrypt - Encrypt with transit key
			t.Run("03_Encrypt", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: base64.StdEncoding.EncodeToString(plaintext1),
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/encrypt",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.EncryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Ciphertext)
				assert.Equal(t, uint(1), response.Version)

				// Store ciphertext for decryption test
				ciphertext1 = response.Ciphertext

				// Verify ciphertext format: <version>:<base64>
				assert.Contains(t, response.Ciphertext, ":")
			})

			// [3/8] Test POST /v1/transit/keys/:name/decrypt - Decrypt with transit key
			t.Run("03_Decrypt", func(t *testing.T) {
				requestBody := transitDTO.DecryptRequest{
					Ciphertext: ciphertext1,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/decrypt",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.DecryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Plaintext)
				assert.Equal(t, uint(1), response.Version)

				// Verify decrypted value matches original
				decoded, err := base64.StdEncoding.DecodeString(response.Plaintext)
				require.NoError(t, err)
				assert.Equal(t, plaintext1, decoded)
			})

			// [4/8] Test POST /v1/transit/keys/:name/encrypt - Encrypt different value
			t.Run("04_EncryptDifferentValue", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: base64.StdEncoding.EncodeToString(plaintext2),
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/encrypt",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.EncryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Ciphertext)
				assert.Equal(t, uint(1), response.Version)

				// Store second ciphertext
				ciphertext2 = response.Ciphertext

				// Verify different plaintext produces different ciphertext
				assert.NotEqual(t, ciphertext1, ciphertext2)
			})

			// [5/8] Test POST /v1/transit/keys/:name/rotate - Rotate transit key
			t.Run("05_RotateTransitKey", func(t *testing.T) {
				requestBody := transitDTO.RotateTransitKeyRequest{
					Algorithm: string(cryptoDomain.AESGCM),
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/rotate",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, transitKeyName, response.Name)
				assert.Equal(t, uint(2), response.Version) // Version should increment to 2
			})

			// [7/11] Test GET /v1/transit/keys/:name?version=1 - Get specific version
			t.Run("07_GetSpecificVersion", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/transit/keys/"+transitKeyName+"?version=1", nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.TransitKeyMetadataResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, transitKeyName, response.Name)
				assert.Equal(t, uint(1), response.Version)
			})

			// [8/11] Test POST /v1/transit/keys/:name/encrypt - Encrypt with rotated key (version 2)
			t.Run("08_EncryptWithRotatedKey", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: base64.StdEncoding.EncodeToString(plaintext1),
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/encrypt",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.EncryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Ciphertext)
				assert.Equal(t, uint(2), response.Version) // Should use new version 2

				// Store version 2 ciphertext
				ciphertextV2 = response.Ciphertext

				// Verify version 2 ciphertext is different from version 1
				assert.NotEqual(t, ciphertext1, ciphertextV2)
			})

			// [9/11] Test POST /v1/transit/keys/:name/decrypt - Decrypt old ciphertext (backward compatibility)
			t.Run("09_DecryptOldCiphertext", func(t *testing.T) {
				requestBody := transitDTO.DecryptRequest{
					Ciphertext: ciphertext1, // Use version 1 ciphertext
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/decrypt",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.DecryptResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Plaintext)
				assert.Equal(t, uint(1), response.Version) // Should indicate version 1 was used

				// Verify decrypted value still matches original (backward compatibility)
				decoded, err := base64.StdEncoding.DecodeString(response.Plaintext)
				require.NoError(t, err)
				assert.Equal(t, plaintext1, decoded)
			})

			// [10/11] Test AEAD Context
			t.Run("10_AEADContext", func(t *testing.T) {
				contextAAD := []byte("integration-test-context")
				wrongContext := []byte("wrong-context")

				// 1. Encrypt with context
				encryptRequest := transitDTO.EncryptRequest{
					Plaintext: base64.StdEncoding.EncodeToString(plaintext1),
					Context:   base64.StdEncoding.EncodeToString(contextAAD),
				}
				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/encrypt",
					encryptRequest,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var encryptResponse transitDTO.EncryptResponse
				err := json.Unmarshal(body, &encryptResponse)
				require.NoError(t, err)
				ciphertextWithContext := encryptResponse.Ciphertext

				// 2. Decrypt with correct context
				decryptRequest := transitDTO.DecryptRequest{
					Ciphertext: ciphertextWithContext,
					Context:    base64.StdEncoding.EncodeToString(contextAAD),
				}
				resp, body = ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/decrypt",
					decryptRequest,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var decryptResponse transitDTO.DecryptResponse
				err = json.Unmarshal(body, &decryptResponse)
				require.NoError(t, err)
				decoded, err := base64.StdEncoding.DecodeString(decryptResponse.Plaintext)
				require.NoError(t, err)
				assert.Equal(t, plaintext1, decoded)

				// 3. Decrypt with wrong context (should fail)
				decryptRequestWrong := transitDTO.DecryptRequest{
					Ciphertext: ciphertextWithContext,
					Context:    base64.StdEncoding.EncodeToString(wrongContext),
				}
				resp, _ = ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/decrypt",
					decryptRequestWrong,
					true,
				)
				// Should return 422 Unprocessable Entity (decryption failed)
				assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
			})

			// [11/11] Test DELETE /v1/transit/keys/:id - Delete transit key
			t.Run("11_DeleteTransitKey", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodDelete,
					"/v1/transit/keys/"+transitKeyID.String(),
					nil,
					true,
				)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, body)
			})

			t.Logf("All 11 transit endpoint tests passed for %s", tc.dbDriver)

		})
	}
}
