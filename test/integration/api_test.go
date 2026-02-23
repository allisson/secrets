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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/secrets"
	_ "gocloud.dev/secrets/localsecrets"

	"github.com/allisson/secrets/internal/app"
	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authDTO "github.com/allisson/secrets/internal/auth/http/dto"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	secretsDTO "github.com/allisson/secrets/internal/secrets/http/dto"
	"github.com/allisson/secrets/internal/testutil"
	tokenizationDTO "github.com/allisson/secrets/internal/tokenization/http/dto"
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
	kmsKeyURI      string
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
	//nolint:gosec // controlled test environment with localhost URLs
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

// generateLocalSecretsKMSKey creates a random 32-byte key and returns a base64key:// URI for testing.
func generateLocalSecretsKMSKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err, "failed to generate KMS key")
	return "base64key://" + base64.URLEncoding.EncodeToString(key)
}

// createMasterKeyChainWithKMS creates a master key chain with KMS-encrypted master keys.
func createMasterKeyChainWithKMS(
	ctx context.Context,
	t *testing.T,
	masterKey *cryptoDomain.MasterKey,
	kmsKeyURI string,
) *cryptoDomain.MasterKeyChain {
	t.Helper()

	// Open KMS keeper
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	require.NoError(t, err, "failed to open KMS keeper")
	defer func() {
		assert.NoError(t, keeperInterface.Close())
	}()

	// Type assert to get Encrypt method
	keeper, ok := keeperInterface.(*secrets.Keeper)
	require.True(t, ok, "keeper should be *secrets.Keeper")

	// Encrypt master key with KMS
	ciphertext, err := keeper.Encrypt(ctx, masterKey.Key)
	require.NoError(t, err, "failed to encrypt master key with KMS")

	// Encode ciphertext to base64
	encodedCiphertext := base64.StdEncoding.EncodeToString(ciphertext)

	// Set environment variables
	err = os.Setenv("MASTER_KEYS", fmt.Sprintf("%s:%s", masterKey.ID, encodedCiphertext))
	require.NoError(t, err, "failed to set MASTER_KEYS env")

	err = os.Setenv("ACTIVE_MASTER_KEY_ID", masterKey.ID)
	require.NoError(t, err, "failed to set ACTIVE_MASTER_KEY_ID env")

	err = os.Setenv("KMS_PROVIDER", "localsecrets")
	require.NoError(t, err, "failed to set KMS_PROVIDER env")

	err = os.Setenv("KMS_KEY_URI", kmsKeyURI)
	require.NoError(t, err, "failed to set KMS_KEY_URI env")

	// Load master key chain using KMS
	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   kmsKeyURI,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	chain, err := cryptoDomain.LoadMasterKeyChain(ctx, cfg, kmsService, logger)
	require.NoError(t, err, "failed to load master key chain from KMS")

	return chain
}

// setupIntegrationTestWithKMS initializes all components for integration testing with KMS-encrypted master keys.
func setupIntegrationTestWithKMS(t *testing.T, dbDriver string) *integrationTestContext {
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

	// Generate KMS key URI and ephemeral master key
	kmsKeyURI := generateLocalSecretsKMSKey(t)
	masterKey := generateMasterKey()
	masterKeyChain := createMasterKeyChainWithKMS(context.Background(), t, masterKey, kmsKeyURI)

	// Create configuration with KMS settings
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
		KMSProvider:          "localsecrets",
		KMSKeyURI:            kmsKeyURI,
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
		Name:     "Root Integration Test Client (KMS)",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path: "*",
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

	handler := httpSrv.GetHandler()
	require.NotNil(t, handler, "handler should not be nil after SetupRouter")

	testServer := httptest.NewServer(handler)

	t.Logf("Integration test setup complete for %s with KMS (client_id=%s)", dbDriver, rootClient.ID)

	return &integrationTestContext{
		container:      container,
		db:             db,
		server:         testServer,
		rootClient:     rootClient,
		rootToken:      tokenOutput.PlainToken,
		rootSecret:     rootClientOutput.PlainSecret,
		masterKeyChain: masterKeyChain,
		dbDriver:       dbDriver,
		kmsKeyURI:      kmsKeyURI,
	}
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
	if err := os.Unsetenv("KMS_PROVIDER"); err != nil {
		t.Logf("Warning: failed to unset KMS_PROVIDER: %v", err)
	}
	if err := os.Unsetenv("KMS_KEY_URI"); err != nil {
		t.Logf("Warning: failed to unset KMS_KEY_URI: %v", err)
	}

	t.Logf("Integration test teardown complete for %s", ctx.dbDriver)
}

