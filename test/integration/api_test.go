// Package integration provides comprehensive end-to-end integration tests for the Secrets API.
// Tests all API endpoints against both PostgreSQL and MySQL databases.
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/app"
	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authDTO "github.com/allisson/secrets/internal/auth/http/dto"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	secretsDTO "github.com/allisson/secrets/internal/secrets/http/dto"
	"github.com/allisson/secrets/internal/testutil"
	transitDTO "github.com/allisson/secrets/internal/transit/http/dto"
)

// integrationTestContext holds all dependencies and state for integration testing.
type integrationTestContext struct {
	container      *app.Container
	db             *sql.DB
	server         *httptest.Server
	rootClient     *authDomain.Client
	rootToken      string
	rootSecret     string
	masterKeyChain *cryptoDomain.MasterKeyChain
	dbDriver       string
}

// makeRequest performs an HTTP request and returns the response and body.
func (ctx *integrationTestContext) makeRequest(
	t *testing.T,
	method, path string,
	body interface{},
	useAuth bool,
) (*http.Response, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err, "failed to marshal request body")
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, ctx.server.URL+path, bodyReader)
	require.NoError(t, err, "failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if useAuth {
		req.Header.Set("Authorization", "Bearer "+ctx.rootToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "failed to perform request")

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	if closeErr := resp.Body.Close(); closeErr != nil {
		t.Logf("Warning: failed to close response body: %v", closeErr)
	}

	return resp, respBody
}

// generateMasterKey creates a new 32-byte master key for testing.
func generateMasterKey() *cryptoDomain.MasterKey {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Sprintf("failed to generate master key: %v", err))
	}
	return &cryptoDomain.MasterKey{
		ID:  "test-key-1",
		Key: key,
	}
}

// createMasterKeyChain creates a master key chain with a single master key.
func createMasterKeyChain(masterKey *cryptoDomain.MasterKey) *cryptoDomain.MasterKeyChain {
	// Use environment variable format to create the chain
	keyBase64 := base64.StdEncoding.EncodeToString(masterKey.Key)
	if err := os.Setenv("MASTER_KEYS", fmt.Sprintf("%s:%s", masterKey.ID, keyBase64)); err != nil {
		panic(fmt.Sprintf("failed to set MASTER_KEYS env: %v", err))
	}
	if err := os.Setenv("ACTIVE_MASTER_KEY_ID", masterKey.ID); err != nil {
		panic(fmt.Sprintf("failed to set ACTIVE_MASTER_KEY_ID env: %v", err))
	}

	chain, err := cryptoDomain.LoadMasterKeyChainFromEnv()
	if err != nil {
		panic(fmt.Sprintf("failed to create master key chain: %v", err))
	}

	return chain
}

