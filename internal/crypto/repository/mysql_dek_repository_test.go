package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	"github.com/allisson/secrets/internal/testutil"
)

func TestNewMySQLDekRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLDekRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLDekRepository{}, repo)
}

func TestMySQLDekRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// First create a KEK that the DEK will reference
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
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
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("unique-nonce-12345"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Verify the DEK was created by reading it back
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	err = readDek.ID.UnmarshalBinary(dekIDBytes)
	require.NoError(t, err)
	err = readDek.KekID.UnmarshalBinary(kekIDBytes)
	require.NoError(t, err)

	assert.Equal(t, dek.ID, readDek.ID)
	assert.Equal(t, dek.KekID, readDek.KekID)
	assert.Equal(t, dek.Algorithm, readDek.Algorithm)
	assert.Equal(t, dek.EncryptedKey, readDek.EncryptedKey)
	assert.Equal(t, dek.Nonce, readDek.Nonce)
	assert.WithinDuration(t, dek.CreatedAt, readDek.CreatedAt, time.Second)
}

func TestMySQLDekRepository_Create_WithChaCha20Algorithm(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK for the DEK to reference
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.ChaCha20,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create DEK with ChaCha20 algorithm
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.ChaCha20,
		EncryptedKey: []byte("chacha20-encrypted-key"),
		Nonce:        []byte("chacha20-nonce-123"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Verify the DEK was created with correct algorithm
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	err = readDek.ID.UnmarshalBinary(dekIDBytes)
	require.NoError(t, err)

	assert.Equal(t, cryptoDomain.ChaCha20, readDek.Algorithm)
}

func TestMySQLDekRepository_Create_MultipleDeksForSameKek(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create first DEK
	dek1 := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-1"),
		Nonce:        []byte("nonce-1"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek1)
	require.NoError(t, err)

	// Create second DEK with same KEK
	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7 ordering
	dek2 := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-2"),
		Nonce:        []byte("nonce-2"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek2)
	require.NoError(t, err)

	// Verify both DEKs were created
	kekIDBytes, err := kekID.MarshalBinary()
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deks WHERE kek_id = ?`, kekIDBytes).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMySQLDekRepository_Create_DuplicateID(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create first DEK
	dekID := uuid.Must(uuid.NewV7())
	dek1 := &cryptoDomain.Dek{
		ID:           dekID,
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-1"),
		Nonce:        []byte("nonce-1"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek1)
	require.NoError(t, err)

	// Try to create another DEK with the same ID
	dek2 := &cryptoDomain.Dek{
		ID:           dekID, // Same ID as dek1
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-2"),
		Nonce:        []byte("nonce-2"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek2)
	assert.Error(t, err, "should fail due to duplicate primary key")
}

func TestMySQLDekRepository_Create_WithInvalidKekID(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Try to create DEK with non-existent KEK ID (should fail due to foreign key constraint)
	nonExistentKekID := uuid.Must(uuid.NewV7())
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        nonExistentKekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	err := repo.Create(ctx, dek)
	assert.Error(t, err, "should fail due to foreign key constraint violation")
}

func TestMySQLDekRepository_Create_WithBinaryData(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create DEK with binary data including null bytes and special characters
	encryptedKey := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x80, 0x7F}
	nonce := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: encryptedKey,
		Nonce:        nonce,
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Verify binary data is stored correctly
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, encryptedKey, readDek.EncryptedKey, "binary encrypted key should be preserved exactly")
	assert.Equal(t, nonce, readDek.Nonce, "binary nonce should be preserved exactly")
}

func TestMySQLDekRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	// Create a KEK outside the transaction
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Verify KEK exists before starting transaction
	kekIDBytes, err := kekID.MarshalBinary()
	require.NoError(t, err)
	var kekExists int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM keks WHERE id = ?`, kekIDBytes).Scan(&kekExists)
	require.NoError(t, err)
	require.Equal(t, 1, kekExists, "KEK must exist before creating DEK")

	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-key"),
		Nonce:        []byte("nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	// Test rollback behavior
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Marshal DEK ID
	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	// Create DEK within transaction (directly using tx.ExecContext)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`,
		dekID,
		kekIDBytes,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the DEK was not created (rollback worked)
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deks WHERE id = ?`, dekID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "DEK should not exist after rollback")
}

func TestMySQLDekRepository_Create_WithTransactionCommit(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-key"),
		Nonce:        []byte("nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Marshal IDs
	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)
	kekIDBytes, err := dek.KekID.MarshalBinary()
	require.NoError(t, err)

	// Create DEK within transaction (directly using tx.ExecContext)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`,
		dekID,
		kekIDBytes,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
	)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify the DEK was created (commit worked)
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deks WHERE id = ?`, dekID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "DEK should exist after commit")
}

func TestMySQLDekRepository_Create_MultipleInTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create multiple DEKs within the same transaction
	dek1 := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-key-1"),
		Nonce:        []byte("nonce-1"),
		CreatedAt:    time.Now().UTC(),
	}

	time.Sleep(time.Millisecond)
	dek2 := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.ChaCha20,
		EncryptedKey: []byte("encrypted-key-2"),
		Nonce:        []byte("nonce-2"),
		CreatedAt:    time.Now().UTC(),
	}

	// Marshal IDs
	dek1ID, err := dek1.ID.MarshalBinary()
	require.NoError(t, err)
	dek2ID, err := dek2.ID.MarshalBinary()
	require.NoError(t, err)
	kekIDBytes, err := kekID.MarshalBinary()
	require.NoError(t, err)

	// Insert first DEK
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`,
		dek1ID,
		kekIDBytes,
		dek1.Algorithm,
		dek1.EncryptedKey,
		dek1.Nonce,
		dek1.CreatedAt,
	)
	require.NoError(t, err)

	// Insert second DEK
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`,
		dek2ID,
		kekIDBytes,
		dek2.Algorithm,
		dek2.EncryptedKey,
		dek2.Nonce,
		dek2.CreatedAt,
	)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify both DEKs were created
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deks WHERE kek_id = ?`, kekIDBytes).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "both DEKs should exist after commit")
}

