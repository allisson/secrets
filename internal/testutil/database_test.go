package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPostgresTestDSN(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "default DSN when env var not set",
			envValue: "",
			want:     defaultPostgresTestDSN,
		},
		//nolint:gosec // test credentials are safe in tests
		{
			name:     "custom DSN from env var",
			envValue: "postgres://custom:password@localhost:5432/customdb",
			want:     "postgres://custom:password@localhost:5432/customdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env var
			original := os.Getenv("TEST_POSTGRES_DSN")
			defer func() {
				if original != "" {
					_ = os.Setenv("TEST_POSTGRES_DSN", original)
				} else {
					_ = os.Unsetenv("TEST_POSTGRES_DSN")
				}
			}()

			// Set test env var
			if tt.envValue != "" {
				_ = os.Setenv("TEST_POSTGRES_DSN", tt.envValue)
			} else {
				_ = os.Unsetenv("TEST_POSTGRES_DSN")
			}

			got := GetPostgresTestDSN()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMySQLTestDSN(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "default DSN when env var not set",
			envValue: "",
			want:     defaultMySQLTestDSN,
		},
		{
			name:     "custom DSN from env var",
			envValue: "custom:password@tcp(localhost:3306)/customdb",
			want:     "custom:password@tcp(localhost:3306)/customdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env var
			original := os.Getenv("TEST_MYSQL_DSN")
			defer func() {
				if original != "" {
					_ = os.Setenv("TEST_MYSQL_DSN", original)
				} else {
					_ = os.Unsetenv("TEST_MYSQL_DSN")
				}
			}()

			// Set test env var
			if tt.envValue != "" {
				_ = os.Setenv("TEST_MYSQL_DSN", tt.envValue)
			} else {
				_ = os.Unsetenv("TEST_MYSQL_DSN")
			}

			got := GetMySQLTestDSN()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMigrationsPath(t *testing.T) {
	tests := []struct {
		name    string
		dbType  string
		wantErr bool
	}{
		{
			name:    "find postgresql migrations",
			dbType:  "postgresql",
			wantErr: false,
		},
		{
			name:    "find mysql migrations",
			dbType:  "mysql",
			wantErr: false,
		},
		{
			name:    "non-existent database type",
			dbType:  "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMigrationsPath(tt.dbType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, got)
				// Verify the path exists
				_, statErr := os.Stat(got)
				assert.NoError(t, statErr, "migrations path should exist")
				// Verify it contains the expected database type
				assert.Contains(t, got, tt.dbType)
			}
		})
	}
}

func TestGetMigrationsPathFromDifferentWorkingDir(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd) // Restore working directory
	}()

	// Change to a subdirectory within the project
	// This simulates running tests from a deeper directory
	subDir := filepath.Join(originalWd, "testdata")
	//nolint:gosec // 0755 is appropriate for test directories
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(subDir) // Clean up test directory
	}()

	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Should still find migrations by walking up from the subdirectory
	path, err := getMigrationsPath("postgresql")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "postgresql")
}

func TestUuidToDriverValue(t *testing.T) {
	testID := uuid.Must(uuid.NewV7())

	tests := []struct {
		name       string
		id         uuid.UUID
		driver     string
		wantErr    bool
		checkValue func(t *testing.T, value interface{})
	}{
		{
			name:    "postgres returns UUID directly",
			id:      testID,
			driver:  "postgres",
			wantErr: false,
			checkValue: func(t *testing.T, value interface{}) {
				gotUUID, ok := value.(uuid.UUID)
				assert.True(t, ok, "value should be uuid.UUID")
				assert.Equal(t, testID, gotUUID)
			},
		},
		{
			name:    "mysql returns binary",
			id:      testID,
			driver:  "mysql",
			wantErr: false,
			checkValue: func(t *testing.T, value interface{}) {
				gotBytes, ok := value.([]byte)
				assert.True(t, ok, "value should be []byte")
				assert.Len(t, gotBytes, 16, "UUID binary should be 16 bytes")
			},
		},
		{
			name:    "unknown driver defaults to mysql behavior",
			id:      testID,
			driver:  "unknown",
			wantErr: false,
			checkValue: func(t *testing.T, value interface{}) {
				_, ok := value.([]byte)
				assert.True(t, ok, "value should be []byte for unknown driver")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := uuidToDriverValue(tt.id, tt.driver)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkValue != nil {
					tt.checkValue(t, value)
				}
			}
		})
	}
}

