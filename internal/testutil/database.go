// Package testutil provides testing utilities for database integration tests.
//
// Environment Variables:
//
// Database connection strings can be customized via environment variables:
//   - TEST_POSTGRES_DSN: PostgreSQL connection string (default: postgres://testuser:testpassword@localhost:5433/testdb?sslmode=disable)
//   - TEST_MYSQL_DSN: MySQL connection string (default: testuser:testpassword@tcp(localhost:3307)/testdb?parseTime=true&multiStatements=true)
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
//
// Migration Path:
//
// Migrations are automatically discovered by walking up from the current
// working directory until a "migrations/{dbType}" directory is found.
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
	// Default test database DSNs (can be overridden via environment variables)
	//nolint:gosec // test database credentials
	defaultPostgresTestDSN = "postgres://testuser:testpassword@localhost:5433/testdb?sslmode=disable"
	//nolint:gosec // test database credentials
	defaultMySQLTestDSN = "testuser:testpassword@tcp(localhost:3307)/testdb?parseTime=true&multiStatements=true"
)

// GetPostgresTestDSN returns the PostgreSQL test DSN, checking environment variable first.
func GetPostgresTestDSN() string {
	if dsn := os.Getenv("TEST_POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return defaultPostgresTestDSN
}

// GetMySQLTestDSN returns the MySQL test DSN, checking environment variable first.
func GetMySQLTestDSN() string {
	if dsn := os.Getenv("TEST_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return defaultMySQLTestDSN
}

// SetupPostgresDB creates a new PostgreSQL database connection and runs migrations.
func SetupPostgresDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", GetPostgresTestDSN())
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

	db, err := sql.Open("mysql", GetMySQLTestDSN())
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

	migrationsPath, err := getMigrationsPath("postgresql")
	require.NoError(t, err, "failed to find postgresql migrations path")

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	require.NoError(t, err, "failed to create migrate instance for postgres")

	// Note: We intentionally do NOT close the migrate instance here because we're using
	// WithInstance() with an existing database connection that we don't own. Closing the
	// migrate instance would close the underlying database connection, which is managed
	// by the caller. The file source driver will be garbage collected automatically.

	// Run migrations up
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, fmt.Sprintf("failed to run postgres migrations from %s", migrationsPath))
	}
}

// runMySQLMigrations applies all pending MySQL migrations for the test database.
func runMySQLMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	require.NoError(t, err, "failed to create mysql driver")

	migrationsPath, err := getMigrationsPath("mysql")
	require.NoError(t, err, "failed to find mysql migrations path")

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"mysql",
		driver,
	)
	require.NoError(t, err, "failed to create migrate instance for mysql")

	// Note: We intentionally do NOT close the migrate instance here because we're using
	// WithInstance() with an existing database connection that we don't own. Closing the
	// migrate instance would close the underlying database connection, which is managed
	// by the caller. The file source driver will be garbage collected automatically.

	// Run migrations up
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, fmt.Sprintf("failed to run mysql migrations from %s", migrationsPath))
	}
}

// getMigrationsPath resolves the absolute path to migration files for the specified database type.
// Walks up the directory tree from current working directory to find the migrations folder.
// Returns an error if the working directory cannot be determined or migrations are not found.
func getMigrationsPath(dbType string) (string, error) {
	// Get the project root by walking up from the current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up the directory tree until we find the migrations directory
	for {
		migrationsPath := filepath.Join(dir, "migrations", dbType)
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			return "", fmt.Errorf("migrations directory not found for %s (started from %s)", dbType, dir)
		}
		dir = parent
	}
}

