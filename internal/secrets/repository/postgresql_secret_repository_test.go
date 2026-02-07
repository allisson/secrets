package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoRepository "github.com/allisson/secrets/internal/crypto/repository"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	"github.com/allisson/secrets/internal/testutil"
)

func TestNewPostgreSQLSecretRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLSecretRepository{}, repo)
}

func TestPostgreSQLSecretRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	// Create dependencies (KEK and DEK)
	_, dekID := createKekAndDek(t, db)

	// Create secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/database/password",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-secret-data"),
		Nonce:      []byte("unique-nonce-12345"),
		CreatedAt:  time.Now().UTC(),
		DeletedAt:  nil,
	}

	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Verify the secret was created by reading it back
	var readSecret secretsDomain.Secret
	query := `SELECT id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at 
			  FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(
		&readSecret.ID,
		&readSecret.Path,
		&readSecret.Version,
		&readSecret.DekID,
		&readSecret.Ciphertext,
		&readSecret.Nonce,
		&readSecret.CreatedAt,
		&readSecret.DeletedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, secret.ID, readSecret.ID)
	assert.Equal(t, secret.Path, readSecret.Path)
	assert.Equal(t, secret.Version, readSecret.Version)
	assert.Equal(t, secret.DekID, readSecret.DekID)
	assert.Equal(t, secret.Ciphertext, readSecret.Ciphertext)
	assert.Equal(t, secret.Nonce, readSecret.Nonce)
	assert.WithinDuration(t, secret.CreatedAt, readSecret.CreatedAt, time.Second)
	assert.Nil(t, readSecret.DeletedAt)
}

func TestPostgreSQLSecretRepository_Create_WithDifferentPaths(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	paths := []string{
		"/app/api/key",
		"/database/credentials",
		"/aws/access-key",
		"/stripe/secret-key",
	}

	for _, path := range paths {
		time.Sleep(time.Millisecond) // Ensure different timestamps for UUIDv7 ordering
		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       path,
			Version:    1,
			DekID:      dekID,
			Ciphertext: []byte("encrypted-data-" + path),
			Nonce:      []byte("nonce-" + path),
			CreatedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, secret)
		require.NoError(t, err, "failed to create secret with path: %s", path)
	}

	// Verify all secrets were created
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secrets`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(paths), count)
}

func TestPostgreSQLSecretRepository_Create_MultipleVersions(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	path := "/app/secret"

	// Create version 1
	secret1 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-v1"),
		Nonce:      []byte("nonce-v1"),
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret1)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	secret2 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    2,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-v2"),
		Nonce:      []byte("nonce-v2"),
		CreatedAt:  time.Now().UTC(),
	}

	err = repo.Create(ctx, secret2)
	require.NoError(t, err)

	// Verify both versions exist
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secrets WHERE path = $1`, path).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPostgreSQLSecretRepository_Create_DuplicateID(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	secretID := uuid.Must(uuid.NewV7())

	// Create first secret
	secret1 := &secretsDomain.Secret{
		ID:         secretID,
		Path:       "/app/secret1",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-1"),
		Nonce:      []byte("nonce-1"),
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret1)
	require.NoError(t, err)

	// Try to create another secret with the same ID
	secret2 := &secretsDomain.Secret{
		ID:         secretID, // Same ID
		Path:       "/app/secret2",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-2"),
		Nonce:      []byte("nonce-2"),
		CreatedAt:  time.Now().UTC(),
	}

	err = repo.Create(ctx, secret2)
	assert.Error(t, err, "should fail due to duplicate primary key")
}

func TestPostgreSQLSecretRepository_Create_DuplicatePathAndVersion(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	path := "/app/secret"
	version := uint(1)

	// Create first secret
	secret1 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    version,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-1"),
		Nonce:      []byte("nonce-1"),
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret1)
	require.NoError(t, err)

	// Try to create another secret with the same path and version
	time.Sleep(time.Millisecond)
	secret2 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    version,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-2"),
		Nonce:      []byte("nonce-2"),
		CreatedAt:  time.Now().UTC(),
	}

	err = repo.Create(ctx, secret2)
	assert.Error(t, err, "should fail due to unique constraint on (path, version)")
}

func TestPostgreSQLSecretRepository_Create_WithInvalidDekID(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	// Try to create secret with non-existent DEK ID
	nonExistentDekID := uuid.Must(uuid.NewV7())
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      nonExistentDekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret)
	assert.Error(t, err, "should fail due to foreign key constraint violation")
}

func TestPostgreSQLSecretRepository_Create_WithBinaryData(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create secret with binary data including null bytes and special characters
	ciphertext := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x80, 0x7F, 0xAA, 0xBB}
	nonce := []byte{0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/binary-secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Verify binary data is stored correctly
	var readSecret secretsDomain.Secret
	query := `SELECT id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at 
			  FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(
		&readSecret.ID,
		&readSecret.Path,
		&readSecret.Version,
		&readSecret.DekID,
		&readSecret.Ciphertext,
		&readSecret.Nonce,
		&readSecret.CreatedAt,
		&readSecret.DeletedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, ciphertext, readSecret.Ciphertext, "binary ciphertext should be preserved exactly")
	assert.Equal(t, nonce, readSecret.Nonce, "binary nonce should be preserved exactly")
}

func TestPostgreSQLSecretRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	// Test rollback behavior
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create secret within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO secrets (id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		secret.ID,
		secret.Path,
		secret.Version,
		secret.DekID,
		secret.Ciphertext,
		secret.Nonce,
		secret.CreatedAt,
		secret.DeletedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the secret was not created (rollback worked)
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secrets WHERE id = $1`, secret.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "secret should not exist after rollback")
}

func TestPostgreSQLSecretRepository_Create_WithTransactionCommit(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create secret within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO secrets (id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		secret.ID,
		secret.Path,
		secret.Version,
		secret.DekID,
		secret.Ciphertext,
		secret.Nonce,
		secret.CreatedAt,
		secret.DeletedAt,
	)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify the secret was created (commit worked)
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secrets WHERE id = $1`, secret.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "secret should exist after commit")
}

func TestPostgreSQLSecretRepository_Delete(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Verify the secret exists and is not deleted
	var deletedAt *time.Time
	query := `SELECT deleted_at FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.Nil(t, deletedAt, "secret should not be deleted initially")

	// Delete the secret (soft delete)
	err = repo.Delete(ctx, secret.ID)
	require.NoError(t, err)

	// Verify the secret still exists but has deleted_at set
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt, "secret should have deleted_at timestamp")
	assert.WithinDuration(t, time.Now().UTC(), *deletedAt, 2*time.Second)

	// Verify the secret row still exists in the database
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secrets WHERE id = $1`, secret.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "secret row should still exist after soft delete")
}

func TestPostgreSQLSecretRepository_Delete_NonExistent(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	// Try to delete a non-existent secret
	nonExistentID := uuid.Must(uuid.NewV7())

	// Delete should not return an error even if no rows are affected
	err := repo.Delete(ctx, nonExistentID)
	assert.NoError(t, err)
}

