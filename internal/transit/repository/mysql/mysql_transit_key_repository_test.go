//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoRepository "github.com/allisson/secrets/internal/crypto/repository/mysql"
	"github.com/allisson/secrets/internal/testutil"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

func TestNewMySQLTransitKeyRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLTransitKeyRepository{}, repo)
}

func TestMySQLTransitKeyRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create prerequisite KEK and DEK
	dekID := createTestDekMySQL(t, db)

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
	var id, dekIDBytes []byte
	query := `SELECT id, name, version, dek_id, created_at, deleted_at FROM transit_keys WHERE id = ?`

	transitKeyIDBytes, err := transitKey.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, transitKeyIDBytes).Scan(
		&id,
		&readKey.Name,
		&readKey.Version,
		&dekIDBytes,
		&readKey.CreatedAt,
		&readKey.DeletedAt,
	)
	require.NoError(t, err)

	err = readKey.ID.UnmarshalBinary(id)
	require.NoError(t, err)
	err = readKey.DekID.UnmarshalBinary(dekIDBytes)
	require.NoError(t, err)

	assert.Equal(t, transitKey.ID, readKey.ID)
	assert.Equal(t, transitKey.Name, readKey.Name)
	assert.Equal(t, transitKey.Version, readKey.Version)
	assert.Equal(t, transitKey.DekID, readKey.DekID)
	assert.WithinDuration(t, transitKey.CreatedAt, readKey.CreatedAt, time.Second)
	assert.Nil(t, readKey.DeletedAt)
}

func TestMySQLTransitKeyRepository_Create_MultipleVersions(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transit_keys WHERE name = ?`, "api-encryption").
		Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMySQLTransitKeyRepository_Delete(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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
	transitKeyIDBytes, err := transitKey.ID.MarshalBinary()
	require.NoError(t, err)

	query := `SELECT deleted_at FROM transit_keys WHERE id = ?`
	err = db.QueryRowContext(ctx, query, transitKeyIDBytes).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt, "deleted_at should be set after soft delete")
}

func TestMySQLTransitKeyRepository_GetByName_Success(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByName_LatestVersion(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByName_NotFound(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	// Try to get a non-existent key
	retrievedKey, err := repo.GetByName(ctx, "non-existent-key")
	assert.Error(t, err)
	assert.Nil(t, retrievedKey)
	assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
}

func TestMySQLTransitKeyRepository_GetByName_IgnoresDeletedKeys(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByName_AllVersionsDeleted(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

	// Marshal UUIDs
	id, err := transitKey.ID.MarshalBinary()
	require.NoError(t, err)
	dekIDBytes, err := transitKey.DekID.MarshalBinary()
	require.NoError(t, err)

	// Create transit key within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id,
		transitKey.Name,
		transitKey.Version,
		dekIDBytes,
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

func TestMySQLTransitKeyRepository_Delete_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

	// Marshal UUID
	id, err := transitKey.ID.MarshalBinary()
	require.NoError(t, err)

	// Delete within transaction
	_, err = tx.ExecContext(ctx, `UPDATE transit_keys SET deleted_at = NOW() WHERE id = ?`, id)
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

func TestMySQLTransitKeyRepository_GetByName_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

	// Marshal UUIDs
	id, err := key2.ID.MarshalBinary()
	require.NoError(t, err)
	dekIDBytes, err := key2.DekID.MarshalBinary()
	require.NoError(t, err)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id,
		key2.Name,
		key2.Version,
		dekIDBytes,
		key2.CreatedAt,
		key2.DeletedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see version 2
	var retrievedKey transitDomain.TransitKey
	var idBytes, dekIDResult []byte
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, name, version, dek_id, created_at, deleted_at 
		 FROM transit_keys 
		 WHERE name = ? AND deleted_at IS NULL 
		 ORDER BY version DESC 
		 LIMIT 1`,
		"tx-read-test",
	).Scan(
		&idBytes,
		&retrievedKey.Name,
		&retrievedKey.Version,
		&dekIDResult,
		&retrievedKey.CreatedAt,
		&retrievedKey.DeletedAt,
	)
	require.NoError(t, err)

	err = retrievedKey.ID.UnmarshalBinary(idBytes)
	require.NoError(t, err)
	err = retrievedKey.DekID.UnmarshalBinary(dekIDResult)
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