// TestIntegration_Health_BasicChecks validates infrastructure health and readiness endpoints.
// Tests health check and database connectivity verification against both PostgreSQL and MySQL.
func TestIntegration_Health_BasicChecks(t *testing.T) {
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

			// [1/2] Test GET /health - Health check endpoint
			t.Run("01_HealthCheck", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/health", nil, false)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response map[string]string
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "healthy", response["status"])
			})

			// [2/2] Test GET /ready - Readiness check endpoint
			t.Run("02_ReadinessCheck", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/ready", nil, false)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response map[string]string
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "ready", response["status"])
			})

			t.Logf("All 2 health endpoint tests passed for %s", tc.dbDriver)
		})
	}
}

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
				assert.NotEmpty(t, response.Clients)
				assert.GreaterOrEqual(t, len(response.Clients), 2, "should have at least root + new client")
			})

			// [6/8] Test GET /v1/audit-logs - List audit logs
			t.Run("06_ListAuditLogs", func(t *testing.T) {
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

			t.Logf("All 8 auth endpoint tests passed for %s", tc.dbDriver)
		})
	}
}

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

			// [2/8] Test POST /v1/transit/keys/:name/encrypt - Encrypt with transit key
			t.Run("02_Encrypt", func(t *testing.T) {
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
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response transitDTO.TransitKeyResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, transitKeyName, response.Name)
				assert.Equal(t, uint(2), response.Version) // Version should increment to 2
			})

			// [6/8] Test POST /v1/transit/keys/:name/encrypt - Encrypt with rotated key (version 2)
			t.Run("06_EncryptWithRotatedKey", func(t *testing.T) {
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

			// [7/8] Test POST /v1/transit/keys/:name/decrypt - Decrypt old ciphertext (backward compatibility)
			t.Run("07_DecryptOldCiphertext", func(t *testing.T) {
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

			// [8/8] Test DELETE /v1/transit/keys/:id - Delete transit key
			t.Run("08_DeleteTransitKey", func(t *testing.T) {
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

			t.Logf("All 8 transit endpoint tests passed for %s", tc.dbDriver)
		})
	}
}

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

				kekUseCase, err := ctx.container.KekUseCase()
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

// setupIntegrationTestWithLockout initializes all components for integration testing with account lockout enabled.
func setupIntegrationTestWithLockout(
	t *testing.T,
	dbDriver string,
	maxAttempts int,
	lockoutDuration time.Duration,
) *integrationTestContext {
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

	// Create configuration with lockout settings
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
		LockoutMaxAttempts:   maxAttempts,
		LockoutDuration:      lockoutDuration,
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
		Name:     "Root Integration Test Client (Lockout)",
		IsActive: true,
		Policies: []authDomain.PolicyDocument{
			{
				Path: "*",
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

	handler := httpSrv.GetHandler()
	require.NotNil(t, handler, "handler should not be nil after SetupRouter")

	testServer := httptest.NewServer(handler)

	t.Logf("Integration test setup complete for %s with lockout (max_attempts=%d, client_id=%s)",
		dbDriver, maxAttempts, rootClient.ID)

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

// TestIntegration_AccountLockout_CompleteFlow tests the full lockout → unlock → re-auth cycle.
// Validates PCI DSS 8.3.4 account lockout enforcement and admin unlock capability against both databases.
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
