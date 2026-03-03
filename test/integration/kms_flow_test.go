//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/secrets"

	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	secretsDTO "github.com/allisson/secrets/internal/secrets/http/dto"
)

// TestIntegration_KMS_CompleteFlow tests KMS master key encryption and lifecycle management.
// Validates KMS-encrypted master key loading, KEK operations, secret encryption/decryption,
// and master key rotation with backward compatibility across both database engines.
func TestIntegration_KMS_CompleteFlow(t *testing.T) {
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
			// Setup with KMS
			ctx := setupIntegrationTestWithKMS(t, tc.dbDriver)
			defer teardownIntegrationTest(t, ctx)

			var (
				//nolint:gosec // test data path, not credentials
				secretPath       = "/kms-test/password"
				secretPathStored = "kms-test/password"
				plaintextValue   = []byte("kms-protected-secret-value")
				plaintextBase64  = base64.StdEncoding.EncodeToString(plaintextValue)
			)

			// [1/7] Verify KMS master key loaded
			t.Run("01_VerifyKMSMasterKeyLoaded", func(t *testing.T) {
				// Verify master key chain is not nil
				assert.NotNil(t, ctx.masterKeyChain)

				// Verify active master key exists
				activeKey, exists := ctx.masterKeyChain.Get(ctx.masterKeyChain.ActiveMasterKeyID())
				assert.True(t, exists, "active master key should exist")
				assert.NotNil(t, activeKey, "active master key should not be nil")
				assert.Equal(t, "test-key-1", activeKey.ID)

				t.Logf("KMS master key loaded: id=%s", activeKey.ID)
			})

			// [2/7] Verify KEK created with KMS master key
			t.Run("02_VerifyKEKCreated", func(t *testing.T) {
				// KEK was created during setup - verify it exists in database
				// This validates KMS-decrypted master key successfully encrypted KEK

				kekUseCase, err := ctx.container.KekUseCase(context.Background())
				require.NoError(t, err)

				kekChain, err := kekUseCase.Unwrap(context.Background(), ctx.masterKeyChain)
				require.NoError(t, err)
				require.NotNil(t, kekChain, "KEK chain should not be nil")

				// Verify at least one KEK exists
				activeKek, exists := kekChain.Get(kekChain.ActiveKekID())
				assert.True(t, exists, "active KEK should exist")
				assert.NotNil(t, activeKek, "active KEK should not be nil")

				t.Logf("KEK created with KMS-protected master key: version=%d", activeKek.Version)
			})

			// [3/7] Create secret (encrypt with KMS-protected KEK)
			t.Run("03_CreateSecret", func(t *testing.T) {
				requestBody := secretsDTO.CreateOrUpdateSecretRequest{
					Value: plaintextBase64,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/secrets"+secretPath, requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(1), response.Version)

				t.Logf("Secret encrypted through KMS chain: path=%s", secretPath)
			})

			// [4/7] Read secret (decrypt with KMS-protected KEK)
			t.Run("04_ReadSecret", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/secrets"+secretPath, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, secretPathStored, response.Path)
				assert.Equal(t, uint(1), response.Version)
				assert.Equal(t, plaintextBase64, response.Value)

				// Verify decryption worked correctly
				decoded, err := base64.StdEncoding.DecodeString(response.Value)
				require.NoError(t, err)
				assert.Equal(t, plaintextValue, decoded)

				t.Logf("Secret decrypted through KMS chain: verified")
			})

			// [5/7] Rotate master key with KMS
			t.Run("05_RotateMasterKeyWithKMS", func(t *testing.T) {
				// Generate new master key
				newMasterKey := &cryptoDomain.MasterKey{
					ID:  "test-key-2",
					Key: make([]byte, 32),
				}
				_, err := rand.Read(newMasterKey.Key)
				require.NoError(t, err)

				// Encrypt new master key with KMS
				kmsService := cryptoService.NewKMSService()
				keeperInterface, err := kmsService.OpenKeeper(context.Background(), ctx.kmsKeyURI)
				require.NoError(t, err)
				defer func() {
					assert.NoError(t, keeperInterface.Close())
				}()

				keeper, ok := keeperInterface.(*secrets.Keeper)
				require.True(t, ok)

				newCiphertext, err := keeper.Encrypt(context.Background(), newMasterKey.Key)
				require.NoError(t, err)
				newEncodedCiphertext := base64.StdEncoding.EncodeToString(newCiphertext)

				// Get old master key ciphertext from environment
				oldMasterKeys := os.Getenv("MASTER_KEYS")

				// Update MASTER_KEYS with both old and new (comma-separated)
				dualKeys := fmt.Sprintf("%s,%s:%s", oldMasterKeys, newMasterKey.ID, newEncodedCiphertext)
				err = os.Setenv("MASTER_KEYS", dualKeys)
				require.NoError(t, err)

				err = os.Setenv("ACTIVE_MASTER_KEY_ID", newMasterKey.ID)
				require.NoError(t, err)

				// Reload master key chain with both keys
				cfg := &config.Config{
					KMSProvider: "localsecrets",
					KMSKeyURI:   ctx.kmsKeyURI,
				}
				logger := slog.New(
					slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}),
				)

				// Close old chain before loading new one
				ctx.masterKeyChain.Close()

				newChain, err := cryptoDomain.LoadMasterKeyChain(
					context.Background(),
					cfg,
					kmsService,
					logger,
				)
				require.NoError(t, err)
				ctx.masterKeyChain = newChain

				// Verify both keys loaded
				oldKey, oldExists := ctx.masterKeyChain.Get("test-key-1")
				assert.True(t, oldExists, "old master key should still exist")
				assert.NotNil(t, oldKey)

				activeKey, activeExists := ctx.masterKeyChain.Get("test-key-2")
				assert.True(t, activeExists, "new master key should exist")
				assert.NotNil(t, activeKey)
				assert.Equal(t, "test-key-2", ctx.masterKeyChain.ActiveMasterKeyID())

				t.Logf("Master key rotated: old=%s, new=%s (active)", "test-key-1", "test-key-2")
			})

			// [6/7] Verify dual master key support (backward compatibility)
			t.Run("06_VerifyBackwardCompatibility", func(t *testing.T) {
				// Read old secret encrypted with old master key
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/secrets"+secretPath, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, plaintextBase64, response.Value)

				// Verify old secret still decrypts correctly
				decoded, err := base64.StdEncoding.DecodeString(response.Value)
				require.NoError(t, err)
				assert.Equal(t, plaintextValue, decoded)

				t.Logf("Old secret decrypts after rotation: backward compatibility verified")
			})

			// [7/7] Create secret after rotation (uses new master key)
			t.Run("07_CreateSecretAfterRotation", func(t *testing.T) {
				//nolint:gosec // test data paths, not credentials
				newSecretPath := "/kms-test/new-secret"
				//nolint:gosec // test data paths, not credentials
				newSecretPathStored := "kms-test/new-secret"
				newPlaintext := []byte("secret-created-after-rotation")
				newPlaintextBase64 := base64.StdEncoding.EncodeToString(newPlaintext)

				requestBody := secretsDTO.CreateOrUpdateSecretRequest{
					Value: newPlaintextBase64,
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/secrets"+newSecretPath,
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var createResponse secretsDTO.SecretResponse
				err := json.Unmarshal(body, &createResponse)
				require.NoError(t, err)
				assert.Equal(t, newSecretPathStored, createResponse.Path)

				// Read back and verify
				resp, body = ctx.makeRequest(t, http.MethodGet, "/v1/secrets"+newSecretPath, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var readResponse secretsDTO.SecretResponse
				err = json.Unmarshal(body, &readResponse)
				require.NoError(t, err)
				assert.Equal(t, newPlaintextBase64, readResponse.Value)

				decoded, err := base64.StdEncoding.DecodeString(readResponse.Value)
				require.NoError(t, err)
				assert.Equal(t, newPlaintext, decoded)

				t.Logf("New secret created with rotated master key: verified")
			})

			t.Logf("All 7 KMS endpoint tests passed for %s", tc.dbDriver)
		})
	}
}
