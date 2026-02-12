package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/testutil"
)

func TestNewMySQLAuditLogRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLAuditLogRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLAuditLogRepository{}, repo)
}

func TestMySQLAuditLogRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLAuditLogRepository(db)
	ctx := context.Background()

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test-key",
		Metadata: map[string]any{
			"method": "GET",
			"status": 200,
		},
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, auditLog)
	require.NoError(t, err)

	// Verify the audit log was created by querying directly
	id, err := auditLog.ID.MarshalBinary()
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE id = ?`, id).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMySQLAuditLogRepository_Create_WithNilMetadata(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLAuditLogRepository(db)
	ctx := context.Background()

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/another-key",
		Metadata:   nil, // Nil metadata should be stored as NULL
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, auditLog)
	require.NoError(t, err)

	// Verify metadata is NULL in database
	id, err := auditLog.ID.MarshalBinary()
	require.NoError(t, err)

	var metadataNull bool
	err = db.QueryRowContext(
		ctx,
		`SELECT metadata IS NULL FROM audit_logs WHERE id = ?`,
		id,
	).Scan(&metadataNull)
	require.NoError(t, err)
	assert.True(t, metadataNull, "metadata should be NULL in database")
}

func TestMySQLAuditLogRepository_Create_WithEmptyMetadata(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLAuditLogRepository(db)
	ctx := context.Background()

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/empty-metadata",
		Metadata:   map[string]any{}, // Empty map should be stored as {}
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, auditLog)
	require.NoError(t, err)

	// Verify metadata is {} (not NULL) in database
	id, err := auditLog.ID.MarshalBinary()
	require.NoError(t, err)

	var metadataJSON string
	err = db.QueryRowContext(
		ctx,
		`SELECT CAST(metadata AS CHAR) FROM audit_logs WHERE id = ?`,
		id,
	).Scan(&metadataJSON)
	require.NoError(t, err)
	assert.Equal(t, "{}", metadataJSON)
}

func TestMySQLAuditLogRepository_Create_MultipleAuditLogs(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLAuditLogRepository(db)
	ctx := context.Background()

	// Create first audit log
	auditLog1 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.EncryptCapability,
		Path:       "/transit/encrypt/key1",
		Metadata:   map[string]any{"plaintext_length": 256},
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, auditLog1)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create second audit log
	auditLog2 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.DecryptCapability,
		Path:       "/transit/decrypt/key2",
		Metadata:   map[string]any{"ciphertext_length": 512},
		CreatedAt:  time.Now().UTC(),
	}

	err = repo.Create(ctx, auditLog2)
	require.NoError(t, err)

	// Verify both audit logs exist
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMySQLAuditLogRepository_Create_AllCapabilities(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLAuditLogRepository(db)
	ctx := context.Background()

	capabilities := []authDomain.Capability{
		authDomain.ReadCapability,
		authDomain.WriteCapability,
		authDomain.DeleteCapability,
		authDomain.EncryptCapability,
		authDomain.DecryptCapability,
		authDomain.RotateCapability,
	}

	for _, capability := range capabilities {
		time.Sleep(time.Millisecond) // Ensure different UUIDv7
		auditLog := &authDomain.AuditLog{
			ID:         uuid.Must(uuid.NewV7()),
			RequestID:  uuid.Must(uuid.NewV7()),
			ClientID:   uuid.Must(uuid.NewV7()),
			Capability: capability,
			Path:       "/test/path",
			CreatedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, auditLog)
		require.NoError(t, err)

		// Verify capability was stored correctly
		id, err := auditLog.ID.MarshalBinary()
		require.NoError(t, err)

		var storedCapability string
		err = db.QueryRowContext(
			ctx,
			`SELECT capability FROM audit_logs WHERE id = ?`,
			id,
		).Scan(&storedCapability)
		require.NoError(t, err)
		assert.Equal(t, string(capability), storedCapability)
	}
}

func TestMySQLAuditLogRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/tx-test",
		Metadata:   map[string]any{"transaction": "commit"},
		CreatedAt:  time.Now().UTC(),
	}

	id, err := auditLog.ID.MarshalBinary()
	require.NoError(t, err)

	requestID, err := auditLog.RequestID.MarshalBinary()
	require.NoError(t, err)

	clientID, err := auditLog.ClientID.MarshalBinary()
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create audit log within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO audit_logs (id, request_id, client_id, capability, path, metadata, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		requestID,
		clientID,
		string(auditLog.Capability),
		auditLog.Path,
		`{"transaction": "commit"}`,
		auditLog.CreatedAt,
	)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Verify the audit log exists after commit
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE id = ?`, id).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMySQLAuditLogRepository_Create_TransactionRollback(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	ctx := context.Background()

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/rollback-test",
		Metadata:   map[string]any{"transaction": "rollback"},
		CreatedAt:  time.Now().UTC(),
	}

	id, err := auditLog.ID.MarshalBinary()
	require.NoError(t, err)

	requestID, err := auditLog.RequestID.MarshalBinary()
	require.NoError(t, err)

	clientID, err := auditLog.ClientID.MarshalBinary()
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create audit log within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO audit_logs (id, request_id, client_id, capability, path, metadata, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		requestID,
		clientID,
		string(auditLog.Capability),
		auditLog.Path,
		`{"transaction": "rollback"}`,
		auditLog.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the audit log does not exist after rollback
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE id = ?`, id).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
