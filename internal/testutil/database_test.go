package testutil

import (
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
			name: "default DSN when env var not set",
			want: defaultPostgresTestDSN,
		},
		{
			name:     "custom DSN from env var",
			envValue: "postgres://custom@localhost:5432/customdb",
			want:     "postgres://custom@localhost:5432/customdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := os.Getenv("TEST_POSTGRES_DSN")
			defer restoreEnv(t, "TEST_POSTGRES_DSN", original)

			if tt.envValue != "" {
				require.NoError(t, os.Setenv("TEST_POSTGRES_DSN", tt.envValue))
			} else {
				require.NoError(t, os.Unsetenv("TEST_POSTGRES_DSN"))
			}

			assert.Equal(t, tt.want, GetPostgresTestDSN())
		})
	}
}

func TestGetMigrationsPath(t *testing.T) {
	got, err := getMigrationsPath()
	require.NoError(t, err)
	assert.NotEmpty(t, got)

	_, statErr := os.Stat(got)
	assert.NoError(t, statErr, "migrations path should exist")
	assert.Equal(t, "migrations", filepath.Base(got))
}

func TestGetMigrationsPathFromDifferentWorkingDir(t *testing.T) {
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(originalWd))
	}()

	subDir := filepath.Join(originalWd, "testdata")
	require.NoError(t, os.MkdirAll(subDir, 0o750))
	defer func() {
		require.NoError(t, os.RemoveAll(subDir))
	}()

	require.NoError(t, os.Chdir(subDir))

	path, err := getMigrationsPath()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Equal(t, "migrations", filepath.Base(path))
}

func TestUuidToDriverValue(t *testing.T) {
	testID := uuid.Must(uuid.NewV7())

	value, err := uuidToDriverValue(testID, "postgres")
	require.NoError(t, err)

	gotUUID, ok := value.(uuid.UUID)
	require.True(t, ok, "value should be uuid.UUID")
	assert.Equal(t, testID, gotUUID)
}

func TestSetupPostgresDB(t *testing.T) {
	SkipIfNoPostgres(t)

	db := SetupPostgresDB(t)
	defer TeardownDB(t, db)

	require.NoError(t, db.Ping())

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count))
	assert.Equal(t, 0, count, "database should be clean after setup")
}

func restoreEnv(t *testing.T, key, value string) {
	t.Helper()

	if value != "" {
		require.NoError(t, os.Setenv(key, value))
		return
	}

	require.NoError(t, os.Unsetenv(key))
}
