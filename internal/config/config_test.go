package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name:    "load default configuration",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.0.0.0", cfg.ServerHost)
				assert.Equal(t, 8080, cfg.ServerPort)
				assert.Equal(t, "postgres", cfg.DBDriver)
				assert.Equal(
					t,
					"postgres://user:password@localhost:5432/mydb?sslmode=disable",
					cfg.DBConnectionString,
				)
				assert.Equal(t, 25, cfg.DBMaxOpenConnections)
				assert.Equal(t, 5, cfg.DBMaxIdleConnections)
				assert.Equal(t, 5*time.Minute, cfg.DBConnMaxLifetime)
				assert.Equal(t, "info", cfg.LogLevel)
				assert.Equal(t, 14400*time.Second, cfg.AuthTokenExpiration)
				assert.Equal(t, true, cfg.RateLimitEnabled)
				assert.Equal(t, 10.0, cfg.RateLimitRequestsPerSec)
				assert.Equal(t, 20, cfg.RateLimitBurst)
				assert.Equal(t, false, cfg.CORSEnabled)
				assert.Equal(t, "", cfg.CORSAllowOrigins)
				assert.Equal(t, true, cfg.MetricsEnabled)
				assert.Equal(t, "secrets", cfg.MetricsNamespace)
			},
		},
		{
			name: "load custom server configuration",
			envVars: map[string]string{
				"SERVER_HOST": "localhost",
				"SERVER_PORT": "9090",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "localhost", cfg.ServerHost)
				assert.Equal(t, 9090, cfg.ServerPort)
			},
		},
		{
			name: "load custom database configuration",
			envVars: map[string]string{
				"DB_DRIVER":               "mysql",
				"DB_CONNECTION_STRING":    "user:password@tcp(localhost:3306)/testdb",
				"DB_MAX_OPEN_CONNECTIONS": "50",
				"DB_MAX_IDLE_CONNECTIONS": "10",
				"DB_CONN_MAX_LIFETIME":    "10",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "mysql", cfg.DBDriver)
				assert.Equal(t, "user:password@tcp(localhost:3306)/testdb", cfg.DBConnectionString)
				assert.Equal(t, 50, cfg.DBMaxOpenConnections)
				assert.Equal(t, 10, cfg.DBMaxIdleConnections)
				assert.Equal(t, 10*time.Minute, cfg.DBConnMaxLifetime)
			},
		},
		{
			name: "load custom auth configuration",
			envVars: map[string]string{
				"AUTH_TOKEN_EXPIRATION_SECONDS": "10",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 10*time.Second, cfg.AuthTokenExpiration)
			},
		},
		{
			name: "load custom log level",
			envVars: map[string]string{
				"LOG_LEVEL": "debug",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "debug", cfg.LogLevel)
			},
		},
		{
			name: "load custom rate limit configuration",
			envVars: map[string]string{
				"RATE_LIMIT_ENABLED":          "false",
				"RATE_LIMIT_REQUESTS_PER_SEC": "5.0",
				"RATE_LIMIT_BURST":            "10",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, false, cfg.RateLimitEnabled)
				assert.Equal(t, 5.0, cfg.RateLimitRequestsPerSec)
				assert.Equal(t, 10, cfg.RateLimitBurst)
			},
		},
		{
			name: "load custom CORS configuration",
			envVars: map[string]string{
				"CORS_ENABLED":       "true",
				"CORS_ALLOW_ORIGINS": "https://example.com,https://app.example.com",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, true, cfg.CORSEnabled)
				assert.Equal(t, "https://example.com,https://app.example.com", cfg.CORSAllowOrigins)
			},
		},
		{
			name: "load custom metrics configuration",
			envVars: map[string]string{
				"METRICS_ENABLED":   "false",
				"METRICS_NAMESPACE": "custom",
				"METRICS_PORT":      "9091",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, false, cfg.MetricsEnabled)
				assert.Equal(t, "custom", cfg.MetricsNamespace)
				assert.Equal(t, 9091, cfg.MetricsPort)
			},
		},
		{
			name: "load custom rate limit token configuration",
			envVars: map[string]string{
				"RATE_LIMIT_TOKEN_ENABLED":          "false",
				"RATE_LIMIT_TOKEN_REQUESTS_PER_SEC": "2.5",
				"RATE_LIMIT_TOKEN_BURST":            "5",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, false, cfg.RateLimitTokenEnabled)
				assert.Equal(t, 2.5, cfg.RateLimitTokenRequestsPerSec)
				assert.Equal(t, 5, cfg.RateLimitTokenBurst)
			},
		},
		{
			name: "load custom KMS configuration",
			envVars: map[string]string{
				"KMS_PROVIDER": "google",
				"KMS_KEY_URI":  "gcpkms://projects/my-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "google", cfg.KMSProvider)
				assert.Equal(
					t,
					"gcpkms://projects/my-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
					cfg.KMSKeyURI,
				)
			},
		},
		{
			name: "load custom lockout configuration",
			envVars: map[string]string{
				"LOCKOUT_MAX_ATTEMPTS":     "5",
				"LOCKOUT_DURATION_MINUTES": "15",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 5, cfg.LockoutMaxAttempts)
				assert.Equal(t, 15*time.Minute, cfg.LockoutDuration)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				err := os.Setenv(key, value)
				require.NoError(t, err)
			}

			// Load configuration
			cfg := Load()

			// Validate
			tt.validate(t, cfg)
		})
	}
}

func TestGetGinMode(t *testing.T) {
	tests := []struct {
		logLevel string
		expected string
	}{
		{"debug", "debug"},
		{"info", "release"},
		{"warn", "release"},
		{"error", "release"},
		{"fatal", "release"},
		{"panic", "release"},
		{"unknown", "release"},
		{"", "release"},
	}

	for _, tt := range tests {
		t.Run(tt.logLevel, func(t *testing.T) {
			cfg := &Config{LogLevel: tt.logLevel}
			assert.Equal(t, tt.expected, cfg.GetGinMode())
		})
	}
}

func TestLoadDotEnv(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a .env file in the temp root
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("TEST_ENV_VAR=found"), 0600)
	require.NoError(t, err)

	// Create a child directory
	childDir := filepath.Join(tmpDir, "child", "grandchild")
	err = os.MkdirAll(childDir, 0700)
	require.NoError(t, err)

	// Change working directory to childDir
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	err = os.Chdir(childDir)
	require.NoError(t, err)

	// Load .env
	loadDotEnv()

	// Verify the env var was loaded
	assert.Equal(t, "found", os.Getenv("TEST_ENV_VAR"))
	err = os.Unsetenv("TEST_ENV_VAR")
	require.NoError(t, err)
}