func TestMySQLTransitKeyRepository_GetByNameAndVersion_Success(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByNameAndVersion_MultipleVersions(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByNameAndVersion_NotFound(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByNameAndVersion_IgnoresDeletedKeys(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

func TestMySQLTransitKeyRepository_GetByNameAndVersion_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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

	// Marshal UUIDs
	id, err := key2.ID.MarshalBinary()
	require.NoError(t, err)
	dekIDBytes, err := key2.DekID.MarshalBinary()
	require.NoError(t, err)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id,
		key2.Name,
		key2.Version,
		dekIDBytes,
		key2.CreatedAt,
		key2.DeletedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see version 2
	var retrievedKey transitDomain.TransitKey
	var idBytes, dekIDResult []byte
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, name, version, dek_id, created_at, deleted_at 
		 FROM transit_keys 
		 WHERE name = ? AND version = ? AND deleted_at IS NULL`,
		"tx-version-test",
		2,
	).Scan(
		&idBytes,
		&retrievedKey.Name,
		&retrievedKey.Version,
		&dekIDResult,
		&retrievedKey.CreatedAt,
		&retrievedKey.DeletedAt,
	)
	require.NoError(t, err)

	err = retrievedKey.ID.UnmarshalBinary(idBytes)
	require.NoError(t, err)
	err = retrievedKey.DekID.UnmarshalBinary(dekIDResult)
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

// createTestDekMySQL creates a KEK and DEK for testing transit keys with MySQL.
func createTestDekMySQL(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	// Create KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := cryptoRepository.NewMySQLKekRepository(db)
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
	dekRepo := cryptoRepository.NewMySQLDekRepository(db)
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

func TestMySQLTransitKeyRepository_GetTransitKey(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create prerequisite KEK and DEK
	dekID := createTestDekMySQL(t, db)
	algorithm := cryptoDomain.AESGCM

	name := "test-key"

	// Create version 1
	key1 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Version:   1,
		DekID:     dekID,
		CreatedAt: time.Now().UTC(),
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create version 2
	key2 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      name,
		Version:   2,
		DekID:     dekID,
		CreatedAt: time.Now().UTC().Add(time.Hour),
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	t.Run("Get latest version", func(t *testing.T) {
		tk, alg, err := repo.GetTransitKey(ctx, name, 0)
		require.NoError(t, err)
		assert.Equal(t, key2.ID, tk.ID)
		assert.Equal(t, name, tk.Name)
		assert.Equal(t, uint(2), tk.Version)
		assert.Equal(t, algorithm, alg)
	})

	t.Run("Get specific version", func(t *testing.T) {
		tk, alg, err := repo.GetTransitKey(ctx, name, 1)
		require.NoError(t, err)
		assert.Equal(t, key1.ID, tk.ID)
		assert.Equal(t, name, tk.Name)
		assert.Equal(t, uint(1), tk.Version)
		assert.Equal(t, algorithm, alg)
	})

	t.Run("Key not found", func(t *testing.T) {
		tk, alg, err := repo.GetTransitKey(ctx, "non-existent", 0)
		assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
		assert.Nil(t, tk)
		assert.Empty(t, alg)
	})

	t.Run("Version not found", func(t *testing.T) {
		tk, alg, err := repo.GetTransitKey(ctx, name, 3)
		assert.ErrorIs(t, err, transitDomain.ErrTransitKeyNotFound)
		assert.Nil(t, tk)
		assert.Empty(t, alg)
	})
}

func TestMySQLTransitKeyRepository_ListCursor_FirstPage(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create KEK and DEK for FK constraint
	kekID := testutil.CreateTestKek(t, db, "mysql", "test-kek")
	dekID := testutil.CreateTestDek(t, db, "mysql", "test-dek", kekID)

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

func TestMySQLTransitKeyRepository_ListCursor_SubsequentPages(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	// Create KEK and DEK for FK constraint
	kekID := testutil.CreateTestKek(t, db, "mysql", "test-kek-2")
	dekID := testutil.CreateTestDek(t, db, "mysql", "test-dek-2", kekID)

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

func TestMySQLTransitKeyRepository_ListCursor_EmptyResult(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	// List with no data
	keys, err := repo.ListCursor(ctx, nil, 10)
	require.NoError(t, err)
	assert.NotNil(t, keys)
	assert.Len(t, keys, 0)
}

func TestMySQLTransitKeyRepository_HardDelete(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTransitKeyRepository(db)
	ctx := context.Background()

	dekID := createTestDekMySQL(t, db)

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
	key2IDBytes, _ := key2.ID.MarshalBinary()
	_, err := db.ExecContext(ctx, "UPDATE transit_keys SET deleted_at = ? WHERE id = ?", tenDaysAgo, key2IDBytes)
	require.NoError(t, err)

	key3 := &transitDomain.TransitKey{
		ID:        uuid.Must(uuid.NewV7()),
		Name:      "deleted-old",
		Version:   1,
		DekID:     dekID,
		CreatedAt: now.AddDate(0, 0, -50),
	}
	require.NoError(t, repo.Create(ctx, key3))
	key3IDBytes, _ := key3.ID.MarshalBinary()
	_, err = db.ExecContext(ctx, "UPDATE transit_keys SET deleted_at = ? WHERE id = ?", fortyDaysAgo, key3IDBytes)
	require.NoError(t, err)

	cutoff := now.AddDate(0, 0, -30)

	t.Run("DryRun", func(t *testing.T) {
		count, err := repo.HardDelete(ctx, cutoff, true)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify key still exists
		var exists bool
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = ?)", key3IDBytes).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ActualDelete", func(t *testing.T) {
		count, err := repo.HardDelete(ctx, cutoff, false)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify key3 is gone
		var exists bool
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = ?)", key3IDBytes).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists)

		// Verify key1 and key2 still exist
		key1IDBytes, _ := key1.ID.MarshalBinary()
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = ?)", key1IDBytes).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)

		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM transit_keys WHERE id = ?)", key2IDBytes).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)
	})
}
