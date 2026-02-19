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
	// Server configuration
	ServerHost string
	ServerPort int

	// Database configuration
	DBDriver             string
	DBConnectionString   string
	DBMaxOpenConnections int
	DBMaxIdleConnections int
	DBConnMaxLifetime    time.Duration

	// Logging
	LogLevel string

	// Auth
	AuthTokenExpiration time.Duration

	// Rate Limiting
	RateLimitEnabled        bool
	RateLimitRequestsPerSec float64
	RateLimitBurst          int

	// CORS
	CORSEnabled      bool
	CORSAllowOrigins string

	// Metrics
	MetricsEnabled   bool
	MetricsNamespace string
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

		// Rate Limiting
		RateLimitEnabled:        env.GetBool("RATE_LIMIT_ENABLED", true),
		RateLimitRequestsPerSec: env.GetFloat64("RATE_LIMIT_REQUESTS_PER_SEC", 10.0),
		RateLimitBurst:          env.GetInt("RATE_LIMIT_BURST", 20),

		// CORS
		CORSEnabled:      env.GetBool("CORS_ENABLED", false),
		CORSAllowOrigins: env.GetString("CORS_ALLOW_ORIGINS", ""),

		// Metrics
		MetricsEnabled:   env.GetBool("METRICS_ENABLED", true),
		MetricsNamespace: env.GetString("METRICS_NAMESPACE", "secrets"),
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