// uuidToDriverValue converts a UUID to the appropriate value for the database driver.
// PostgreSQL uses UUID natively, MySQL requires binary encoding.
func uuidToDriverValue(id uuid.UUID, driver string) (interface{}, error) {
	if driver == "postgres" {
		return id, nil
	}
	// MySQL needs binary format
	return id.MarshalBinary()
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
		idValue, marshalErr := uuidToDriverValue(clientID, driver)
		require.NoError(t, marshalErr, "failed to convert client UUID for driver "+driver)
		_, err = db.ExecContext(ctx,
			`INSERT INTO clients (id, secret, name, is_active, policies, created_at) 
			 VALUES (?, ?, ?, ?, ?, NOW())`,
			idValue,
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
// The KEK is created with algorithm 'aes-gcm' and random encrypted key data.
func CreateTestKek(t *testing.T, db *sql.DB, driver, name string) uuid.UUID {
	t.Helper()

	kekID := uuid.Must(uuid.NewV7())
	ctx := context.Background()

	// Dummy encrypted KEK data (32 bytes for AES-256)
	encryptedKey := make([]byte, 32)
	_, err := rand.Read(encryptedKey)
	require.NoError(t, err, "failed to generate random KEK data")

	// Generate nonce (12 bytes for AES-GCM)
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	require.NoError(t, err, "failed to generate random nonce")

	masterKeyID := "test-master-key"

	var execErr error
	if driver == "postgres" {
		_, execErr = db.ExecContext(ctx,
			`INSERT INTO keks (id, master_key_id, version, algorithm, encrypted_key, nonce, created_at) 
			 VALUES ($1, $2, 1, 'aes-gcm', $3, $4, NOW())`,
			kekID,
			masterKeyID,
			encryptedKey,
			nonce,
		)
	} else { // mysql
		idValue, marshalErr := uuidToDriverValue(kekID, driver)
		require.NoError(t, marshalErr, "failed to convert KEK UUID for driver "+driver)
		_, execErr = db.ExecContext(ctx,
			`INSERT INTO keks (id, master_key_id, version, algorithm, encrypted_key, nonce, created_at) 
			 VALUES (?, ?, 1, 'aes-gcm', ?, ?, NOW())`,
			idValue,
			masterKeyID,
			encryptedKey,
			nonce,
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

// SkipIfNoPostgres skips the test if PostgreSQL test database is not available.
// Useful for running tests in environments without database access.
func SkipIfNoPostgres(t *testing.T) {
	t.Helper()
	db, err := sql.Open("postgres", GetPostgresTestDSN())
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer func() {
		_ = db.Close() // Ignore close error in skip helper
	}()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
}

// SkipIfNoMySQL skips the test if MySQL test database is not available.
// Useful for running tests in environments without database access.
func SkipIfNoMySQL(t *testing.T) {
	t.Helper()
	db, err := sql.Open("mysql", GetMySQLTestDSN())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer func() {
		_ = db.Close() // Ignore close error in skip helper
	}()

	if err := db.Ping(); err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
}

// CreateTestDek creates a minimal test DEK (Data Encryption Key) for repository tests.
// Returns the DEK ID. The DEK is associated with the provided KEK ID.
func CreateTestDek(t *testing.T, db *sql.DB, driver, name string, kekID uuid.UUID) uuid.UUID {
	t.Helper()

	dekID := uuid.Must(uuid.NewV7())
	ctx := context.Background()

	// Dummy encrypted DEK data (32 bytes for AES-256)
	encryptedKey := make([]byte, 32)
	_, err := rand.Read(encryptedKey)
	require.NoError(t, err, "failed to generate random DEK data")

	// Generate nonce (12 bytes for AES-GCM)
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	require.NoError(t, err, "failed to generate random nonce")

	var execErr error
	if driver == "postgres" {
		_, execErr = db.ExecContext(ctx,
			`INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			 VALUES ($1, $2, 'aes-gcm', $3, $4, NOW())`,
			dekID,
			kekID,
			encryptedKey,
			nonce,
		)
	} else { // mysql
		dekIDValue, marshalErr := uuidToDriverValue(dekID, driver)
		require.NoError(t, marshalErr, "failed to convert DEK UUID for driver "+driver)

		kekIDValue, marshalErr := uuidToDriverValue(kekID, driver)
		require.NoError(t, marshalErr, "failed to convert KEK UUID for driver "+driver)

		_, execErr = db.ExecContext(ctx,
			`INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			 VALUES (?, ?, 'aes-gcm', ?, ?, NOW())`,
			dekIDValue,
			kekIDValue,
			encryptedKey,
			nonce,
		)
	}

	require.NoError(t, execErr, "failed to create test DEK: "+name)
	return dekID
}

// ValidateTestClient verifies that a test client was created with expected values.
// Returns true if the client exists and is active, false otherwise.
func ValidateTestClient(t *testing.T, db *sql.DB, driver string, clientID uuid.UUID) bool {
	t.Helper()

	ctx := context.Background()
	var isActive bool
	var err error

	if driver == "postgres" {
		err = db.QueryRowContext(ctx, `SELECT is_active FROM clients WHERE id = $1`, clientID).Scan(&isActive)
	} else { // mysql
		idValue, marshalErr := uuidToDriverValue(clientID, driver)
		require.NoError(t, marshalErr, "failed to convert client UUID for validation")
		err = db.QueryRowContext(ctx, `SELECT is_active FROM clients WHERE id = ?`, idValue).Scan(&isActive)
	}

	if err != nil {
		return false
	}

	return isActive
}

// ValidateTestKek verifies that a test KEK was created with expected values.
// Returns true if the KEK exists, false otherwise.
func ValidateTestKek(t *testing.T, db *sql.DB, driver string, kekID uuid.UUID) bool {
	t.Helper()

	ctx := context.Background()
	var version int
	var err error

	if driver == "postgres" {
		err = db.QueryRowContext(ctx, `SELECT version FROM keks WHERE id = $1`, kekID).Scan(&version)
	} else { // mysql
		idValue, marshalErr := uuidToDriverValue(kekID, driver)
		require.NoError(t, marshalErr, "failed to convert KEK UUID for validation")
		err = db.QueryRowContext(ctx, `SELECT version FROM keks WHERE id = ?`, idValue).Scan(&version)
	}

	if err != nil {
		return false
	}

	return version > 0
}