func TestPostgreSQLSecretRepository_Delete_AlreadyDeleted(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Delete the secret first time
	err = repo.Delete(ctx, secret.ID)
	require.NoError(t, err)

	// Get the first deletion timestamp
	var firstDeletedAt *time.Time
	query := `SELECT deleted_at FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(&firstDeletedAt)
	require.NoError(t, err)
	require.NotNil(t, firstDeletedAt)

	time.Sleep(100 * time.Millisecond)

	// Delete the secret second time (should update deleted_at)
	err = repo.Delete(ctx, secret.ID)
	require.NoError(t, err)

	// Get the second deletion timestamp
	var secondDeletedAt *time.Time
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(&secondDeletedAt)
	require.NoError(t, err)
	require.NotNil(t, secondDeletedAt)

	// The second deletion should have a newer timestamp
	assert.True(t, secondDeletedAt.After(*firstDeletedAt), "second delete should update timestamp")
}

func TestPostgreSQLSecretRepository_Delete_MultipleSecrets(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create multiple secrets
	secretIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond)
		secret := &secretsDomain.Secret{
			ID:         uuid.Must(uuid.NewV7()),
			Path:       fmt.Sprintf("/app/secret-%d", i),
			Version:    1,
			DekID:      dekID,
			Ciphertext: []byte(fmt.Sprintf("encrypted-data-%d", i)),
			Nonce:      []byte(fmt.Sprintf("nonce-%d", i)),
			CreatedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, secret)
		require.NoError(t, err)
		secretIDs[i] = secret.ID
	}

	// Delete only the first and third secrets
	err := repo.Delete(ctx, secretIDs[0])
	require.NoError(t, err)

	err = repo.Delete(ctx, secretIDs[2])
	require.NoError(t, err)

	// Verify deletion status
	var deletedAt *time.Time

	// First secret should be deleted
	query := `SELECT deleted_at FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secretIDs[0]).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt)

	// Second secret should NOT be deleted
	err = db.QueryRowContext(ctx, query, secretIDs[1]).Scan(&deletedAt)
	require.NoError(t, err)
	assert.Nil(t, deletedAt)

	// Third secret should be deleted
	err = db.QueryRowContext(ctx, query, secretIDs[2]).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt)
}

