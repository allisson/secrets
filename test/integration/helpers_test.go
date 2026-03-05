//go:build integration

// Package integration provides shared helper functions and utilities for integration tests.
// This file contains test context, setup/teardown functions, and key generation helpers
// used across all integration test files.
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/secrets"
	_ "gocloud.dev/secrets/localsecrets"

	"github.com/allisson/secrets/internal/app"
	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	"github.com/allisson/secrets/internal/testutil"
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

// createMasterKeyChain creates a master key chain with KMS encryption using localsecrets provider.
func createMasterKeyChain(masterKey *cryptoDomain.MasterKey) *cryptoDomain.MasterKeyChain {
	ctx := context.Background()

	// Generate a random KMS key for localsecrets provider
	kmsKey := make([]byte, 32)
	if _, err := rand.Read(kmsKey); err != nil {
		panic(fmt.Sprintf("failed to generate KMS key: %v", err))
	}
	kmsKeyURI := "base64key://" + base64.URLEncoding.EncodeToString(kmsKey)

	// Open KMS keeper
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		panic(fmt.Sprintf("failed to open KMS keeper: %v", err))
	}
	defer func() {
		_ = keeperInterface.Close()
	}()

	// Type assert to get Encrypt method
	keeper, ok := keeperInterface.(*secrets.Keeper)
	if !ok {
		panic("keeper should be *secrets.Keeper")
	}

	// Encrypt master key with KMS
	ciphertext, err := keeper.Encrypt(ctx, masterKey.Key)
	if err != nil {
		panic(fmt.Sprintf("failed to encrypt master key with KMS: %v", err))
	}

	// Encode ciphertext to base64
	encodedCiphertext := base64.StdEncoding.EncodeToString(ciphertext)

	// Set environment variables
	if err := os.Setenv("MASTER_KEYS", fmt.Sprintf("%s:%s", masterKey.ID, encodedCiphertext)); err != nil {
		panic(fmt.Sprintf("failed to set MASTER_KEYS env: %v", err))
	}
	if err := os.Setenv("ACTIVE_MASTER_KEY_ID", masterKey.ID); err != nil {
		panic(fmt.Sprintf("failed to set ACTIVE_MASTER_KEY_ID env: %v", err))
	}
	if err := os.Setenv("KMS_PROVIDER", "localsecrets"); err != nil {
		panic(fmt.Sprintf("failed to set KMS_PROVIDER env: %v", err))
	}
	if err := os.Setenv("KMS_KEY_URI", kmsKeyURI); err != nil {
		panic(fmt.Sprintf("failed to set KMS_KEY_URI env: %v", err))
	}

	// Load master key chain using KMS
	cfg := &config.Config{
		KMSProvider:               "localsecrets",
		KMSKeyURI:                 kmsKeyURI,
		SecretValueSizeLimitBytes: 1024 * 1024,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	chain, err := cryptoDomain.LoadMasterKeyChain(ctx, cfg, kmsService, logger)
	if err != nil {
		panic(fmt.Sprintf("failed to load master key chain: %v", err))
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
		KMSProvider:               "localsecrets",
		KMSKeyURI:                 kmsKeyURI,
		SecretValueSizeLimitBytes: 1024 * 1024,
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
		dsn = testutil.GetPostgresTestDSN()
	} else {
		db = testutil.SetupMySQLDB(t)
		dsn = testutil.GetMySQLTestDSN()
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
		MaxRequestBodySize:   1024 * 1024,
		SecretValueSizeLimitBytes: 1024 * 1024,
		KMSProvider:          "localsecrets",
		KMSKeyURI:            kmsKeyURI,
	}

	// Create DI container
	container := app.NewContainer(cfg)

	// Initialize KEK
	kekUseCase, err := container.KekUseCase(context.Background())
	require.NoError(t, err, "failed to get kek use case")

	err = kekUseCase.Create(context.Background(), masterKeyChain, cryptoDomain.AESGCM)
	require.NoError(t, err, "failed to create initial KEK")

	// Create root client with all capabilities
	clientUseCase, err := container.ClientUseCase(context.Background())
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
	tokenUseCase, err := container.TokenUseCase(context.Background())
	require.NoError(t, err, "failed to get token use case")

	issueTokenInput := &authDomain.IssueTokenInput{
		ClientID:     rootClientOutput.ID,
		ClientSecret: rootClientOutput.PlainSecret,
	}

	tokenOutput, err := tokenUseCase.Issue(context.Background(), issueTokenInput)
	require.NoError(t, err, "failed to issue token")

	// Setup HTTP server
	httpSrv, err := container.HTTPServer(context.Background())
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
		dsn = testutil.GetPostgresTestDSN()
	} else {
		db = testutil.SetupMySQLDB(t)
		dsn = testutil.GetMySQLTestDSN()
	}

	// Generate KMS key URI and ephemeral master key for testing
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
		MaxRequestBodySize:   1024 * 1024,
		SecretValueSizeLimitBytes: 1024 * 1024,
		KMSProvider:          "localsecrets",
		KMSKeyURI:            kmsKeyURI,
	}

	// Create DI container
	container := app.NewContainer(cfg)

	// Initialize KEK
	kekUseCase, err := container.KekUseCase(context.Background())
	require.NoError(t, err, "failed to get kek use case")

	err = kekUseCase.Create(context.Background(), masterKeyChain, cryptoDomain.AESGCM)
	require.NoError(t, err, "failed to create initial KEK")

	// Create root client with all capabilities
	clientUseCase, err := container.ClientUseCase(context.Background())
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
	tokenUseCase, err := container.TokenUseCase(context.Background())
	require.NoError(t, err, "failed to get token use case")

	issueTokenInput := &authDomain.IssueTokenInput{
		ClientID:     rootClientOutput.ID,
		ClientSecret: rootClientOutput.PlainSecret,
	}

	tokenOutput, err := tokenUseCase.Issue(context.Background(), issueTokenInput)
	require.NoError(t, err, "failed to issue token")

	// Setup HTTP server
	httpSrv, err := container.HTTPServer(context.Background())
	require.NoError(t, err, "failed to get HTTP server")

	// Get the handler from the server
	// The SetupRouter has already been called by container.HTTPServer(context.Background())
	handler := httpSrv.GetHandler()
	require.NotNil(t, handler, "handler should not be nil after SetupRouter")

	// Create test server with the handler
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

