// Package config provides application configuration through environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/allisson/go-env"
	validation "github.com/jellydator/validation"
	"github.com/joho/godotenv"
)

// Default configuration values.
const (
	DefaultServerHost            = "0.0.0.0"
	DefaultServerPort            = 8080
	DefaultServerShutdownTimeout = 10 // seconds
	DefaultServerReadTimeout     = 15 // seconds
	DefaultServerWriteTimeout    = 15 // seconds
	DefaultServerIdleTimeout     = 60 // seconds
	DefaultDBDriver              = "postgres"
	DefaultDBConnectionString    = "postgres://user:password@localhost:5432/mydb?sslmode=disable" //nolint:gosec
	DefaultDBMaxOpenConnections  = 25

	DefaultDBMaxIdleConnections   = 5
	DefaultDBConnMaxLifetime      = 5 // minutes
	DefaultLogLevel               = "info"
	DefaultAuthTokenExpiration    = 14400 // seconds
	DefaultRateLimitEnabled       = true
	DefaultRateLimitRequests      = 10.0
	DefaultRateLimitBurst         = 20
	DefaultRateLimitTokenEnabled  = true
	DefaultRateLimitTokenRequests = 5.0
	DefaultRateLimitTokenBurst    = 10
	DefaultCORSEnabled            = false
	DefaultCORSAllowOrigins       = ""
	DefaultMetricsEnabled         = true
	DefaultMetricsNamespace       = "secrets"
	DefaultMetricsPort            = 8081
	DefaultLockoutMaxAttempts     = 10
	DefaultLockoutDuration        = 30 // minutes
	DefaultMaxRequestBodySize     = 1048576
	DefaultSecretValueSizeLimit   = 524288
)

// Config holds all application configuration.
type Config struct {
	// ServerHost is the host address the server will bind to.
	ServerHost string
	// ServerPort is the port number the server will listen on.
	ServerPort int
	// ServerShutdownTimeout is the maximum time to wait for the server to gracefully shutdown.
	ServerShutdownTimeout time.Duration
	// ServerReadTimeout is the maximum duration for reading the entire request, including the body.
	ServerReadTimeout time.Duration
	// ServerWriteTimeout is the maximum duration before timing out writes of the response.
	ServerWriteTimeout time.Duration
	// ServerIdleTimeout is the maximum time to wait for the next request when keep-alives are enabled.
	ServerIdleTimeout time.Duration

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

	// LogLevel is the logging level (e.g., "debug", "info", "warn", "error", "fatal", "panic").
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

	// KMSProvider is the KMS provider to use (e.g., "google", "aws", "azure", "hashivault", "localsecrets").
	KMSProvider string
	// KMSKeyURI is the URI for the master key in the KMS.
	KMSKeyURI string

	// LockoutMaxAttempts is the maximum number of failed login attempts before a lockout.
	LockoutMaxAttempts int
	// LockoutDuration is the duration for which an account is locked out after maximum attempts.
	LockoutDuration time.Duration
	// MaxRequestBodySize is the maximum size of the request body in bytes.
	MaxRequestBodySize int64
	// SecretValueSizeLimitBytes is the maximum size of a secret value in bytes.
	SecretValueSizeLimitBytes int
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.DBDriver, validation.Required, validation.In("postgres", "mysql")),
		validation.Field(&c.DBConnectionString, validation.Required),
		validation.Field(&c.ServerPort, validation.Required, validation.Min(1), validation.Max(65535)),
		validation.Field(
			&c.ServerReadTimeout,
			validation.Required,
			validation.Min(1*time.Second),
			validation.Max(300*time.Second),
		),
		validation.Field(
			&c.ServerWriteTimeout,
			validation.Required,
			validation.Min(1*time.Second),
			validation.Max(300*time.Second),
		),
		validation.Field(
			&c.ServerIdleTimeout,
			validation.Required,
			validation.Min(1*time.Second),
			validation.Max(300*time.Second),
		),
		validation.Field(
			&c.MetricsPort,
			validation.Required,
			validation.Min(1),
			validation.Max(65535),
			validation.NotIn(c.ServerPort),
		),
		validation.Field(
			&c.LogLevel,
			validation.Required,
			validation.In("debug", "info", "warn", "error", "fatal", "panic"),
		),
		validation.Field(
			&c.KMSProvider,
			validation.When(c.KMSKeyURI != "", validation.Required),
			validation.When(
				c.KMSProvider != "",
				validation.In("localsecrets", "gcpkms", "awskms", "azurekeyvault", "hashivault"),
			),
		),
		validation.Field(&c.KMSKeyURI, validation.When(c.KMSProvider != "", validation.Required)),
		validation.Field(
			&c.RateLimitRequestsPerSec,
			validation.When(c.RateLimitEnabled, validation.Required, validation.Min(0.1)),
		),
		validation.Field(
			&c.RateLimitTokenRequestsPerSec,
			validation.When(c.RateLimitTokenEnabled, validation.Required, validation.Min(0.1)),
		),
		validation.Field(&c.MaxRequestBodySize, validation.Required, validation.Min(int64(1))),
		validation.Field(&c.SecretValueSizeLimitBytes, validation.Required, validation.Min(1)),
	)
}

