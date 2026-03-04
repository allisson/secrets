//go:build integration

package postgresql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoRepository "github.com/allisson/secrets/internal/crypto/repository/postgresql"
	"github.com/allisson/secrets/internal/testutil"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

func TestNewPostgreSQLTransitKeyRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLTransitKeyRepository{}, repo)
}

func TestPostgreSQLTransitKeyRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create prerequisite KEK and DEK
	dekID := createTestDek(t, db)

	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "payment-encryption",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Verify the transit key was created by reading it back
	var readKey transitDomain.TransitKey
	query := `SELECT id, name, version, dek_id, created_at, deleted_at FROM transit_keys WHERE id = $1`
	err = db.QueryRowContext(ctx, query, transitKey.ID).Scan(
		&readKey.ID,
		&readKey.Name,
		&readKey.Version,
		&readKey.DekID,
		&readKey.CreatedAt,
		&readKey.DeletedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, transitKey.ID, readKey.ID)
	assert.Equal(t, transitKey.Name, readKey.Name)
	assert.Equal(t, transitKey.Version, readKey.Version)
	assert.Equal(t, transitKey.DekID, readKey.DekID)
	assert.WithinDuration(t, transitKey.CreatedAt, readKey.CreatedAt, time.Second)
	assert.Nil(t, readKey.DeletedAt)
}

func TestPostgreSQLTransitKeyRepository_Create_MultipleVersions(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "api-encryption",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "api-encryption",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	// Create version 3
	time.Sleep(time.Millisecond)
	key3 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "api-encryption",
		Version:   3,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key3)
	require.NoError(t, err)

	// Verify all three versions exist
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transit_keys WHERE name = $1`, "api-encryption").
		Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestPostgreSQLTransitKeyRepository_Delete(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create a transit key
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "test-key",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Delete the transit key (soft delete)
	err = repo.Delete(ctx, transitKey.ID)
	require.NoError(t, err)

	// Verify the key still exists but has deleted_at set
	var deletedAt *time.Time
	query := `SELECT deleted_at FROM transit_keys WHERE id = $1`
	err = db.QueryRowContext(ctx, query, transitKey.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt, "deleted_at should be set after soft delete")
}

func TestPostgreSQLTransitKeyRepository_GetByName_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create a transit key
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "user-data-encryption",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Retrieve the key by name
	retrievedKey, err := repo.GetByName(ctx, "user-data-encryption")
	require.NoError(t, err)
	require.NotNil(t, retrievedKey)

	assert.Equal(t, transitKey.ID, retrievedKey.ID)
	assert.Equal(t, transitKey.Name, retrievedKey.Name)
	assert.Equal(t, transitKey.Version, retrievedKey.Version)
	assert.Equal(t, transitKey.DekID, retrievedKey.DekID)
	assert.WithinDuration(t, transitKey.CreatedAt, retrievedKey.CreatedAt, time.Second)
}

func TestPostgreSQLTransitKeyRepository_GetByName_LatestVersion(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "multi-version-key",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create version 3 (out of order)
	time.Sleep(time.Millisecond)
	key3 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "multi-version-key",
		Version:   3,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key3)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "multi-version-key",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	// GetByName should return version 3 (highest version)
	retrievedKey, err := repo.GetByName(ctx, "multi-version-key")
	require.NoError(t, err)
	require.NotNil(t, retrievedKey)

	assert.Equal(t, uint(3), retrievedKey.Version)
	assert.Equal(t, key3.ID, retrievedKey.ID)
}

func TestPostgreSQLTransitKeyRepository_GetByName_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	// Try to get a non-existent key
	retrievedKey, err := repo.GetByName(ctx, "non-existent-key")
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
}

func TestPostgreSQLTransitKeyRepository_GetByName_IgnoresDeletedKeys(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-key-test",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-key-test",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	// Delete version 2 (the latest)
	err = repo.Delete(ctx, key2.ID)
	require.NoError(t, err)

	// GetByName should return version 1 (since version 2 is deleted)
	retrievedKey, err := repo.GetByName(ctx, "deleted-key-test")
	require.NoError(t, err)
	require.NotNil(t, retrievedKey)

	assert.Equal(t, uint(1), retrievedKey.Version)
	assert.Equal(t, key1.ID, retrievedKey.ID)
}

func TestPostgreSQLTransitKeyRepository_GetByName_AllVersionsDeleted(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create a transit key
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "all-deleted-test",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Delete the key
	err = repo.Delete(ctx, transitKey.ID)
	require.NoError(t, err)

	// GetByName should return not found error
	retrievedKey, err := repo.GetByName(ctx, "all-deleted-test")
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
}

func TestPostgreSQLTransitKeyRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "tx-test-key",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create transit key within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		transitKey.ID,
		transitKey.Name,
		transitKey.Version,
		transitKey.DekID,
		transitKey.CreatedAt,
		transitKey.DeletedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the transit key was not created (rollback worked)
	retrievedKey, err := repo.GetByName(ctx, "tx-test-key")
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
}

func TestPostgreSQLTransitKeyRepository_Delete_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create initial transit key
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "tx-delete-test",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Delete within transaction
	_, err = tx.ExecContext(ctx, `UPDATE transit_keys SET deleted_at = NOW() WHERE id = $1`, transitKey.ID)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the transit key was not deleted (rollback worked)
	retrievedKey, err := repo.GetByName(ctx, "tx-delete-test")
	require.NoError(t, err)
	assert.NotNil(t, retrievedKey)
	assert.Equal(t, transitKey.ID, retrievedKey.ID)
}

func TestPostgreSQLTransitKeyRepository_GetByName_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create a transit key outside transaction
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "tx-read-test",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create another version inside transaction
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "tx-read-test",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		key2.ID,
		key2.Name,
		key2.Version,
		key2.DekID,
		key2.CreatedAt,
		key2.DeletedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see version 2
	var retrievedKey transitDomain.TransitKey
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, name, version, dek_id, created_at, deleted_at 
		 FROM transit_keys 
		 WHERE name = $1 AND deleted_at IS NULL 
		 ORDER BY version DESC 
		 LIMIT 1`,
		"tx-read-test",
	).Scan(
		&retrievedKey.ID,
		&retrievedKey.Name,
		&retrievedKey.Version,
		&retrievedKey.DekID,
		&retrievedKey.CreatedAt,
		&retrievedKey.DeletedAt,
	)
	require.NoError(t, err)
	assert.Equal(t, uint(2), retrievedKey.Version)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Query outside transaction should also see version 2
	retrievedKey2, err := repo.GetByName(ctx, "tx-read-test")
	require.NoError(t, err)
	assert.Equal(t, uint(2), retrievedKey2.Version)
}

