// Package testutil provides testing utilities for database integration tests.
//
// Database Setup:
//
//	db := testutil.SetupPostgresDB(t)
//	defer testutil.TeardownDB(t, db)
//	defer testutil.CleanupPostgresDB(t, db)
//
// Test Fixtures (for foreign key constraints):
//
//	clientID := testutil.CreateTestClient(t, db, "postgres", "my-test-client")
//	kekID := testutil.CreateTestKek(t, db, "postgres", "my-test-kek")
//
//	// Or both:
//	clientID, kekID := testutil.CreateTestClientAndKek(t, db, "postgres", "my-test")
package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	//nolint:gosec // test database credentials
	PostgresTestDSN = "postgres://testuser:testpassword@localhost:5433/testdb?sslmode=disable"
	//nolint:gosec // test database credentials
	MySQLTestDSN = "testuser:testpassword@tcp(localhost:3307)/testdb?parseTime=true&multiStatements=true"
)

// SetupPostgresDB creates a new PostgreSQL database connection and runs migrations.
func SetupPostgresDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", PostgresTestDSN)
	require.NoError(t, err, "failed to connect to postgres")

	err = db.Ping()
	require.NoError(t, err, "failed to ping postgres database")

	// Run migrations
	runPostgresMigrations(t, db)

	// Clean up any existing data before the test runs
	CleanupPostgresDB(t, db)

	return db
}

// SetupMySQLDB creates a new MySQL database connection and runs migrations.
func SetupMySQLDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("mysql", MySQLTestDSN)
	require.NoError(t, err, "failed to connect to mysql")

	err = db.Ping()
	require.NoError(t, err, "failed to ping mysql database")

	// Run migrations
	runMySQLMigrations(t, db)

	// Clean up any existing data before the test runs
	CleanupMySQLDB(t, db)

	return db
}

// TeardownDB closes the database connection and cleans up.
func TeardownDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if db != nil {
		err := db.Close()
		require.NoError(t, err, "failed to close database connection")
	}
}

// CleanupPostgresDB truncates all tables in the PostgreSQL database.
func CleanupPostgresDB(t *testing.T, db *sql.DB) {
	t.Helper()

	// Truncate tables in reverse order to respect foreign key constraints
	_, err := db.Exec(
		"TRUNCATE TABLE audit_logs, transit_keys, secrets, tokenization_tokens, tokenization_keys, deks, keks, tokens, clients RESTART IDENTITY CASCADE",
	)
	require.NoError(t, err, "failed to truncate postgres tables")
}

// CleanupMySQLDB truncates all tables in the MySQL database.
func CleanupMySQLDB(t *testing.T, db *sql.DB) {
	t.Helper()

	// Disable foreign key checks temporarily
	_, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	require.NoError(t, err, "failed to disable foreign key checks")

	// Truncate tables
	_, err = db.Exec("TRUNCATE TABLE audit_logs")
	require.NoError(t, err, "failed to truncate audit_logs table")

	_, err = db.Exec("TRUNCATE TABLE transit_keys")
	require.NoError(t, err, "failed to truncate transit_keys table")

	_, err = db.Exec("TRUNCATE TABLE secrets")
	require.NoError(t, err, "failed to truncate secrets table")

	_, err = db.Exec("TRUNCATE TABLE tokenization_tokens")
	require.NoError(t, err, "failed to truncate tokenization_tokens table")

	_, err = db.Exec("TRUNCATE TABLE tokenization_keys")
	require.NoError(t, err, "failed to truncate tokenization_keys table")

	_, err = db.Exec("TRUNCATE TABLE deks")
	require.NoError(t, err, "failed to truncate deks table")

	_, err = db.Exec("TRUNCATE TABLE keks")
	require.NoError(t, err, "failed to truncate keks table")

	_, err = db.Exec("TRUNCATE TABLE tokens")
	require.NoError(t, err, "failed to truncate tokens table")

	_, err = db.Exec("TRUNCATE TABLE clients")
	require.NoError(t, err, "failed to truncate clients table")

	// Re-enable foreign key checks
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	require.NoError(t, err, "failed to enable foreign key checks")
}