// Load loads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Try to load .env file recursively
	loadDotEnv()

	cfg := &Config{
		// Server configuration
		ServerHost: env.GetString("SERVER_HOST", DefaultServerHost),
		ServerPort: env.GetInt("SERVER_PORT", DefaultServerPort),
		ServerShutdownTimeout: env.GetDuration(
			"SERVER_SHUTDOWN_TIMEOUT_SECONDS",
			DefaultServerShutdownTimeout,
			time.Second,
		),
		ServerReadTimeout: env.GetDuration(
			"SERVER_READ_TIMEOUT_SECONDS",
			DefaultServerReadTimeout,
			time.Second,
		),
		ServerWriteTimeout: env.GetDuration(
			"SERVER_WRITE_TIMEOUT_SECONDS",
			DefaultServerWriteTimeout,
			time.Second,
		),
		ServerIdleTimeout: env.GetDuration(
			"SERVER_IDLE_TIMEOUT_SECONDS",
			DefaultServerIdleTimeout,
			time.Second,
		),

		// Database configuration
		DBDriver: env.GetString("DB_DRIVER", DefaultDBDriver),
		DBConnectionString: env.GetString(
			"DB_CONNECTION_STRING",
			DefaultDBConnectionString,
		),
		DBMaxOpenConnections: env.GetInt("DB_MAX_OPEN_CONNECTIONS", DefaultDBMaxOpenConnections),
		DBMaxIdleConnections: env.GetInt("DB_MAX_IDLE_CONNECTIONS", DefaultDBMaxIdleConnections),
		DBConnMaxLifetime: env.GetDuration(
			"DB_CONN_MAX_LIFETIME_MINUTES",
			DefaultDBConnMaxLifetime,
			time.Minute,
		),

		// Logging
		LogLevel: env.GetString("LOG_LEVEL", DefaultLogLevel),

		// Auth
		AuthTokenExpiration: env.GetDuration(
			"AUTH_TOKEN_EXPIRATION_SECONDS",
			DefaultAuthTokenExpiration,
			time.Second,
		),

		// Rate Limiting (authenticated endpoints)
		RateLimitEnabled:        env.GetBool("RATE_LIMIT_ENABLED", DefaultRateLimitEnabled),
		RateLimitRequestsPerSec: env.GetFloat64("RATE_LIMIT_REQUESTS_PER_SEC", DefaultRateLimitRequests),
		RateLimitBurst:          env.GetInt("RATE_LIMIT_BURST", DefaultRateLimitBurst),

		// Rate Limiting for Token Endpoint (IP-based, unauthenticated)
		RateLimitTokenEnabled: env.GetBool("RATE_LIMIT_TOKEN_ENABLED", DefaultRateLimitTokenEnabled),
		RateLimitTokenRequestsPerSec: env.GetFloat64(
			"RATE_LIMIT_TOKEN_REQUESTS_PER_SEC",
			DefaultRateLimitTokenRequests,
		),
		RateLimitTokenBurst: env.GetInt("RATE_LIMIT_TOKEN_BURST", DefaultRateLimitTokenBurst),

		// CORS
		CORSEnabled:      env.GetBool("CORS_ENABLED", DefaultCORSEnabled),
		CORSAllowOrigins: env.GetString("CORS_ALLOW_ORIGINS", DefaultCORSAllowOrigins),

		// Metrics
		MetricsEnabled:   env.GetBool("METRICS_ENABLED", DefaultMetricsEnabled),
		MetricsNamespace: env.GetString("METRICS_NAMESPACE", DefaultMetricsNamespace),
		MetricsPort:      env.GetInt("METRICS_PORT", DefaultMetricsPort),

		// KMS configuration
		KMSProvider: env.GetString("KMS_PROVIDER", ""),
		KMSKeyURI:   env.GetString("KMS_KEY_URI", ""),

		// Account Lockout
		LockoutMaxAttempts: env.GetInt("LOCKOUT_MAX_ATTEMPTS", DefaultLockoutMaxAttempts),
		LockoutDuration:    env.GetDuration("LOCKOUT_DURATION_MINUTES", DefaultLockoutDuration, time.Minute),

		// Request Body Size
		MaxRequestBodySize: env.GetInt64("MAX_REQUEST_BODY_SIZE", DefaultMaxRequestBodySize),

		// Secret Value Size Limit
		SecretValueSizeLimitBytes: env.GetInt(
			"SECRET_VALUE_SIZE_LIMIT_BYTES",
			DefaultSecretValueSizeLimit,
		),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
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