func TestPostgreSQLSecretRepository_Delete_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	err := db.QueryRowContext(ctx,
		`INSERT INTO secrets (id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		secret.ID, secret.Path, secret.Version, secret.DekID, secret.Ciphertext,
		secret.Nonce, secret.CreatedAt, secret.DeletedAt).Scan(&secret.ID)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Delete within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE secrets 
			  SET deleted_at = $1
			  WHERE id = $2`,
		time.Now().UTC(),
		secret.ID,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the secret was not deleted (rollback worked)
	var deletedAt *time.Time
	query := `SELECT deleted_at FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.Nil(t, deletedAt, "secret should not be deleted after rollback")
}

func TestPostgreSQLSecretRepository_Delete_WithTransactionCommit(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}

	err := db.QueryRowContext(ctx,
		`INSERT INTO secrets (id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		secret.ID, secret.Path, secret.Version, secret.DekID, secret.Ciphertext,
		secret.Nonce, secret.CreatedAt, secret.DeletedAt).Scan(&secret.ID)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Delete within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE secrets 
			  SET deleted_at = $1
			  WHERE id = $2`,
		time.Now().UTC(),
		secret.ID,
	)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify the secret was deleted (commit worked)
	var deletedAt *time.Time
	query := `SELECT deleted_at FROM secrets WHERE id = $1`
	err = db.QueryRowContext(ctx, query, secret.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt, "secret should be deleted after commit")
}

// Helper functions

// createKekAndDek creates a KEK and DEK for testing and returns their IDs
func createKekAndDek(t *testing.T, db *sql.DB) (kekID uuid.UUID, dekID uuid.UUID) {
	t.Helper()

	ctx := context.Background()

	// Create KEK
	kekID = uuid.Must(uuid.NewV7())
	kekRepo := cryptoRepository.NewPostgreSQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce-12345"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create DEK
	dekID = createDek(t, db, kekID)

	return kekID, dekID
}

// createDek creates a DEK for testing and returns its ID
func createDek(t *testing.T, db *sql.DB, kekID uuid.UUID) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	dekID := uuid.Must(uuid.NewV7())

	dekRepo := cryptoRepository.NewPostgreSQLDekRepository(db)
	dek := &cryptoDomain.Dek{
		ID:           dekID,
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("dek-nonce-12345"),
		CreatedAt:    time.Now().UTC(),
	}
	err := dekRepo.Create(ctx, dek)
	require.NoError(t, err)

	return dekID
}

func TestPostgreSQLSecretRepository_GetByPath(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/database/password",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-secret-data"),
		Nonce:      []byte("unique-nonce-12345"),
		CreatedAt:  time.Now().UTC(),
		DeletedAt:  nil,
	}
	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Get the secret by path
	retrievedSecret, err := repo.GetByPath(ctx, "/app/database/password")
	require.NoError(t, err)
	assert.NotNil(t, retrievedSecret)

	// Verify all fields
	assert.Equal(t, secret.ID, retrievedSecret.ID)
	assert.Equal(t, secret.Path, retrievedSecret.Path)
	assert.Equal(t, secret.Version, retrievedSecret.Version)
	assert.Equal(t, secret.DekID, retrievedSecret.DekID)
	assert.Equal(t, secret.Ciphertext, retrievedSecret.Ciphertext)
	assert.Equal(t, secret.Nonce, retrievedSecret.Nonce)
	assert.WithinDuration(t, secret.CreatedAt, retrievedSecret.CreatedAt, time.Second)
	assert.Nil(t, retrievedSecret.DeletedAt)
}

func TestPostgreSQLSecretRepository_GetByPath_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	// Try to get a non-existent secret
	secret, err := repo.GetByPath(ctx, "/non/existent/path")

	assert.Error(t, err)
	assert.Nil(t, secret)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestPostgreSQLSecretRepository_GetByPath_MultipleVersions(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	path := "/app/versioned-secret"

	// Create version 1
	secret1 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-v1"),
		Nonce:      []byte("nonce-v1"),
		CreatedAt:  time.Now().UTC(),
	}
	err := repo.Create(ctx, secret1)
	require.NoError(t, err)

	// Create version 2
	time.Sleep(time.Millisecond)
	secret2 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    2,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-v2"),
		Nonce:      []byte("nonce-v2"),
		CreatedAt:  time.Now().UTC(),
	}
	err = repo.Create(ctx, secret2)
	require.NoError(t, err)

	// Create version 3
	time.Sleep(time.Millisecond)
	secret3 := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       path,
		Version:    3,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-v3"),
		Nonce:      []byte("nonce-v3"),
		CreatedAt:  time.Now().UTC(),
	}
	err = repo.Create(ctx, secret3)
	require.NoError(t, err)

	// GetByPath should return the highest version (version 3)
	retrievedSecret, err := repo.GetByPath(ctx, path)
	require.NoError(t, err)
	assert.NotNil(t, retrievedSecret)
	assert.Equal(t, secret3.ID, retrievedSecret.ID)
	assert.Equal(t, uint(3), retrievedSecret.Version)
	assert.Equal(t, []byte("encrypted-v3"), retrievedSecret.Ciphertext)
}

func TestPostgreSQLSecretRepository_GetByPath_WithDeletedSecret(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/deleted-secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}
	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Delete the secret
	err = repo.Delete(ctx, secret.ID)
	require.NoError(t, err)

	// GetByPath should still return the secret (including deleted_at timestamp)
	retrievedSecret, err := repo.GetByPath(ctx, "/app/deleted-secret")
	require.NoError(t, err)
	assert.NotNil(t, retrievedSecret)
	assert.NotNil(t, retrievedSecret.DeletedAt)
	assert.WithinDuration(t, time.Now().UTC(), *retrievedSecret.DeletedAt, 2*time.Second)
}

func TestPostgreSQLSecretRepository_GetByPath_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLSecretRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDek(t, db)

	// Create a secret
	secret := &secretsDomain.Secret{
		ID:         uuid.Must(uuid.NewV7()),
		Path:       "/app/transaction-secret",
		Version:    1,
		DekID:      dekID,
		Ciphertext: []byte("encrypted-data"),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
	}
	err := repo.Create(ctx, secret)
	require.NoError(t, err)

	// Use TxManager to get the secret within a transaction
	txManager := database.NewTxManager(db)
	var retrievedSecret *secretsDomain.Secret

	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
		var txErr error
		retrievedSecret, txErr = repo.GetByPath(txCtx, "/app/transaction-secret")
		return txErr
	})

	require.NoError(t, err)
	assert.NotNil(t, retrievedSecret)
	assert.Equal(t, secret.ID, retrievedSecret.ID)
	assert.Equal(t, secret.Path, retrievedSecret.Path)
}