func TestSetupPostgresDB(t *testing.T) {
	// Skip if PostgreSQL is not available
	SkipIfNoPostgres(t)

	db := SetupPostgresDB(t)
	defer TeardownDB(t, db)

	// Verify database connection is working
	err := db.Ping()
	assert.NoError(t, err)

	// Verify database is clean (no clients should exist)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "database should be clean after setup")
}

func TestSetupMySQLDB(t *testing.T) {
	// Skip if MySQL is not available
	SkipIfNoMySQL(t)

	db := SetupMySQLDB(t)
	defer TeardownDB(t, db)

	// Verify database connection is working
	err := db.Ping()
	assert.NoError(t, err)

	// Verify database is clean (no clients should exist)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "database should be clean after setup")
}

func TestTeardownDB(t *testing.T) {
	SkipIfNoPostgres(t)

	db := SetupPostgresDB(t)
	require.NotNil(t, db)

	// Teardown should close the connection
	TeardownDB(t, db)

	// Attempting to ping after teardown should fail
	err := db.Ping()
	assert.Error(t, err, "database should be closed after teardown")
}

func TestTeardownDBWithNilDB(t *testing.T) {
	// Should not panic with nil database
	assert.NotPanics(t, func() {
		TeardownDB(t, nil)
	})
}

func TestCleanupPostgresDB(t *testing.T) {
	SkipIfNoPostgres(t)

	db := SetupPostgresDB(t)
	defer TeardownDB(t, db)

	// Create test data
	clientID := CreateTestClient(t, db, "postgres", "test-cleanup-client")
	require.NotEqual(t, uuid.Nil, clientID)

	// Verify data exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Cleanup should remove all data
	CleanupPostgresDB(t, db)

	// Verify data is removed
	err = db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "cleanup should remove all data")
}

func TestCleanupMySQLDB(t *testing.T) {
	SkipIfNoMySQL(t)

	db := SetupMySQLDB(t)
	defer TeardownDB(t, db)

	// Create test data
	clientID := CreateTestClient(t, db, "mysql", "test-cleanup-client")
	require.NotEqual(t, uuid.Nil, clientID)

	// Verify data exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Cleanup should remove all data
	CleanupMySQLDB(t, db)

	// Verify data is removed
	err = db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "cleanup should remove all data")
}

func TestCreateTestClient(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		setup  func(t *testing.T) *sql.DB
		skip   func(t *testing.T)
	}{
		{
			name:   "create client in postgres",
			driver: "postgres",
			setup:  SetupPostgresDB,
			skip:   SkipIfNoPostgres,
		},
		{
			name:   "create client in mysql",
			driver: "mysql",
			setup:  SetupMySQLDB,
			skip:   SkipIfNoMySQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.skip(t)

			db := tt.setup(t)
			defer TeardownDB(t, db)

			clientID := CreateTestClient(t, db, tt.driver, "test-client")
			assert.NotEqual(t, uuid.Nil, clientID)

			// Verify client was created
			valid := ValidateTestClient(t, db, tt.driver, clientID)
			assert.True(t, valid, "client should exist and be active")
		})
	}
}

func TestCreateTestKek(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		setup  func(t *testing.T) *sql.DB
		skip   func(t *testing.T)
	}{
		{
			name:   "create KEK in postgres",
			driver: "postgres",
			setup:  SetupPostgresDB,
			skip:   SkipIfNoPostgres,
		},
		{
			name:   "create KEK in mysql",
			driver: "mysql",
			setup:  SetupMySQLDB,
			skip:   SkipIfNoMySQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.skip(t)

			db := tt.setup(t)
			defer TeardownDB(t, db)

			kekID := CreateTestKek(t, db, tt.driver, "test-kek")
			assert.NotEqual(t, uuid.Nil, kekID)

			// Verify KEK was created
			valid := ValidateTestKek(t, db, tt.driver, kekID)
			assert.True(t, valid, "KEK should exist")
		})
	}
}

