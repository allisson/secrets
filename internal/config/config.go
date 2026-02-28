// Package config provides application configuration through environment variables.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/allisson/go-env"
	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	// ServerHost is the host address the server will bind to.
	ServerHost string
	// ServerPort is the port number the server will listen on.
	ServerPort int

	// DBDriver is the database driver to use (e.g., "postgres", "mysql").
	DBDriver string
	// DBConnectionString is the connection string for the database.
	DBConnectionString string
	// DBMaxOpenConnections is the maximum number of open connections to the database.
	DBMaxOpenConnections int
	// DBMaxIdleConnections is the maximum number of idle connections in the database pool.
	DBMaxIdleConnections int
	// DBConnMaxLifetime is the maximum amount of time a connection may be reused.
	DBConnMaxLifetime time.Duration

	// LogLevel is the logging level (e.g., "debug", "info", "warn", "error").
	LogLevel string

	// AuthTokenExpiration is the duration after which an authentication token expires.
	AuthTokenExpiration time.Duration

	// RateLimitEnabled indicates whether rate limiting for authenticated endpoints is enabled.
	RateLimitEnabled bool
	// RateLimitRequestsPerSec is the number of requests allowed per second for authenticated endpoints.
	RateLimitRequestsPerSec float64
	// RateLimitBurst is the burst size for authenticated endpoints rate limiting.
	RateLimitBurst int

	// RateLimitTokenEnabled indicates whether rate limiting for the token endpoint is enabled.
	RateLimitTokenEnabled bool
	// RateLimitTokenRequestsPerSec is the number of requests allowed per second for the token endpoint.
	RateLimitTokenRequestsPerSec float64
	// RateLimitTokenBurst is the burst size for the token endpoint rate limiting.
	RateLimitTokenBurst int

	// CORSEnabled indicates whether CORS is enabled.
	CORSEnabled bool
	// CORSAllowOrigins is a comma-separated list of allowed origins for CORS.
	CORSAllowOrigins string

	// MetricsEnabled indicates whether metrics collection is enabled.
	MetricsEnabled bool
	// MetricsNamespace is the namespace for the application metrics.
	MetricsNamespace string
	// MetricsPort is the port number for the metrics server.
	MetricsPort int

	// KMSProvider is the KMS provider to use (e.g., "google", "aws", "azure").
	KMSProvider string
	// KMSKeyURI is the URI for the master key in the KMS.
	KMSKeyURI string

	// LockoutMaxAttempts is the maximum number of failed login attempts before a lockout.
	LockoutMaxAttempts int
	// LockoutDuration is the duration for which an account is locked out after maximum attempts.
	LockoutDuration time.Duration
}

// Load loads configuration from environment variables and .env file.
func Load() *Config {
	// Try to load .env file recursively
	loadDotEnv()

	return &Config{
		// Server configuration
		ServerHost: env.GetString("SERVER_HOST", "0.0.0.0"),
		ServerPort: env.GetInt("SERVER_PORT", 8080),

		// Database configuration
		DBDriver: env.GetString("DB_DRIVER", "postgres"),
		DBConnectionString: env.GetString(
			"DB_CONNECTION_STRING",
			"postgres://user:password@localhost:5432/mydb?sslmode=disable",
		),
		DBMaxOpenConnections: env.GetInt("DB_MAX_OPEN_CONNECTIONS", 25),
		DBMaxIdleConnections: env.GetInt("DB_MAX_IDLE_CONNECTIONS", 5),
		DBConnMaxLifetime:    env.GetDuration("DB_CONN_MAX_LIFETIME", 5, time.Minute),

		// Logging
		LogLevel: env.GetString("LOG_LEVEL", "info"),

		// Auth
		AuthTokenExpiration: env.GetDuration("AUTH_TOKEN_EXPIRATION_SECONDS", 14400, time.Second),

		// Rate Limiting (authenticated endpoints)
		RateLimitEnabled:        env.GetBool("RATE_LIMIT_ENABLED", true),
		RateLimitRequestsPerSec: env.GetFloat64("RATE_LIMIT_REQUESTS_PER_SEC", 10.0),
		RateLimitBurst:          env.GetInt("RATE_LIMIT_BURST", 20),

		// Rate Limiting for Token Endpoint (IP-based, unauthenticated)
		RateLimitTokenEnabled:        env.GetBool("RATE_LIMIT_TOKEN_ENABLED", true),
		RateLimitTokenRequestsPerSec: env.GetFloat64("RATE_LIMIT_TOKEN_REQUESTS_PER_SEC", 5.0),
		RateLimitTokenBurst:          env.GetInt("RATE_LIMIT_TOKEN_BURST", 10),

		// CORS
		CORSEnabled:      env.GetBool("CORS_ENABLED", false),
		CORSAllowOrigins: env.GetString("CORS_ALLOW_ORIGINS", ""),

		// Metrics
		MetricsEnabled:   env.GetBool("METRICS_ENABLED", true),
		MetricsNamespace: env.GetString("METRICS_NAMESPACE", "secrets"),
		MetricsPort:      env.GetInt("METRICS_PORT", 8081),

		// KMS configuration
		KMSProvider: env.GetString("KMS_PROVIDER", ""),
		KMSKeyURI:   env.GetString("KMS_KEY_URI", ""),

		// Account Lockout
		LockoutMaxAttempts: env.GetInt("LOCKOUT_MAX_ATTEMPTS", 10),
		LockoutDuration:    env.GetDuration("LOCKOUT_DURATION_MINUTES", 30, time.Minute),
	}
}

// GetGinMode returns the appropriate Gin mode based on log level.
func (c *Config) GetGinMode() string {
	switch c.LogLevel {
	case "debug":
		return "debug"
	case "info", "warn", "error":
		return "release"
	default:
		return "release"
	}
}

// loadDotEnv searches for a .env file recursively from the current directory
// up to the root directory and loads it if found.
func loadDotEnv() {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// Search for .env file recursively up the directory tree
	dir := cwd
	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			// .env file found, load it
			_ = godotenv.Load(envPath)
			return
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}
}