// setupIntegrationTest initializes all components for integration testing.
func setupIntegrationTest(t *testing.T, dbDriver string) *integrationTestContext {
	t.Helper()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup database
	var db *sql.DB
	var dsn string
	if dbDriver == "postgres" {
		db = testutil.SetupPostgresDB(t)
		dsn = testutil.PostgresTestDSN
	} else {
		db = testutil.SetupMySQLDB(t)
		dsn = testutil.MySQLTestDSN
	}

	// Generate ephemeral master key for testing
	masterKey := generateMasterKey()
	masterKeyChain := createMasterKeyChain(masterKey)

	// Create configuration
	cfg := &config.Config{
		DBDriver:             dbDriver,
		DBConnectionString:   dsn,
		DBMaxOpenConnections: 10,
		DBMaxIdleConnections: 5,
		DBConnMaxLifetime:    time.Hour,
		ServerHost:           "localhost",
		ServerPort:           8080,
		LogLevel:             "error",
		AuthTokenExpiration:  time.Hour,
	}

	// Create DI container
	container := app.NewContainer(cfg)

	// Initialize KEK
	kekUseCase, err := container.KekUseCase()
	require.NoError(t, err, "failed to get kek use case")

	err = kekUseCase.Create(context.Background(), masterKeyChain, cryptoDomain.AESGCM)
	require.NoError(t, err, "failed to create initial KEK")

	// Create root client with all capabilities
	clientUseCase, err := container.ClientUseCase()
	require.NoError(t, err, "failed to get client use case")

	rootClientInput := &authDomain.CreateClientInput{
		Name:     "Root Integration Test Client",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path: "*", // Wildcard access to all paths
				Capabilities: []authDomain.Capability{
					authDomain.ReadCapability,
					authDomain.WriteCapability,
					authDomain.DeleteCapability,
					authDomain.EncryptCapability,
					authDomain.DecryptCapability,
					authDomain.RotateCapability,
				},
			},
		},
	}

	rootClientOutput, err := clientUseCase.Create(context.Background(), rootClientInput)
	require.NoError(t, err, "failed to create root client")

	// Get the created client
	rootClient, err := clientUseCase.Get(context.Background(), rootClientOutput.ID)
	require.NoError(t, err, "failed to get root client")

	// Issue token for root client
	tokenUseCase, err := container.TokenUseCase()
	require.NoError(t, err, "failed to get token use case")

	issueTokenInput := &authDomain.IssueTokenInput{
		ClientID:     rootClientOutput.ID,
		ClientSecret: rootClientOutput.PlainSecret,
	}

	tokenOutput, err := tokenUseCase.Issue(context.Background(), issueTokenInput)
	require.NoError(t, err, "failed to issue token")

	// Setup HTTP server
	httpSrv, err := container.HTTPServer()
	require.NoError(t, err, "failed to get HTTP server")

	// Get the handler from the server
	// The SetupRouter has already been called by container.HTTPServer()
	handler := httpSrv.GetHandler()
	require.NotNil(t, handler, "handler should not be nil after SetupRouter")

	// Create test server with the handler
	testServer := httptest.NewServer(handler)

	t.Logf("Integration test setup complete for %s (client_id=%s)", dbDriver, rootClient.ID)

	return &integrationTestContext{
		container:      container,
		db:             db,
		server:         testServer,
		rootClient:     rootClient,
		rootToken:      tokenOutput.PlainToken,
		rootSecret:     rootClientOutput.PlainSecret,
		masterKeyChain: masterKeyChain,
		dbDriver:       dbDriver,
	}
}

// teardownIntegrationTest cleans up all resources.
func teardownIntegrationTest(t *testing.T, ctx *integrationTestContext) {
	t.Helper()

	if ctx.server != nil {
		ctx.server.Close()
	}

	if ctx.masterKeyChain != nil {
		ctx.masterKeyChain.Close()
	}

	if ctx.container != nil {
		err := ctx.container.Shutdown(context.Background())
		if err != nil {
			t.Logf("Warning: container shutdown error: %v", err)
		}
	}

	if ctx.db != nil {
		testutil.TeardownDB(t, ctx.db)
	}

	// Clean up environment variables
	if err := os.Unsetenv("MASTER_KEYS"); err != nil {
		t.Logf("Warning: failed to unset MASTER_KEYS: %v", err)
	}
	if err := os.Unsetenv("ACTIVE_MASTER_KEY_ID"); err != nil {
		t.Logf("Warning: failed to unset ACTIVE_MASTER_KEY_ID: %v", err)
	}

	t.Logf("Integration test teardown complete for %s", ctx.dbDriver)
}