func TestPostgreSQLTransitKeyRepository_GetByNameAndVersion_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create a transit key
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "version-specific-key",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Retrieve the key by name and version
	retrievedKey, err := repo.GetByNameAndVersion(ctx, "version-specific-key", 2)
	require.NoError(t, err)
	require.NotNil(t, retrievedKey)

	assert.Equal(t, transitKey.ID, retrievedKey.ID)
	assert.Equal(t, transitKey.Name, retrievedKey.Name)
	assert.Equal(t, transitKey.Version, retrievedKey.Version)
	assert.Equal(t, transitKey.DekID, retrievedKey.DekID)
	assert.WithinDuration(t, transitKey.CreatedAt, retrievedKey.CreatedAt, time.Second)
}

func TestPostgreSQLTransitKeyRepository_GetByNameAndVersion_MultipleVersions(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "versioned-key",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "versioned-key",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	// Create version 3
	time.Sleep(time.Millisecond)
	key3 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "versioned-key",
		Version:   3,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key3)
	require.NoError(t, err)

	// GetByNameAndVersion should return exact version 1
	retrievedKey1, err := repo.GetByNameAndVersion(ctx, "versioned-key", 1)
	require.NoError(t, err)
	require.NotNil(t, retrievedKey1)
	assert.Equal(t, uint(1), retrievedKey1.Version)
	assert.Equal(t, key1.ID, retrievedKey1.ID)

	// GetByNameAndVersion should return exact version 2
	retrievedKey2, err := repo.GetByNameAndVersion(ctx, "versioned-key", 2)
	require.NoError(t, err)
	require.NotNil(t, retrievedKey2)
	assert.Equal(t, uint(2), retrievedKey2.Version)
	assert.Equal(t, key2.ID, retrievedKey2.ID)

	// GetByNameAndVersion should return exact version 3
	retrievedKey3, err := repo.GetByNameAndVersion(ctx, "versioned-key", 3)
	require.NoError(t, err)
	require.NotNil(t, retrievedKey3)
	assert.Equal(t, uint(3), retrievedKey3.Version)
	assert.Equal(t, key3.ID, retrievedKey3.ID)
}