// setupIntegrationTestWithTokenExpiration initializes test context with custom token expiration.
// Useful for testing token expiry behavior without long wait times.
func setupIntegrationTestWithTokenExpiration(
	t *testing.T,
	dbDriver string,
	tokenExpiration time.Duration,
) *integrationTestContext {
	t.Helper()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup database
	var db *sql.DB
	var dsn string
	if dbDriver == "postgres" {
		db = testutil.SetupPostgresDB(t)
		dsn = testutil.GetPostgresTestDSN()
	} else {
		db = testutil.SetupMySQLDB(t)
		dsn = testutil.GetMySQLTestDSN()
	}

	// Generate KMS key URI and ephemeral master key for testing
	kmsKeyURI := generateLocalSecretsKMSKey(t)
	masterKey := generateMasterKey()
	masterKeyChain := createMasterKeyChainWithKMS(context.Background(), t, masterKey, kmsKeyURI)

	// Create configuration with KMS settings and CUSTOM token expiration
	cfg := &config.Config{
		DBDriver:             dbDriver,
		DBConnectionString:   dsn,
		DBMaxOpenConnections: 10,
		DBMaxIdleConnections: 5,
		DBConnMaxLifetime:    time.Hour,
		ServerHost:           "localhost",
		ServerPort:           8080,
		LogLevel:             "error",
		AuthTokenExpiration:  tokenExpiration, // Custom expiration
		MaxRequestBodySize:   1024 * 1024,
		SecretValueSizeLimitBytes: 1024 * 1024,
		KMSProvider:          "localsecrets",
		KMSKeyURI:            kmsKeyURI,
	}

	// Create DI container
	container := app.NewContainer(cfg)

	// Initialize KEK
	kekUseCase, err := container.KekUseCase(context.Background())
	require.NoError(t, err, "failed to get kek use case")

	err = kekUseCase.Create(context.Background(), masterKeyChain, cryptoDomain.AESGCM)
	require.NoError(t, err, "failed to create initial KEK")

	// Create root client with all capabilities
	clientUseCase, err := container.ClientUseCase(context.Background())
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
	tokenUseCase, err := container.TokenUseCase(context.Background())
	require.NoError(t, err, "failed to get token use case")

	issueTokenInput := &authDomain.IssueTokenInput{
		ClientID:     rootClientOutput.ID,
		ClientSecret: rootClientOutput.PlainSecret,
	}

	tokenOutput, err := tokenUseCase.Issue(context.Background(), issueTokenInput)
	require.NoError(t, err, "failed to issue token")

	// Setup HTTP server
	httpSrv, err := container.HTTPServer(context.Background())
	require.NoError(t, err, "failed to get HTTP server")

	// Get the handler from the server
	handler := httpSrv.GetHandler()
	require.NotNil(t, handler, "handler should not be nil after SetupRouter")

	// Create test server with the handler
	testServer := httptest.NewServer(handler)

	t.Logf("Integration test setup complete for %s with custom token expiration %v (client_id=%s)",
		dbDriver, tokenExpiration, rootClient.ID)

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
		dsn = testutil.GetPostgresTestDSN()
	} else {
		db = testutil.SetupMySQLDB(t)
		dsn = testutil.GetMySQLTestDSN()
	}

	// Generate KMS key URI and ephemeral master key for testing
	kmsKeyURI := generateLocalSecretsKMSKey(t)
	masterKey := generateMasterKey()
	masterKeyChain := createMasterKeyChainWithKMS(context.Background(), t, masterKey, kmsKeyURI)

	// Create configuration with lockout settings and KMS
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
		MaxRequestBodySize:   1024 * 1024,
		SecretValueSizeLimitBytes: 1024 * 1024,
		KMSProvider:          "localsecrets",
		KMSKeyURI:            kmsKeyURI,
	}

	// Create DI container
	container := app.NewContainer(cfg)

	// Initialize KEK
	kekUseCase, err := container.KekUseCase(context.Background())
	require.NoError(t, err, "failed to get kek use case")

	err = kekUseCase.Create(context.Background(), masterKeyChain, cryptoDomain.AESGCM)
	require.NoError(t, err, "failed to create initial KEK")

	// Create root client with all capabilities
	clientUseCase, err := container.ClientUseCase(context.Background())
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
	tokenUseCase, err := container.TokenUseCase(context.Background())
	require.NoError(t, err, "failed to get token use case")

	issueTokenInput := &authDomain.IssueTokenInput{
		ClientID:     rootClientOutput.ID,
		ClientSecret: rootClientOutput.PlainSecret,
	}

	tokenOutput, err := tokenUseCase.Issue(context.Background(), issueTokenInput)
	require.NoError(t, err, "failed to issue token")

	// Setup HTTP server
	httpSrv, err := container.HTTPServer(context.Background())
	require.NoError(t, err, "failed to get HTTP server")

	handler := httpSrv.GetHandler()
	require.NotNil(t, handler, "handler should not be nil after SetupRouter")

	testServer := httptest.NewServer(handler)

	t.Logf("Integration test setup complete for %s with lockout and KMS (max_attempts=%d, client_id=%s)",
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
		kmsKeyURI:      kmsKeyURI,
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
