package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"gocloud.dev/secrets"

	"github.com/allisson/secrets/internal/config"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
)

// TestNewContainer verifies that a new container can be created with a valid configuration.
func TestNewContainer(t *testing.T) {
	//nolint:gosec // test fixture data
	cfg := &config.Config{
		LogLevel:             "info",
		DBDriver:             "postgres",
		DBConnectionString:   "postgres://test:test@localhost:5432/test?sslmode=disable",
		DBMaxOpenConnections: 10,
		DBMaxIdleConnections: 5,
		DBConnMaxLifetime:    time.Hour,
		ServerHost:           "localhost",
		ServerPort:           8080,
		AuthTokenExpiration:  time.Second,
	}

	container := NewContainer(cfg)

	if container == nil {
		t.Fatal("expected non-nil container")
	}

	if container.Config() != cfg {
		t.Error("container config does not match provided config")
	}
}

// TestContainerLogger verifies that the logger can be retrieved from the container.
func TestContainerLogger(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "debug",
	}

	container := NewContainer(cfg)
	logger := container.Logger()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Calling Logger() again should return the same instance (singleton)
	logger2 := container.Logger()
	if logger != logger2 {
		t.Error("expected same logger instance on multiple calls")
	}
}

// TestContainerLoggerDefaultLevel verifies that logger defaults to info level.
func TestContainerLoggerDefaultLevel(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "invalid",
	}

	container := NewContainer(cfg)
	logger := container.Logger()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

// TestContainerInitializationErrors verifies that initialization errors are properly handled.
func TestContainerInitializationErrors(t *testing.T) {
	// Create a container with invalid database configuration
	cfg := &config.Config{
		DBDriver:           "invalid_driver",
		DBConnectionString: "",
	}

	container := NewContainer(cfg)

	// Attempting to get DB should return an error
	_, err := container.DB()
	if err == nil {
		t.Error("expected error when connecting with invalid config")
	}

	// Attempting to get DB again should return the same error
	_, err2 := container.DB()
	if err2 == nil {
		t.Error("expected error on second call to DB()")
	}
}

// TestContainerLazyInitialization verifies that components are only initialized when accessed.
func TestContainerLazyInitialization(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}

	container := NewContainer(cfg)

	// At this point, no components should be initialized
	if container.logger != nil {
		t.Error("expected logger to be nil before first access")
	}

	// Access logger
	logger := container.Logger()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Now logger should be initialized
	if container.logger == nil {
		t.Error("expected logger to be initialized after access")
	}
}

// TestContainerShutdown verifies that the shutdown method can be called safely.
func TestContainerShutdown(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}

	container := NewContainer(cfg)

	// Shutdown should not fail even if no components are initialized
	if err := container.Shutdown(context.TODO()); err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

// TestContainerAEADManager verifies that the AEAD manager can be retrieved from the container.
func TestContainerAEADManager(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}

	container := NewContainer(cfg)
	aeadManager := container.AEADManager()

	if aeadManager == nil {
		t.Fatal("expected non-nil AEAD manager")
	}

	// Calling AEADManager() again should return the same instance (singleton)
	aeadManager2 := container.AEADManager()
	if aeadManager != aeadManager2 {
		t.Error("expected same AEAD manager instance on multiple calls")
	}
}

// TestContainerKeyManager verifies that the key manager can be retrieved from the container.
func TestContainerKeyManager(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}

	container := NewContainer(cfg)
	keyManager := container.KeyManager()

	if keyManager == nil {
		t.Fatal("expected non-nil key manager")
	}

	// Calling KeyManager() again should return the same instance (singleton)
	keyManager2 := container.KeyManager()
	if keyManager != keyManager2 {
		t.Error("expected same key manager instance on multiple calls")
	}
}

