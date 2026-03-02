// Package database provides database connection management and utilities.
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Config holds database configuration settings.
type Config struct {
	Driver             string        // Database driver name (e.g., "postgres", "mysql").
	ConnectionString   string        // Connection string for the database.
	MaxOpenConnections int           // Maximum number of open connections to the database.
	MaxIdleConnections int           // Maximum number of idle connections in the pool.
	ConnMaxLifetime    time.Duration // Maximum amount of time a connection may be reused.
	ConnMaxIdleTime    time.Duration // Maximum amount of time a connection may be idle.
}

// Connect establishes a database connection with the given configuration.
// It sets connection pool settings and verifies the connection with a ping.
// If ping fails, the database connection is closed to prevent leaks.
func Connect(cfg Config) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		_ = db.Close() // Close to avoid connection leak on failure
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
