package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/testutil"
)

func TestNewMySQLKekRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLKekRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLKekRepository{}, repo)
}

func TestMySQLKekRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("unique-nonce-12345"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, kek)
	require.NoError(t, err)

	// Verify the KEK was created by listing all KEKs
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 1)

	assert.Equal(t, kek.ID, keks[0].ID)
	assert.Equal(t, kek.MasterKeyID, keks[0].MasterKeyID)
	assert.Equal(t, kek.Algorithm, keks[0].Algorithm)
	assert.Equal(t, kek.EncryptedKey, keks[0].EncryptedKey)
	assert.Equal(t, kek.Nonce, keks[0].Nonce)
	assert.Equal(t, kek.Version, keks[0].Version)
	assert.WithinDuration(t, kek.CreatedAt, keks[0].CreatedAt, time.Second)
}

func TestMySQLKekRepository_Create_WithChaCha20Algorithm(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-2",
		Algorithm:    cryptoDomain.ChaCha20,
		EncryptedKey: []byte("chacha20-encrypted-key"),
		Nonce:        []byte("chacha20-nonce-123"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, kek)
	require.NoError(t, err)

	// Verify the KEK was created with correct algorithm
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 1)
	assert.Equal(t, cryptoDomain.ChaCha20, keks[0].Algorithm)
}

func TestMySQLKekRepository_Create_MultipleVersions(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Create first version (active)
	kek1 := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-v1"),
		Nonce:        []byte("nonce-v1"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, kek1)
	require.NoError(t, err)

	// Create second version (inactive - rotated)
	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7 ordering
	kek2 := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-v2"),
		Nonce:        []byte("nonce-v2"),
		Version:      2,
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, kek2)
	require.NoError(t, err)

	// Verify both KEKs were created
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 2)

	// List should return them ordered by version DESC
	assert.Equal(t, uint(2), keks[0].Version)
	assert.Equal(t, uint(1), keks[1].Version)
}

func TestMySQLKekRepository_Update(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Create initial KEK
	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-encrypted-key"),
		Nonce:        []byte("original-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, kek)
	require.NoError(t, err)

	// Update the KEK (e.g., deactivate after rotation)
	kek.MasterKeyID = "master-key-2"
	kek.EncryptedKey = []byte("updated-encrypted-key")

	err = repo.Update(ctx, kek)
	require.NoError(t, err)

	// Verify the update
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 1)

	assert.Equal(t, kek.ID, keks[0].ID)
	assert.Equal(t, "master-key-2", keks[0].MasterKeyID)
	assert.Equal(t, []byte("updated-encrypted-key"), keks[0].EncryptedKey)
}

func TestMySQLKekRepository_Update_NonExistent(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Try to update a non-existent KEK
	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-key"),
		Nonce:        []byte("nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	// Update should not return an error even if no rows are affected
	err := repo.Update(ctx, kek)
	assert.NoError(t, err)
}

func TestMySQLKekRepository_List_Empty(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	keks, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, keks)
}

func TestMySQLKekRepository_List_OrderedByVersionDesc(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Create multiple KEKs with different versions
	versions := []uint{1, 5, 3, 2, 4}
	for _, version := range versions {
		time.Sleep(time.Millisecond) // Ensure different timestamps
		kek := &cryptoDomain.Kek{
			ID:           uuid.Must(uuid.NewV7()),
			MasterKeyID:  "master-key-1",
			Algorithm:    cryptoDomain.AESGCM,
			EncryptedKey: []byte("encrypted-key"),
			Nonce:        []byte("nonce"),
			Version:      version,
			CreatedAt:    time.Now().UTC(),
		}
		err := repo.Create(ctx, kek)
		require.NoError(t, err)
	}

	// List should return in descending version order
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 5)

	assert.Equal(t, uint(5), keks[0].Version)
	assert.Equal(t, uint(4), keks[1].Version)
	assert.Equal(t, uint(3), keks[2].Version)
	assert.Equal(t, uint(2), keks[3].Version)
	assert.Equal(t, uint(1), keks[4].Version)
}

func TestMySQLKekRepository_List_WithActiveAndInactive(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Create active KEK
	activeKek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("active-key"),
		Nonce:        []byte("active-nonce"),
		Version:      2,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, activeKek)
	require.NoError(t, err)

	time.Sleep(time.Millisecond)

	// Create inactive KEK (rotated out)
	inactiveKek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("inactive-key"),
		Nonce:        []byte("inactive-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, inactiveKek)
	require.NoError(t, err)

	// List should return both
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 2)

	// First should be version 2 (active)
	assert.Equal(t, uint(2), keks[0].Version)

	// Second should be version 1 (inactive)
	assert.Equal(t, uint(1), keks[1].Version)
}

func TestMySQLKekRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-key"),
		Nonce:        []byte("nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	// Test rollback behavior using a function that will fail
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	id, err := kek.ID.MarshalBinary()
	require.NoError(t, err)

	// Create KEK within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the KEK was not created (rollback worked)
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, keks)
}

func TestMySQLKekRepository_Update_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Create initial KEK
	kek := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-key"),
		Nonce:        []byte("original-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, kek)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	id, err := kek.ID.MarshalBinary()
	require.NoError(t, err)

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE keks 
			  SET master_key_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  version = ?, 
				  created_at = ?
			  WHERE id = ?`,
		"master-key-2",
		kek.Algorithm,
		[]byte("updated-key"),
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
		id,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the KEK was not updated (rollback worked)
	keks, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, keks, 1)
	assert.Equal(t, "master-key-1", keks[0].MasterKeyID, "KEK should have original master key after rollback")
}

func TestMySQLKekRepository_List_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLKekRepository(db)
	ctx := context.Background()

	// Create a KEK outside transaction
	kek1 := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("key-1"),
		Nonce:        []byte("nonce-1"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, kek1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create another KEK inside transaction
	time.Sleep(time.Millisecond)
	kek2 := &cryptoDomain.Kek{
		ID:           uuid.Must(uuid.NewV7()),
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("key-2"),
		Nonce:        []byte("nonce-2"),
		Version:      2,
		CreatedAt:    time.Now().UTC(),
	}

	id, err := kek2.ID.MarshalBinary()
	require.NoError(t, err)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		kek2.MasterKeyID,
		kek2.Algorithm,
		kek2.EncryptedKey,
		kek2.Nonce,
		kek2.Version,
		kek2.CreatedAt,
	)
	require.NoError(t, err)

	// List within transaction should see both KEKs
	rows, err := tx.QueryContext(
		ctx,
		`SELECT id, master_key_id, algorithm, encrypted_key, nonce, version, created_at 
		 FROM keks ORDER BY version DESC`,
	)
	require.NoError(t, err)
	defer func() {
		_ = rows.Close()
	}()

	var keks []*cryptoDomain.Kek
	for rows.Next() {
		var kek cryptoDomain.Kek
		var kekID []byte

		err := rows.Scan(
			&kekID,
			&kek.MasterKeyID,
			&kek.Algorithm,
			&kek.EncryptedKey,
			&kek.Nonce,
			&kek.Version,
			&kek.CreatedAt,
		)
		require.NoError(t, err)

		err = kek.ID.UnmarshalBinary(kekID)
		require.NoError(t, err)

		keks = append(keks, &kek)
	}
	assert.Len(t, keks, 2)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// List outside transaction should also see both KEKs
	keks, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, keks, 2)
}