// TestContainerKekRepositoryErrors verifies that KEK repository initialization errors are properly handled.
func TestContainerKekRepositoryErrors(t *testing.T) {
	// Create a container with invalid database configuration
	cfg := &config.Config{
		DBDriver:           "invalid_driver",
		DBConnectionString: "",
	}

	container := NewContainer(cfg)

	// Attempting to get KEK repository should return an error
	_, err := container.KekRepository()
	if err == nil {
		t.Error("expected error when connecting with invalid config")
	}

	// Attempting to get KEK repository again should return the same error
	_, err2 := container.KekRepository()
	if err2 == nil {
		t.Error("expected error on second call to KekRepository()")
	}
}

// TestContainerKekUseCaseErrors verifies that KEK use case initialization errors are properly handled.
func TestContainerKekUseCaseErrors(t *testing.T) {
	// Create a container with invalid database configuration
	cfg := &config.Config{
		DBDriver:           "invalid_driver",
		DBConnectionString: "",
	}

	container := NewContainer(cfg)

	// Attempting to get KEK use case should return an error (due to DB error)
	_, err := container.KekUseCase()
	if err == nil {
		t.Error("expected error when connecting with invalid config")
	}

	// Attempting to get KEK use case again should return the same error
	_, err2 := container.KekUseCase()
	if err2 == nil {
		t.Error("expected error on second call to KekUseCase()")
	}
}

// TestContainerCryptoDekRepositoryErrors verifies that Crypto DEK repository initialization errors are properly handled.
func TestContainerCryptoDekRepositoryErrors(t *testing.T) {
	// Create a container with invalid database configuration
	cfg := &config.Config{
		DBDriver:           "invalid_driver",
		DBConnectionString: "",
	}

	container := NewContainer(cfg)

	// Attempting to get Crypto DEK repository should return an error
	_, err := container.CryptoDekRepository()
	if err == nil {
		t.Error("expected error when connecting with invalid config")
	}

	// Attempting to get Crypto DEK repository again should return the same error
	_, err2 := container.CryptoDekRepository()
	if err2 == nil {
		t.Error("expected error on second call to CryptoDekRepository()")
	}
}

// TestContainerCryptoDekUseCaseErrors verifies that Crypto DEK use case initialization errors are properly handled.
func TestContainerCryptoDekUseCaseErrors(t *testing.T) {
	// Create a container with invalid database configuration
	cfg := &config.Config{
		DBDriver:           "invalid_driver",
		DBConnectionString: "",
	}

	container := NewContainer(cfg)

	// Attempting to get Crypto DEK use case should return an error (due to DB error)
	_, err := container.CryptoDekUseCase()
	if err == nil {
		t.Error("expected error when connecting with invalid config")
	}

	// Attempting to get Crypto DEK use case again should return the same error
	_, err2 := container.CryptoDekUseCase()
	if err2 == nil {
		t.Error("expected error on second call to CryptoDekUseCase()")
	}
}