// runPostgresMigrations applies all pending PostgreSQL migrations for the test database.
func runPostgresMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	require.NoError(t, err, "failed to create postgres driver")

	migrationsPath := getMigrationsPath("postgresql")
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	require.NoError(t, err, "failed to create migrate instance")

	// Run migrations up
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "failed to run postgres migrations")
	}
}

// runMySQLMigrations applies all pending MySQL migrations for the test database.
func runMySQLMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	require.NoError(t, err, "failed to create mysql driver")

	migrationsPath := getMigrationsPath("mysql")
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"mysql",
		driver,
	)
	require.NoError(t, err, "failed to create migrate instance")

	// Run migrations up
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "failed to run mysql migrations")
	}
}

// getMigrationsPath resolves the absolute path to migration files for the specified database type.
// Walks up the directory tree from current working directory to find the migrations folder.
func getMigrationsPath(dbType string) string {
	// Get the project root by walking up from the current directory
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get working directory: %v", err))
	}

	// Walk up the directory tree until we find the migrations directory
	for {
		migrationsPath := filepath.Join(dir, "migrations", dbType)
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			panic("migrations directory not found")
		}
		dir = parent
	}
}

// CreateTestClient creates a minimal active test client for repository tests.
// Returns the client ID for use in foreign key relationships. The client is
// created with a wildcard policy allowing all capabilities on all paths.
func CreateTestClient(t *testing.T, db *sql.DB, driver, name string) uuid.UUID {
	t.Helper()

	clientID := uuid.Must(uuid.NewV7())
	ctx := context.Background()

	// Minimal wildcard policy for test clients
	policiesJSON := `[{"path":"*","capabilities":["read","write","delete","encrypt","decrypt","rotate"]}]`

	var err error
	if driver == "postgres" {
		_, err = db.ExecContext(ctx,
			`INSERT INTO clients (id, secret, name, is_active, policies, created_at) 
			 VALUES ($1, $2, $3, $4, $5, NOW())`,
			clientID,
			"test-secret-hash",
			name,
			true,
			policiesJSON,
		)
	} else { // mysql
		idBinary, marshalErr := clientID.MarshalBinary()
		require.NoError(t, marshalErr, "failed to marshal client UUID")
		_, err = db.ExecContext(ctx,
			`INSERT INTO clients (id, secret, name, is_active, policies, created_at) 
			 VALUES (?, ?, ?, ?, ?, NOW())`,
			idBinary,
			"test-secret-hash",
			name,
			true,
			policiesJSON,
		)
	}

	require.NoError(t, err, "failed to create test client: "+name)
	return clientID
}

// CreateTestKek creates a minimal test KEK for repository tests that need
// to reference a KEK (e.g., signed audit logs). Returns the KEK ID.
func CreateTestKek(t *testing.T, db *sql.DB, driver, name string) uuid.UUID {
	t.Helper()

	kekID := uuid.Must(uuid.NewV7())
	ctx := context.Background()

	// Dummy encrypted KEK data (32 bytes for AES-256)
	encryptedKey := make([]byte, 32)
	_, err := rand.Read(encryptedKey)
	require.NoError(t, err, "failed to generate random KEK data")

	var execErr error
	if driver == "postgres" {
		_, execErr = db.ExecContext(ctx,
			`INSERT INTO keks (id, version, algorithm, encrypted_key, created_at) 
			 VALUES ($1, 1, 'aes-gcm', $2, NOW())`,
			kekID,
			encryptedKey,
		)
	} else { // mysql
		idBinary, marshalErr := kekID.MarshalBinary()
		require.NoError(t, marshalErr, "failed to marshal KEK UUID")
		_, execErr = db.ExecContext(ctx,
			`INSERT INTO keks (id, version, algorithm, encrypted_key, created_at) 
			 VALUES (?, 1, 'aes-gcm', ?, NOW())`,
			idBinary,
			encryptedKey,
		)
	}

	require.NoError(t, execErr, "failed to create test KEK: "+name)
	return kekID
}

// CreateTestClientAndKek creates both a test client and KEK, returning both IDs.
// Convenience wrapper for tests that need both fixtures.
func CreateTestClientAndKek(t *testing.T, db *sql.DB, driver, baseName string) (clientID, kekID uuid.UUID) {
	t.Helper()
	clientID = CreateTestClient(t, db, driver, baseName+"-client")
	kekID = CreateTestKek(t, db, driver, baseName+"-kek")
	return clientID, kekID
}
