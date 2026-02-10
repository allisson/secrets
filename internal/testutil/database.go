package testutil

import (
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
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	PostgresTestDSN = "postgres://testuser:testpassword@localhost:5433/testdb?sslmode=disable"
	MySQLTestDSN    = "testuser:testpassword@tcp(localhost:3307)/testdb?parseTime=true&multiStatements=true"
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
		"TRUNCATE TABLE audit_logs, transit_keys, secrets, deks, keks, client_policies, policies, tokens, clients RESTART IDENTITY CASCADE",
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

	_, err = db.Exec("TRUNCATE TABLE deks")
	require.NoError(t, err, "failed to truncate deks table")

	_, err = db.Exec("TRUNCATE TABLE keks")
	require.NoError(t, err, "failed to truncate keks table")

	_, err = db.Exec("TRUNCATE TABLE client_policies")
	require.NoError(t, err, "failed to truncate client_policies table")

	_, err = db.Exec("TRUNCATE TABLE policies")
	require.NoError(t, err, "failed to truncate policies table")

	_, err = db.Exec("TRUNCATE TABLE tokens")
	require.NoError(t, err, "failed to truncate tokens table")

	_, err = db.Exec("TRUNCATE TABLE clients")
	require.NoError(t, err, "failed to truncate clients table")

	// Re-enable foreign key checks
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	require.NoError(t, err, "failed to enable foreign key checks")
}

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