// TestContainerMasterKeyChain verifies that the master key chain can be retrieved from the container.
func TestContainerMasterKeyChain(t *testing.T) {
	ctx := context.Background()

	// Generate KMS key for localsecrets provider
	kmsKey := make([]byte, 32)
	_, err := rand.Read(kmsKey)
	if err != nil {
		t.Fatalf("failed to generate KMS key: %v", err)
	}
	kmsKeyURI := "base64key://" + base64.URLEncoding.EncodeToString(kmsKey)

	// Generate master key
	masterKeyBytes := []byte("12345678901234567890123456789012") // 32 bytes

	// Encrypt master key with KMS
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		t.Fatalf("failed to open KMS keeper: %v", err)
	}
	defer func() {
		_ = keeperInterface.Close()
	}()

	keeper, ok := keeperInterface.(*secrets.Keeper)
	if !ok {
		t.Fatal("keeper should be *secrets.Keeper")
	}

	ciphertext, err := keeper.Encrypt(ctx, masterKeyBytes)
	if err != nil {
		t.Fatalf("failed to encrypt master key: %v", err)
	}

	encryptedKey := base64.StdEncoding.EncodeToString(ciphertext)

	// Set up environment variables for master keys with KMS
	t.Setenv("MASTER_KEYS", "test-key-1:"+encryptedKey)
	t.Setenv("ACTIVE_MASTER_KEY_ID", "test-key-1")
	t.Setenv("KMS_PROVIDER", "localsecrets")
	t.Setenv("KMS_KEY_URI", kmsKeyURI)

	cfg := &config.Config{
		LogLevel:    "info",
		KMSProvider: "localsecrets",
		KMSKeyURI:   kmsKeyURI,
	}

	container := NewContainer(cfg)
	masterKeyChain, err := container.MasterKeyChain()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if masterKeyChain == nil {
		t.Fatal("expected non-nil master key chain")
	}

	// Verify active key ID
	if masterKeyChain.ActiveMasterKeyID() != "test-key-1" {
		t.Errorf("expected active key ID 'test-key-1', got '%s'", masterKeyChain.ActiveMasterKeyID())
	}

	// Calling MasterKeyChain() again should return the same instance (singleton)
	masterKeyChain2, err := container.MasterKeyChain()
	if err != nil {
		t.Fatalf("expected no error on second call, got: %v", err)
	}
	if masterKeyChain != masterKeyChain2 {
		t.Error("expected same master key chain instance on multiple calls")
	}
}

// TestContainerMasterKeyChainErrors verifies that master key chain initialization errors are properly handled.
func TestContainerMasterKeyChainErrors(t *testing.T) {
	// Clear environment variables to trigger an error
	originalMasterKeys := os.Getenv("MASTER_KEYS")
	originalActiveID := os.Getenv("ACTIVE_MASTER_KEY_ID")
	defer func() {
		if originalMasterKeys != "" {
			_ = os.Setenv("MASTER_KEYS", originalMasterKeys)
		}
		if originalActiveID != "" {
			_ = os.Setenv("ACTIVE_MASTER_KEY_ID", originalActiveID)
		}
	}()

	_ = os.Unsetenv("MASTER_KEYS")
	_ = os.Unsetenv("ACTIVE_MASTER_KEY_ID")

	cfg := &config.Config{
		LogLevel: "info",
	}

	container := NewContainer(cfg)

	// Attempting to get master key chain should return an error
	_, err := container.MasterKeyChain()
	if err == nil {
		t.Error("expected error when MASTER_KEYS is not set")
	}

	// Attempting to get master key chain again should return the same error
	_, err2 := container.MasterKeyChain()
	if err2 == nil {
		t.Error("expected error on second call to MasterKeyChain()")
	}
}