func TestPostgreSQLTransitKeyRepository_GetByNameAndVersion_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1
	transitKey := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "test-key",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, transitKey)
	require.NoError(t, err)

	// Try to get non-existent version
	retrievedKey, err := repo.GetByNameAndVersion(ctx, "test-key", 2)
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)

	// Try to get non-existent name
	retrievedKey, err = repo.GetByNameAndVersion(ctx, "non-existent-key", 1)
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
}

func TestPostgreSQLTransitKeyRepository_GetByNameAndVersion_IgnoresDeletedKeys(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-version-test",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-version-test",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	// Delete version 2
	err = repo.Delete(ctx, key2.ID)
	require.NoError(t, err)

	// GetByNameAndVersion should not find version 2 (it's deleted)
	retrievedKey, err := repo.GetByNameAndVersion(ctx, "deleted-version-test", 2)
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)

	// GetByNameAndVersion should still find version 1 (not deleted)
	retrievedKey, err = repo.GetByNameAndVersion(ctx, "deleted-version-test", 1)
	require.NoError(t, err)
	require.NotNil(t, retrievedKey)
	assert.Equal(t, uint(1), retrievedKey.Version)
	assert.Equal(t, key1.ID, retrievedKey.ID)
}

func TestPostgreSQLTransitKeyRepository_GetByNameAndVersion_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create version 1 outside transaction
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "tx-version-test",
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create version 2 inside transaction
	time.Sleep(time.Millisecond)
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "tx-version-test",
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		key2.ID,
		key2.Name,
		key2.Version,
		key2.DekID,
		key2.CreatedAt,
		key2.DeletedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see version 2
	var retrievedKey transitDomain.TransitKey
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, name, version, dek_id, created_at, deleted_at 
		 FROM transit_keys 
		 WHERE name = $1 AND version = $2 AND deleted_at IS NULL`,
		"tx-version-test",
		2,
	).Scan(
		&retrievedKey.ID,
		&retrievedKey.Name,
		&retrievedKey.Version,
		&retrievedKey.DekID,
		&retrievedKey.CreatedAt,
		&retrievedKey.DeletedAt,
	)
	require.NoError(t, err)
	assert.Equal(t, uint(2), retrievedKey.Version)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Query outside transaction should not see version 2
	retrievedKey2, err := repo.GetByNameAndVersion(ctx, "tx-version-test", 2)
	assert.Error(t, err)
	assert.Nil(t, retrievedKey2)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)

	// But version 1 should still be accessible
	retrievedKey1, err := repo.GetByNameAndVersion(ctx, "tx-version-test", 1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), retrievedKey1.Version)
}

// createTestDek creates a KEK and DEK for testing transit keys.
func createTestDek(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	// Create KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := cryptoRepository.NewPostgreSQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-test",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create DEK
	dekID := uuid.Must(uuid.NewV7())
	dekRepo := cryptoRepository.NewPostgreSQLDekRepository(db)
	dek := &cryptoDomain.Dek{
		ID:           dekID,
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("dek-nonce"),
		CreatedAt:    time.Now().UTC(),
	}
	err = dekRepo.Create(ctx, dek)
	require.NoError(t, err)

	return dekID
}

func TestPostgreSQLTransitKeyRepository_ListCursor_FirstPage(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create KEK and DEK for FK constraint
	kekID := testutil.CreateTestKek(t, db, "postgres", "test-kek")
	dekID := testutil.CreateTestDek(t, db, "postgres", "test-dek", kekID)

	// Create 5 transit keys with different names (alphabetically ordered)
	names := []string{"a-key", "b-key", "c-key", "d-key", "e-key"}
	for _, name := range names {
		key := &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   1,
			DekID:     dekID,
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, key))
	}

	// First page with no cursor (limit 3)
	keys, err := repo.ListCursor(ctx, nil, 3)
	require.NoError(t, err)
	assert.Len(t, keys, 3)

	// Verify ASC ordering by name
	assert.Equal(t, "a-key", keys[0].Name)
	assert.Equal(t, "b-key", keys[1].Name)
	assert.Equal(t, "c-key", keys[2].Name)
}

func TestPostgreSQLTransitKeyRepository_ListCursor_SubsequentPages(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create KEK and DEK for FK constraint
	kekID := testutil.CreateTestKek(t, db, "postgres", "test-kek-2")
	dekID := testutil.CreateTestDek(t, db, "postgres", "test-dek-2", kekID)

	// Create 10 transit keys with alphabetically ordered names
	names := []string{"a-key", "b-key", "c-key", "d-key", "e-key",
		"f-key", "g-key", "h-key", "i-key", "j-key"}
	for _, name := range names {
		key := &transitDomain.TransitKey{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      name,
			Version:   1,
			DekID:     dekID,
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, key))
	}

	// First page (no cursor, limit 3)
	page1, err := repo.ListCursor(ctx, nil, 3)
	require.NoError(t, err)
	require.Len(t, page1, 3)
	assert.Equal(t, "a-key", page1[0].Name)

	// Second page (use last name from page1 as cursor)
	lastNamePage1 := page1[len(page1)-1].Name
	page2, err := repo.ListCursor(ctx, &lastNamePage1, 3)
	require.NoError(t, err)
	require.Len(t, page2, 3)

	// Verify pages don't overlap
	assert.NotEqual(t, page1[len(page1)-1].Name, page2[0].Name)

	// Verify ASC ordering (name > cursor means later names)
	assert.True(t, page2[0].Name > page1[len(page1)-1].Name)
	assert.Equal(t, "d-key", page2[0].Name)

	// Third page
	lastNamePage2 := page2[len(page2)-1].Name
	page3, err := repo.ListCursor(ctx, &lastNamePage2, 3)
	require.NoError(t, err)
	require.Len(t, page3, 3)

	// Verify no overlap
	assert.NotEqual(t, page2[len(page2)-1].Name, page3[0].Name)
	assert.Equal(t, "g-key", page3[0].Name)
}

func TestPostgreSQLTransitKeyRepository_ListCursor_EmptyResult(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	// List with no data
	keys, err := repo.ListCursor(ctx, nil, 10)
	require.NoError(t, err)
	assert.NotNil(t, keys)
	assert.Len(t, keys, 0)
}

func TestPostgreSQLTransitKeyRepository_HardDelete(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDek(t, db)

	// Create keys:
	// 1. Not deleted
	// 2. Deleted 10 days ago
	// 3. Deleted 40 days ago

	now := time.Now().UTC()
	tenDaysAgo := now.AddDate(0, 0, -10)
	fortyDaysAgo := now.AddDate(0, 0, -40)

	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "active-key",
		Version:   1,
		DekID:     dekID,
		CreatedAt: now.AddDate(0, 0, -50),
	}
	require.NoError(t, repo.Create(ctx, key1))

	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-recent",
		Version:   1,
		DekID:     dekID,
		CreatedAt: now.AddDate(0, 0, -50),
	}
	require.NoError(t, repo.Create(ctx, key2))
	_, err := db.ExecContext(ctx, "UPDATE transit_keys SET deleted_at = $1 WHERE id = $2", tenDaysAgo, key2.ID)
	require.NoError(t, err)

	key3 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-old",
		Version:   1,
		DekID:     dekID,
		CreatedAt: now.AddDate(0, 0, -50),
	}
	require.NoError(t, repo.Create(ctx, key3))
	_, err = db.ExecContext(ctx, "UPDATE transit_keys SET deleted_at = $1 WHERE id = $2", fortyDaysAgo, key3.ID)
	require.NoError(t, err)

	cutoff := now.AddDate(0, 0, -30)

	t.Run("DryRun", func(t *testing.T) {
		count, err := repo.HardDelete(ctx, cutoff, true)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify key still exists
		var exists bool
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = $1)", key3.ID).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ActualDelete", func(t *testing.T) {
		count, err := repo.HardDelete(ctx, cutoff, false)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify key3 is gone
		var exists bool
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = $1)", key3.ID).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists)

		// Verify key1 and key2 still exist
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = $1)", key1.ID).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)

		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = $1)", key2.ID).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)
	})
}
