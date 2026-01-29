package app

import (
	"context"
	"testing"
	"time"

	"github.com/allisson/go-project-template/internal/config"
)

// TestNewContainer verifies that a new container can be created with a valid configuration.
func TestNewContainer(t *testing.T) {
	cfg := &config.Config{
		LogLevel:             "info",
		DBDriver:             "postgres",
		DBConnectionString:   "postgres://test:test@localhost:5432/test?sslmode=disable",
		DBMaxOpenConnections: 10,
		DBMaxIdleConnections: 5,
		DBConnMaxLifetime:    time.Hour,
		ServerHost:           "localhost",
		ServerPort:           8080,
		WorkerInterval:       time.Second,
		WorkerBatchSize:      100,
		WorkerMaxRetries:     3,
		WorkerRetryInterval:  time.Second,
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