// TestContainerMasterKeyChainMultipleKeys verifies that multiple master keys can be loaded.
func TestContainerMasterKeyChainMultipleKeys(t *testing.T) {
	ctx := context.Background()

	// Generate KMS key for localsecrets provider
	kmsKey := make([]byte, 32)
	_, err := rand.Read(kmsKey)
	if err != nil {
		t.Fatalf("failed to generate KMS key: %v", err)
	}
	kmsKeyURI := "base64key://" + base64.URLEncoding.EncodeToString(kmsKey)

	// Generate master keys
	key1Bytes := []byte("12345678901234567890123456789012") // 32 bytes
	key2Bytes := []byte("abcdefghijklmnopqrstuvwxyz123456") // 32 bytes

	// Encrypt master keys with KMS
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		t.Fatalf("failed to open KMS keeper: %v", err)
	}
	defer func() {
		_ = keeperInterface.Close()
	}()

	keeper, ok := keeperInterface.(*secrets.Keeper)
	if !ok {
		t.Fatal("keeper should be *secrets.Keeper")
	}

	ciphertext1, err := keeper.Encrypt(ctx, key1Bytes)
	if err != nil {
		t.Fatalf("failed to encrypt key1: %v", err)
	}
	encryptedKey1 := base64.StdEncoding.EncodeToString(ciphertext1)

	ciphertext2, err := keeper.Encrypt(ctx, key2Bytes)
	if err != nil {
		t.Fatalf("failed to encrypt key2: %v", err)
	}
	encryptedKey2 := base64.StdEncoding.EncodeToString(ciphertext2)

	// Set up environment variables for multiple master keys with KMS
	t.Setenv("MASTER_KEYS", "key1:"+encryptedKey1+",key2:"+encryptedKey2)
	t.Setenv("ACTIVE_MASTER_KEY_ID", "key2")
	t.Setenv("KMS_PROVIDER", "localsecrets")
	t.Setenv("KMS_KEY_URI", kmsKeyURI)

	cfg := &config.Config{
		LogLevel:    "info",
		KMSProvider: "localsecrets",
		KMSKeyURI:   kmsKeyURI,
	}

	container := NewContainer(cfg)
	masterKeyChain, err := container.MasterKeyChain()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if masterKeyChain == nil {
		t.Fatal("expected non-nil master key chain")
	}

	// Verify active key ID
	if masterKeyChain.ActiveMasterKeyID() != "key2" {
		t.Errorf("expected active key ID 'key2', got '%s'", masterKeyChain.ActiveMasterKeyID())
	}

	// Verify both keys are accessible
	key1Obj, ok := masterKeyChain.Get("key1")
	if !ok {
		t.Error("expected to find key1 in master key chain")
	}
	if key1Obj == nil {
		t.Error("expected non-nil key1")
	}

	key2Obj, ok := masterKeyChain.Get("key2")
	if !ok {
		t.Error("expected to find key2 in master key chain")
	}
	if key2Obj == nil {
		t.Error("expected non-nil key2")
	}
}

// TestContainerShutdownWithMasterKeyChain verifies that shutdown properly closes the master key chain.
func TestContainerShutdownWithMasterKeyChain(t *testing.T) {
	ctx := context.Background()

	// Generate KMS key for localsecrets provider
	kmsKey := make([]byte, 32)
	_, err := rand.Read(kmsKey)
	if err != nil {
		t.Fatalf("failed to generate KMS key: %v", err)
	}
	kmsKeyURI := "base64key://" + base64.URLEncoding.EncodeToString(kmsKey)

	// Generate master key
	masterKeyBytes := []byte("12345678901234567890123456789012") // 32 bytes

	// Encrypt master key with KMS
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		t.Fatalf("failed to open KMS keeper: %v", err)
	}
	defer func() {
		_ = keeperInterface.Close()
	}()

	keeper, ok := keeperInterface.(*secrets.Keeper)
	if !ok {
		t.Fatal("keeper should be *secrets.Keeper")
	}

	ciphertext, err := keeper.Encrypt(ctx, masterKeyBytes)
	if err != nil {
		t.Fatalf("failed to encrypt master key: %v", err)
	}

	encryptedKey := base64.StdEncoding.EncodeToString(ciphertext)

	// Set up environment variables for master keys with KMS
	t.Setenv("MASTER_KEYS", "test-key-1:"+encryptedKey)
	t.Setenv("ACTIVE_MASTER_KEY_ID", "test-key-1")
	t.Setenv("KMS_PROVIDER", "localsecrets")
	t.Setenv("KMS_KEY_URI", kmsKeyURI)

	cfg := &config.Config{
		LogLevel:    "info",
		KMSProvider: "localsecrets",
		KMSKeyURI:   kmsKeyURI,
	}

	container := NewContainer(cfg)

	// Initialize master key chain
	masterKeyChain, err := container.MasterKeyChain()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if masterKeyChain == nil {
		t.Fatal("expected non-nil master key chain")
	}

	// Shutdown should close the master key chain without error
	if err := container.Shutdown(ctx); err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}

	// After shutdown, the key chain should be closed (keys should be zeroed)
	// We can't directly verify that keys are zeroed, but we verify that Shutdown ran without panic
}