func TestCreateTestClientAndKek(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		setup  func(t *testing.T) *sql.DB
		skip   func(t *testing.T)
	}{
		{
			name:   "create client and KEK in postgres",
			driver: "postgres",
			setup:  SetupPostgresDB,
			skip:   SkipIfNoPostgres,
		},
		{
			name:   "create client and KEK in mysql",
			driver: "mysql",
			setup:  SetupMySQLDB,
			skip:   SkipIfNoMySQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.skip(t)

			db := tt.setup(t)
			defer TeardownDB(t, db)

			clientID, kekID := CreateTestClientAndKek(t, db, tt.driver, "test-fixtures")

			assert.NotEqual(t, uuid.Nil, clientID)
			assert.NotEqual(t, uuid.Nil, kekID)
			assert.NotEqual(t, clientID, kekID, "client ID and KEK ID should be different")

			// Verify both were created
			clientValid := ValidateTestClient(t, db, tt.driver, clientID)
			assert.True(t, clientValid, "client should exist")

			kekValid := ValidateTestKek(t, db, tt.driver, kekID)
			assert.True(t, kekValid, "KEK should exist")
		})
	}
}

func TestCreateTestDek(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		setup  func(t *testing.T) *sql.DB
		skip   func(t *testing.T)
	}{
		{
			name:   "create DEK in postgres",
			driver: "postgres",
			setup:  SetupPostgresDB,
			skip:   SkipIfNoPostgres,
		},
		{
			name:   "create DEK in mysql",
			driver: "mysql",
			setup:  SetupMySQLDB,
			skip:   SkipIfNoMySQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.skip(t)

			db := tt.setup(t)
			defer TeardownDB(t, db)

			// Create prerequisites (only need KEK, not client)
			kekID := CreateTestKek(t, db, tt.driver, "test-dek-kek")

			// Create DEK
			dekID := CreateTestDek(t, db, tt.driver, "test-dek", kekID)
			assert.NotEqual(t, uuid.Nil, dekID)

			// Verify DEK was created by checking it exists
			var algorithm string
			var err error
			if tt.driver == "postgres" {
				err = db.QueryRow("SELECT algorithm FROM deks WHERE id = $1", dekID).Scan(&algorithm)
			} else {
				idValue, marshalErr := uuidToDriverValue(dekID, tt.driver)
				require.NoError(t, marshalErr)
				err = db.QueryRow("SELECT algorithm FROM deks WHERE id = ?", idValue).Scan(&algorithm)
			}
			assert.NoError(t, err)
			assert.Equal(t, "aes-gcm", algorithm, "DEK should have aes-gcm algorithm")
		})
	}
}

func TestValidateTestClient(t *testing.T) {
	SkipIfNoPostgres(t)

	db := SetupPostgresDB(t)
	defer TeardownDB(t, db)

	// Test with valid client
	clientID := CreateTestClient(t, db, "postgres", "valid-client")
	valid := ValidateTestClient(t, db, "postgres", clientID)
	assert.True(t, valid, "should validate existing client")

	// Test with non-existent client
	nonExistentID := uuid.Must(uuid.NewV7())
	valid = ValidateTestClient(t, db, "postgres", nonExistentID)
	assert.False(t, valid, "should not validate non-existent client")
}

func TestValidateTestKek(t *testing.T) {
	SkipIfNoPostgres(t)

	db := SetupPostgresDB(t)
	defer TeardownDB(t, db)

	// Test with valid KEK
	kekID := CreateTestKek(t, db, "postgres", "valid-kek")
	valid := ValidateTestKek(t, db, "postgres", kekID)
	assert.True(t, valid, "should validate existing KEK")

	// Test with non-existent KEK
	nonExistentID := uuid.Must(uuid.NewV7())
	valid = ValidateTestKek(t, db, "postgres", nonExistentID)
	assert.False(t, valid, "should not validate non-existent KEK")
}

func TestSkipIfNoPostgres(t *testing.T) {
	// This test verifies that SkipIfNoPostgres doesn't panic
	// We can't easily test the actual skipping behavior without mocking
	t.Run("does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			SkipIfNoPostgres(t)
		})
	})
}

func TestSkipIfNoMySQL(t *testing.T) {
	// This test verifies that SkipIfNoMySQL doesn't panic
	// We can't easily test the actual skipping behavior without mocking
	t.Run("does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			SkipIfNoMySQL(t)
		})
	})
}
