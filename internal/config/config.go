// Package config provides application configuration management through environment variables.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/allisson/go-env"
	"github.com/joho/godotenv"
)

// Config holds all application configuration
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

	// Master key
	MasterKey []byte

	// Worker configuration
	WorkerInterval      time.Duration
	WorkerBatchSize     int
	WorkerMaxRetries    int
	WorkerRetryInterval time.Duration
}

// Load loads configuration from environment variables.
// It first attempts to load a .env file by searching recursively from the current directory
// up to the root directory. If no .env file is found, it continues with existing environment variables.
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

		// Master key
		MasterKey: env.GetBase64ToBytes("MASTER_KEY", []byte("")),

		// Worker configuration
		WorkerInterval:      env.GetDuration("WORKER_INTERVAL", 5, time.Second),
		WorkerBatchSize:     env.GetInt("WORKER_BATCH_SIZE", 10),
		WorkerMaxRetries:    env.GetInt("WORKER_MAX_RETRIES", 3),
		WorkerRetryInterval: env.GetDuration("WORKER_RETRY_INTERVAL", 1, time.Minute),
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