func TestMySQLDekRepository_Update(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create initial DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-encrypted-key"),
		Nonce:        []byte("original-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Update the DEK
	dek.EncryptedKey = []byte("updated-encrypted-key")
	dek.Nonce = []byte("updated-nonce")

	err = repo.Update(ctx, dek)
	require.NoError(t, err)

	// Verify the update
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	err = readDek.ID.UnmarshalBinary(dekIDBytes)
	require.NoError(t, err)
	err = readDek.KekID.UnmarshalBinary(kekIDBytes)
	require.NoError(t, err)

	assert.Equal(t, dek.ID, readDek.ID)
	assert.Equal(t, dek.KekID, readDek.KekID)
	assert.Equal(t, []byte("updated-encrypted-key"), readDek.EncryptedKey)
	assert.Equal(t, []byte("updated-nonce"), readDek.Nonce)
}

func TestMySQLDekRepository_Update_ChangeKek(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create first KEK
	kek1ID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek1 := &cryptoDomain.Kek{
		ID:           kek1ID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-1"),
		Nonce:        []byte("kek-nonce-1"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek1)
	require.NoError(t, err)

	// Create second KEK
	time.Sleep(time.Millisecond)
	kek2ID := uuid.Must(uuid.NewV7())
	kek2 := &cryptoDomain.Kek{
		ID:           kek2ID,
		MasterKeyID:  "master-key-2",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-2"),
		Nonce:        []byte("kek-nonce-2"),
		Version:      2,
		CreatedAt:    time.Now().UTC(),
	}
	err = kekRepo.Create(ctx, kek2)
	require.NoError(t, err)

	// Create DEK with first KEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kek1ID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-encrypted-key"),
		Nonce:        []byte("original-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Update DEK to use second KEK (simulating key rotation)
	dek.KekID = kek2ID
	dek.EncryptedKey = []byte("re-encrypted-with-kek2")
	dek.Nonce = []byte("new-nonce")

	err = repo.Update(ctx, dek)
	require.NoError(t, err)

	// Verify the KEK was changed
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	err = readDek.KekID.UnmarshalBinary(kekIDBytes)
	require.NoError(t, err)

	assert.Equal(t, kek2ID, readDek.KekID, "DEK should reference the new KEK")
	assert.Equal(t, []byte("re-encrypted-with-kek2"), readDek.EncryptedKey)
}

func TestMySQLDekRepository_Update_ChangeAlgorithm(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create DEK with AES-GCM
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("aes-encrypted-key"),
		Nonce:        []byte("aes-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Update to ChaCha20
	dek.Algorithm = cryptoDomain.ChaCha20
	dek.EncryptedKey = []byte("chacha20-encrypted-key")
	dek.Nonce = []byte("chacha20-nonce")

	err = repo.Update(ctx, dek)
	require.NoError(t, err)

	// Verify the algorithm was changed
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, cryptoDomain.ChaCha20, readDek.Algorithm, "algorithm should be changed to ChaCha20")
	assert.Equal(t, []byte("chacha20-encrypted-key"), readDek.EncryptedKey)
}

func TestMySQLDekRepository_Update_NonExistent(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Try to update a non-existent DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-key"),
		Nonce:        []byte("nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	// Update should not return an error even if no rows are affected
	err = repo.Update(ctx, dek)
	assert.NoError(t, err)
}

func TestMySQLDekRepository_Update_WithInvalidKekID(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-encrypted-key"),
		Nonce:        []byte("original-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Try to update DEK with non-existent KEK ID (should fail due to foreign key constraint)
	nonExistentKekID := uuid.Must(uuid.NewV7())
	dek.KekID = nonExistentKekID
	dek.EncryptedKey = []byte("updated-key")

	err = repo.Update(ctx, dek)
	assert.Error(t, err, "should fail due to foreign key constraint violation")
}

func TestMySQLDekRepository_Update_WithBinaryData(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create initial DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-key"),
		Nonce:        []byte("original-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Update with binary data including null bytes and special characters
	updatedKey := []byte{0xFF, 0xFE, 0xFD, 0x00, 0x01, 0x02, 0x80, 0x7F, 0xAA, 0xBB}
	updatedNonce := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC}

	dek.EncryptedKey = updatedKey
	dek.Nonce = updatedNonce

	err = repo.Update(ctx, dek)
	require.NoError(t, err)

	// Verify binary data is stored correctly
	var readDek cryptoDomain.Dek
	var dekIDBytes, kekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`

	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&dekIDBytes,
		&kekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)

	assert.Equal(t, updatedKey, readDek.EncryptedKey, "binary encrypted key should be preserved exactly")
	assert.Equal(t, updatedNonce, readDek.Nonce, "binary nonce should be preserved exactly")
}

func TestMySQLDekRepository_Update_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create initial DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-key"),
		Nonce:        []byte("original-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	// Marshal IDs
	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)
	kekIDBytes, err := dek.KekID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		dekID, kekIDBytes, dek.Algorithm, dek.EncryptedKey, dek.Nonce, dek.CreatedAt).Err()
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE deks 
			  SET kek_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  created_at = ?
			  WHERE id = ?`,
		kekIDBytes,
		dek.Algorithm,
		[]byte("updated-in-tx"),
		dek.Nonce,
		dek.CreatedAt,
		dekID,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the DEK was not updated (rollback worked)
	var readDek cryptoDomain.Dek
	var readDekIDBytes, readKekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`
	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&readDekIDBytes,
		&readKekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]byte("original-key"),
		readDek.EncryptedKey,
		"DEK should have original key after rollback",
	)
}

func TestMySQLDekRepository_Update_WithTransactionCommit(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create initial DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("original-key"),
		Nonce:        []byte("original-nonce"),
		CreatedAt:    time.Now().UTC(),
	}

	// Marshal IDs
	dekID, err := dek.ID.MarshalBinary()
	require.NoError(t, err)
	kekIDBytes, err := dek.KekID.MarshalBinary()
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		dekID, kekIDBytes, dek.Algorithm, dek.EncryptedKey, dek.Nonce, dek.CreatedAt).Err()
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE deks 
			  SET kek_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  created_at = ?
			  WHERE id = ?`,
		kekIDBytes,
		dek.Algorithm,
		[]byte("updated-in-tx"),
		dek.Nonce,
		dek.CreatedAt,
		dekID,
	)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify the DEK was updated (commit worked)
	var readDek cryptoDomain.Dek
	var readDekIDBytes, readKekIDBytes []byte
	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at FROM deks WHERE id = ?`
	err = db.QueryRowContext(ctx, query, dekID).Scan(
		&readDekIDBytes,
		&readKekIDBytes,
		&readDek.Algorithm,
		&readDek.EncryptedKey,
		&readDek.Nonce,
		&readDek.CreatedAt,
	)
	require.NoError(t, err)
	assert.Equal(t, []byte("updated-in-tx"), readDek.EncryptedKey, "DEK should have updated key after commit")
}

func TestMySQLDekRepository_Get(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create a DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("unique-nonce"),
		CreatedAt:    time.Now().UTC(),
	}
	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Get the DEK
	retrievedDek, err := repo.Get(ctx, dek.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedDek)

	// Verify all fields
	assert.Equal(t, dek.ID, retrievedDek.ID)
	assert.Equal(t, dek.KekID, retrievedDek.KekID)
	assert.Equal(t, dek.Algorithm, retrievedDek.Algorithm)
	assert.Equal(t, dek.EncryptedKey, retrievedDek.EncryptedKey)
	assert.Equal(t, dek.Nonce, retrievedDek.Nonce)
	assert.WithinDuration(t, dek.CreatedAt, retrievedDek.CreatedAt, time.Second)
}

func TestMySQLDekRepository_Get_NotFound(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Try to get a non-existent DEK
	nonExistentID := uuid.Must(uuid.NewV7())
	dek, err := repo.Get(ctx, nonExistentID)

	assert.Error(t, err)
	assert.Nil(t, dek)
	assert.ErrorIs(t, err, cryptoDomain.ErrDekNotFound)
}

func TestMySQLDekRepository_Get_WithDifferentAlgorithm(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK with ChaCha20
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.ChaCha20,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create a DEK with ChaCha20
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.ChaCha20,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("unique-nonce"),
		CreatedAt:    time.Now().UTC(),
	}
	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Get the DEK
	retrievedDek, err := repo.Get(ctx, dek.ID)
	require.NoError(t, err)
	assert.Equal(t, cryptoDomain.ChaCha20, retrievedDek.Algorithm)
}

func TestMySQLDekRepository_Get_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLDekRepository(db)
	ctx := context.Background()

	// Create a KEK
	kekID := uuid.Must(uuid.NewV7())
	kekRepo := NewMySQLKekRepository(db)
	kek := &cryptoDomain.Kek{
		ID:           kekID,
		MasterKeyID:  "master-key-1",
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-kek-data"),
		Nonce:        []byte("kek-nonce"),
		Version:      1,
		CreatedAt:    time.Now().UTC(),
	}
	err := kekRepo.Create(ctx, kek)
	require.NoError(t, err)

	// Create a DEK
	dek := &cryptoDomain.Dek{
		ID:           uuid.Must(uuid.NewV7()),
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("unique-nonce"),
		CreatedAt:    time.Now().UTC(),
	}
	err = repo.Create(ctx, dek)
	require.NoError(t, err)

	// Use TxManager to get the DEK within a transaction
	txManager := database.NewTxManager(db)
	var retrievedDek *cryptoDomain.Dek

	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
		var txErr error
		retrievedDek, txErr = repo.Get(txCtx, dek.ID)
		return txErr
	})

	require.NoError(t, err)
	assert.NotNil(t, retrievedDek)
	assert.Equal(t, dek.ID, retrievedDek.ID)
}
