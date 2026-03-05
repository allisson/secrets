package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				DBDriver:                     "postgres",
				DBConnectionString:           "postgres://localhost",
				ServerPort:                   8080,
				MetricsPort:                  8081,
				LogLevel:                     "info",
				ServerReadTimeout:            15 * time.Second,
				ServerWriteTimeout:           15 * time.Second,
				ServerIdleTimeout:            60 * time.Second,
				RateLimitEnabled:             true,
				RateLimitRequestsPerSec:      10,
				RateLimitTokenEnabled:        true,
				RateLimitTokenRequestsPerSec: 5,
				MaxRequestBodySize:           1048576,
				SecretValueSizeLimitBytes:    524288,
			},
			wantErr: false,
		},
		{
			name: "invalid secret value size limit - zero",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid db driver",
			cfg: &Config{
				DBDriver:           "sqlite",
				DBConnectionString: "postgres://localhost",
				ServerPort:         8080,
				MetricsPort:        8081,
				LogLevel:           "info",
			},
			wantErr: true,
		},
		{
			name: "missing db connection string",
			cfg: &Config{
				DBDriver:           "postgres",
				DBConnectionString: "",
				ServerPort:         8080,
				MetricsPort:        8081,
				LogLevel:           "info",
			},
			wantErr: true,
		},
		{
			name: "invalid server port",
			cfg: &Config{
				DBDriver:           "postgres",
				DBConnectionString: "postgres://localhost",
				ServerPort:         70000,
				MetricsPort:        8081,
				LogLevel:           "info",
			},
			wantErr: true,
		},
		{
			name: "conflicting ports",
			cfg: &Config{
				DBDriver:           "postgres",
				DBConnectionString: "postgres://localhost",
				ServerPort:         8080,
				MetricsPort:        8080,
				LogLevel:           "info",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				DBDriver:           "postgres",
				DBConnectionString: "postgres://localhost",
				ServerPort:         8080,
				MetricsPort:        8081,
				LogLevel:           "trace",
			},
			wantErr: true,
		},
		{
			name: "missing KMS provider when key URI is present",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSKeyURI:                 "gcpkms://...",
			},
			wantErr: true,
		},
		{
			name: "missing KMS key URI when provider is present",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "google",
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit requests",
			cfg: &Config{
				DBDriver:                "postgres",
				DBConnectionString:      "postgres://localhost",
				ServerPort:              8080,
				MetricsPort:             8081,
				LogLevel:                "info",
				ServerReadTimeout:       15 * time.Second,
				ServerWriteTimeout:      15 * time.Second,
				ServerIdleTimeout:       60 * time.Second,
				RateLimitEnabled:        true,
				RateLimitRequestsPerSec: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid KMS provider name",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "invalid_provider",
				KMSKeyURI:                 "invalid://key",
			},
			wantErr: true,
		},
		{
			name: "valid KMS provider - localsecrets",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "localsecrets",
				KMSKeyURI:                 "base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=",
			},
			wantErr: false,
		},
		{
			name: "valid KMS provider - gcpkms",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "gcpkms",
				KMSKeyURI:                 "gcpkms://projects/my-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
			},
			wantErr: false,
		},
		{
			name: "valid KMS provider - awskms",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "awskms",
				KMSKeyURI:                 "awskms://arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012?region=us-east-1",
			},
			wantErr: false,
		},
		{
			name: "valid KMS provider - azurekeyvault",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "azurekeyvault",
				KMSKeyURI:                 "azurekeyvault://myvault.vault.azure.net/keys/mykey",
			},
			wantErr: false,
		},
		{
			name: "valid KMS provider - hashivault",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
				KMSProvider:               "hashivault",
				KMSKeyURI:                 "hashivault://mykey",
			},
			wantErr: false,
		},
		{
			name: "valid server timeouts - default values",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
			},
			wantErr: false,
		},
		{
			name: "valid server timeouts - minimum values",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         1 * time.Second,
				ServerWriteTimeout:        1 * time.Second,
				ServerIdleTimeout:         1 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
			},
			wantErr: false,
		},
		{
			name: "valid server timeouts - maximum values",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         300 * time.Second,
				ServerWriteTimeout:        300 * time.Second,
				ServerIdleTimeout:         300 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
			},
			wantErr: false,
		},
		{
			name: "invalid server read timeout - below minimum",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         0 * time.Second,
				ServerWriteTimeout:        15 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
			},
			wantErr: true,
		},
		{
			name: "invalid server write timeout - above maximum",
			cfg: &Config{
				DBDriver:                  "postgres",
				DBConnectionString:        "postgres://localhost",
				ServerPort:                8080,
				MetricsPort:               8081,
				LogLevel:                  "info",
				ServerReadTimeout:         15 * time.Second,
				ServerWriteTimeout:        301 * time.Second,
				ServerIdleTimeout:         60 * time.Second,
				MaxRequestBodySize:        1048576,
				SecretValueSizeLimitBytes: 524288,
			},
			wantErr: true,
		},
		{
			name: "invalid server idle timeout - negative value",
			cfg: &Config{
				DBDriver:           "postgres",
				DBConnectionString: "postgres://localhost",
				ServerPort:         8080,
				MetricsPort:        8081,
				LogLevel:           "info",
				ServerReadTimeout:  15 * time.Second,
				ServerWriteTimeout: 15 * time.Second,
				ServerIdleTimeout:  -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid max request body size - zero",
			cfg: &Config{
				DBDriver:           "postgres",
				DBConnectionString: "postgres://localhost",
				ServerPort:         8080,
				MetricsPort:        8081,
				LogLevel:           "info",
				ServerReadTimeout:  15 * time.Second,
				ServerWriteTimeout: 15 * time.Second,
				ServerIdleTimeout:  60 * time.Second,
				MaxRequestBodySize: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		validate    func(t *testing.T, cfg *Config)
		expectError bool
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
				assert.Equal(t, 15*time.Second, cfg.ServerReadTimeout)
				assert.Equal(t, 15*time.Second, cfg.ServerWriteTimeout)
				assert.Equal(t, 60*time.Second, cfg.ServerIdleTimeout)
				assert.Equal(t, int64(1048576), cfg.MaxRequestBodySize)
				assert.Equal(t, 524288, cfg.SecretValueSizeLimitBytes)
			},
		},
		{
			name: "load custom secret value size limit",
			envVars: map[string]string{
				"SECRET_VALUE_SIZE_LIMIT_BYTES": "1048576",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1048576, cfg.SecretValueSizeLimitBytes)
			},
		},
		{
			name: "load custom body size limit",
			envVars: map[string]string{
				"MAX_REQUEST_BODY_SIZE": "2097152",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, int64(2097152), cfg.MaxRequestBodySize)
			},
		},
		{
			name: "load custom server configuration",
			envVars: map[string]string{
				"SERVER_HOST":                     "localhost",
				"SERVER_PORT":                     "9090",
				"SERVER_SHUTDOWN_TIMEOUT_SECONDS": "20",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "localhost", cfg.ServerHost)
				assert.Equal(t, 9090, cfg.ServerPort)
				assert.Equal(t, 20*time.Second, cfg.ServerShutdownTimeout)
			},
		},
		{
			name: "load custom server timeout configuration",
			envVars: map[string]string{
				"SERVER_READ_TIMEOUT_SECONDS":  "30",
				"SERVER_WRITE_TIMEOUT_SECONDS": "45",
				"SERVER_IDLE_TIMEOUT_SECONDS":  "120",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 30*time.Second, cfg.ServerReadTimeout)
				assert.Equal(t, 45*time.Second, cfg.ServerWriteTimeout)
				assert.Equal(t, 120*time.Second, cfg.ServerIdleTimeout)
			},
		},
		{
			name: "load custom database configuration",
			envVars: map[string]string{
				"DB_DRIVER":                    "mysql",
				"DB_CONNECTION_STRING":         "user:password@tcp(localhost:3306)/testdb",
				"DB_MAX_OPEN_CONNECTIONS":      "50",
				"DB_MAX_IDLE_CONNECTIONS":      "10",
				"DB_CONN_MAX_LIFETIME_MINUTES": "10",
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
				"KMS_PROVIDER": "gcpkms",
				"KMS_KEY_URI":  "gcpkms://projects/my-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "gcpkms", cfg.KMSProvider)
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
			expectError: false,
		},
		{
			name: "invalid db driver fails to load",
			envVars: map[string]string{
				"DB_DRIVER":            "invalid_driver",
				"DB_CONNECTION_STRING": "postgres://localhost",
				"SERVER_PORT":          "8080",
				"METRICS_PORT":         "8081",
				"LOG_LEVEL":            "info",
			},
			expectError: true,
		},
		{
			name: "invalid server port fails to load",
			envVars: map[string]string{
				"DB_DRIVER":            "postgres",
				"DB_CONNECTION_STRING": "postgres://localhost",
				"SERVER_PORT":          "99999",
				"METRICS_PORT":         "8081",
				"LOG_LEVEL":            "info",
			},
			expectError: true,
		},
		{
			name: "conflicting server and metrics ports fails to load",
			envVars: map[string]string{
				"DB_DRIVER":            "postgres",
				"DB_CONNECTION_STRING": "postgres://localhost",
				"SERVER_PORT":          "8080",
				"METRICS_PORT":         "8080",
				"LOG_LEVEL":            "info",
			},
			expectError: true,
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
			cfg, err := Load()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				assert.Contains(t, err.Error(), "configuration validation failed")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					// Validate
					tt.validate(t, cfg)
				}
			}
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
	err = os.WriteFile(
		filepath.Join(tmpDir, ".env"),
		[]byte(
			"TEST_ENV_VAR=found\nDB_DRIVER=postgres\nDB_CONNECTION_STRING=postgres://localhost\nSERVER_PORT=8080\nMETRICS_PORT=8081\nLOG_LEVEL=info",
		),
		0600,
	)
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