// TestIntegration_AllEndpoints_HappyPath tests all API endpoints in a happy path scenario.
// This test validates the complete API functionality against both PostgreSQL and MySQL databases.
func TestIntegration_AllEndpoints_HappyPath(t *testing.T) {
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
				secretPath           = "/integration-test/password"
				secretPathStored     = "integration-test/password" // API stores without leading slash
				transitKeyName       = "integration-test-key"
				transitKeyID         uuid.UUID
				newClientID          uuid.UUID
				plaintextValue       = []byte("super-secret-value")
				plaintextValueBase64 = base64.StdEncoding.EncodeToString(plaintextValue)
				transitPlaintext     = []byte("transit-test-data")
				transitCiphertext    string
			)

			// [1/15] Test GET /health - Health check endpoint
			t.Run("01_Health", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/health", nil, false)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response map[string]string
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "healthy", response["status"])
			})

			// [2/15] Test POST /v1/token - Issue authentication token
			t.Run("02_IssueToken", func(t *testing.T) {
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

			// [3/15] Test POST /v1/secrets/*path - Create/update secret
			t.Run("03_CreateSecret", func(t *testing.T) {
				requestBody := secretsDTO.CreateOrUpdateSecretRequest{
					Value: plaintextValueBase64,
				}

				resp, body := ctx.makeRequest(t, http.MethodPost, "/v1/secrets"+secretPath, requestBody, true)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, secretPathStored, response.Path) // API stores without leading slash
				assert.Equal(t, uint(1), response.Version)
				assert.Empty(t, response.Value) // Value not returned on create
			})

			// [4/15] Test GET /v1/secrets/*path - Read secret
			t.Run("04_ReadSecret", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/secrets"+secretPath, nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response secretsDTO.SecretResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
				assert.Equal(t, secretPathStored, response.Path) // API stores without leading slash
				assert.Equal(t, uint(1), response.Version)
				assert.Equal(t, plaintextValueBase64, response.Value) // Value returned on read

				// Verify the value decodes correctly
				decoded, err := base64.StdEncoding.DecodeString(response.Value)
				require.NoError(t, err)
				assert.Equal(t, plaintextValue, decoded)
			})

			// [5/15] Test POST /v1/transit/keys - Create transit key
			t.Run("05_CreateTransitKey", func(t *testing.T) {
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

				// Store transit key ID for later deletion
				parsedID, err := uuid.Parse(response.ID)
				require.NoError(t, err)
				transitKeyID = parsedID
			})

			// [6/15] Test POST /v1/transit/keys/:name/encrypt - Encrypt with transit key
			t.Run("06_TransitEncrypt", func(t *testing.T) {
				requestBody := transitDTO.EncryptRequest{
					Plaintext: base64.StdEncoding.EncodeToString(transitPlaintext),
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
				transitCiphertext = response.Ciphertext

				// Verify ciphertext format: <version>:<base64>
				assert.Contains(t, response.Ciphertext, ":")
			})

			// [7/15] Test POST /v1/transit/keys/:name/decrypt - Decrypt with transit key
			t.Run("07_TransitDecrypt", func(t *testing.T) {
				requestBody := transitDTO.DecryptRequest{
					Ciphertext: transitCiphertext,
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
				assert.Equal(t, transitPlaintext, decoded)
			})

			// [8/15] Test POST /v1/clients - Create new client
			t.Run("08_CreateClient", func(t *testing.T) {
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

			// [9/15] Test GET /v1/clients/:id - Get client by ID
			t.Run("09_GetClient", func(t *testing.T) {
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

			// [10/15] Test PUT /v1/clients/:id - Update client
			t.Run("10_UpdateClient", func(t *testing.T) {
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
			})

			// [11/15] Test GET /v1/clients - List clients
			t.Run("11_ListClients", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/clients", nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ListClientsResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Clients)
				assert.GreaterOrEqual(t, len(response.Clients), 2) // At least root + new client
			})

			// [12/15] Test POST /v1/transit/keys/:name/rotate - Rotate transit key
			t.Run("12_RotateTransitKey", func(t *testing.T) {
				requestBody := transitDTO.RotateTransitKeyRequest{
					Algorithm: "aes-gcm",
				}

				resp, body := ctx.makeRequest(
					t,
					http.MethodPost,
					"/v1/transit/keys/"+transitKeyName+"/rotate",
					requestBody,
					true,
				)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, transitKeyName, response.Name)
				assert.Equal(t, uint(2), response.Version) // Version should increment
			})

			// [13/15] Test GET /v1/audit-logs - List audit logs
			t.Run("13_ListAuditLogs", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/v1/audit-logs", nil, true)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response authDTO.ListAuditLogsResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.AuditLogs)

				// Verify some audit log entries exist for our operations
				assert.GreaterOrEqual(t, len(response.AuditLogs), 5, "should have multiple audit log entries")

				// Verify audit log structure
				firstLog := response.AuditLogs[0]
				assert.NotEmpty(t, firstLog.ID)
				assert.NotEmpty(t, firstLog.ClientID)
				assert.NotEmpty(t, firstLog.Capability)
			})

			// [14/15] Test DELETE /v1/clients/:id - Delete client (soft delete)
			t.Run("14_DeleteClient", func(t *testing.T) {
				resp, body := ctx.makeRequest(
					t,
					http.MethodDelete,
					"/v1/clients/"+newClientID.String(),
					nil,
					true,
				)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, body)

				// Verify client is soft-deleted (IsActive = false)
				resp, body = ctx.makeRequest(
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

			// [15/15] Test DELETE /v1/transit/keys/:id - Delete transit key
			t.Run("15_DeleteTransitKey", func(t *testing.T) {
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

			t.Logf("All 15 endpoint tests passed for %s", tc.dbDriver)
		})
	}
}
