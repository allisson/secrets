package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoRepository "github.com/allisson/secrets/internal/crypto/repository"
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
