package config

import (
	"os"
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
				assert.Equal(t, 86400*time.Second, cfg.AuthTokenExpiration)
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
