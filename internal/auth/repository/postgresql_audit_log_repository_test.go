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

func TestNewPostgreSQLAuditLogRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLAuditLogRepository{}, repo)
}

func TestPostgreSQLAuditLogRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test client to satisfy FK constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-create")

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID,
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
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE id = $1`, auditLog.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPostgreSQLAuditLogRepository_Create_WithNilMetadata(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test client to satisfy FK constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-nil-metadata")

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID,
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/another-key",
		Metadata:   nil, // Nil metadata should be stored as NULL
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, auditLog)
	require.NoError(t, err)

	// Verify metadata is NULL in database
	var metadataNull bool
	err = db.QueryRowContext(
		ctx,
		`SELECT metadata IS NULL FROM audit_logs WHERE id = $1`,
		auditLog.ID,
	).Scan(&metadataNull)
	require.NoError(t, err)
	assert.True(t, metadataNull, "metadata should be NULL in database")
}

func TestPostgreSQLAuditLogRepository_Create_WithEmptyMetadata(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test client to satisfy FK constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-empty-metadata")

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID,
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/empty-metadata",
		Metadata:   map[string]any{}, // Empty map should be stored as {}
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Create(ctx, auditLog)
	require.NoError(t, err)

	// Verify metadata is {} (not NULL) in database
	var metadataJSON string
	err = db.QueryRowContext(
		ctx,
		`SELECT metadata::text FROM audit_logs WHERE id = $1`,
		auditLog.ID,
	).Scan(&metadataJSON)
	require.NoError(t, err)
	assert.Equal(t, "{}", metadataJSON)
}

func TestPostgreSQLAuditLogRepository_Create_MultipleAuditLogs(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test clients to satisfy FK constraint
	clientID1 := testutil.CreateTestClient(t, db, "postgres", "test-multiple-1")
	clientID2 := testutil.CreateTestClient(t, db, "postgres", "test-multiple-2")

	// Create first audit log
	auditLog1 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID1,
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
		ClientID:   clientID2,
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

func TestPostgreSQLAuditLogRepository_Create_AllCapabilities(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test client to satisfy FK constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-capabilities")

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
			ClientID:   clientID,
			Capability: capability,
			Path:       "/test/path",
			CreatedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, auditLog)
		require.NoError(t, err)

		// Verify capability was stored correctly
		var storedCapability string
		err = db.QueryRowContext(
			ctx,
			`SELECT capability FROM audit_logs WHERE id = $1`,
			auditLog.ID,
		).Scan(&storedCapability)
		require.NoError(t, err)
		assert.Equal(t, string(capability), storedCapability)
	}
}

func TestPostgreSQLAuditLogRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	ctx := context.Background()

	// Create test client to satisfy FK constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-tx")

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID,
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/tx-test",
		Metadata:   map[string]any{"transaction": "commit"},
		CreatedAt:  time.Now().UTC(),
	}

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create audit log within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO audit_logs (id, request_id, client_id, capability, path, metadata, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		auditLog.ID,
		auditLog.RequestID,
		auditLog.ClientID,
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
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE id = $1`, auditLog.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPostgreSQLAuditLogRepository_Create_TransactionRollback(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	ctx := context.Background()

	// Create test client to satisfy FK constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-rollback")

	auditLog := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID,
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/rollback-test",
		Metadata:   map[string]any{"transaction": "rollback"},
		CreatedAt:  time.Now().UTC(),
	}

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create audit log within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO audit_logs (id, request_id, client_id, capability, path, metadata, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		auditLog.ID,
		auditLog.RequestID,
		auditLog.ClientID,
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
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE id = $1`, auditLog.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestPostgreSQLAuditLogRepository_List_SortingByCreatedAt(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test clients for foreign key constraints
	clientID1 := testutil.CreateTestClient(t, db, "postgres", "test-sort-1")
	clientID2 := testutil.CreateTestClient(t, db, "postgres", "test-sort-2")
	clientID3 := testutil.CreateTestClient(t, db, "postgres", "test-sort-3")

	// Create audit logs with different created_at timestamps
	now := time.Now().UTC()
	auditLog1 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID1,
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/oldest",
		Metadata:   nil,
		CreatedAt:  now.Add(-2 * time.Hour),
	}
	auditLog2 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID2,
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/middle",
		Metadata:   nil,
		CreatedAt:  now.Add(-1 * time.Hour),
	}
	auditLog3 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID3,
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/newest",
		Metadata:   nil,
		CreatedAt:  now,
	}

	require.NoError(t, repo.Create(ctx, auditLog1))
	require.NoError(t, repo.Create(ctx, auditLog2))
	require.NoError(t, repo.Create(ctx, auditLog3))

	// List all audit logs without filters
	auditLogs, err := repo.List(ctx, 0, 10, nil, nil)
	require.NoError(t, err)
	require.Len(t, auditLogs, 3)

	// Verify they are sorted by created_at DESC (newest first)
	assert.Equal(t, auditLog3.ID, auditLogs[0].ID)
	assert.Equal(t, auditLog2.ID, auditLogs[1].ID)
	assert.Equal(t, auditLog1.ID, auditLogs[2].ID)
}

func TestPostgreSQLAuditLogRepository_List_WithCreatedAtFromFilter(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test clients for foreign key constraints
	clientID1 := testutil.CreateTestClient(t, db, "postgres", "test-from-1")
	clientID2 := testutil.CreateTestClient(t, db, "postgres", "test-from-2")
	clientID3 := testutil.CreateTestClient(t, db, "postgres", "test-from-3")

	// Create audit logs with different timestamps
	now := time.Now().UTC()
	auditLog1 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID1,
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/before",
		Metadata:   nil,
		CreatedAt:  now.Add(-3 * time.Hour),
	}
	auditLog2 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID2,
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/after1",
		Metadata:   nil,
		CreatedAt:  now.Add(-1 * time.Hour),
	}
	auditLog3 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID3,
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/after2",
		Metadata:   nil,
		CreatedAt:  now,
	}

	require.NoError(t, repo.Create(ctx, auditLog1))
	require.NoError(t, repo.Create(ctx, auditLog2))
	require.NoError(t, repo.Create(ctx, auditLog3))

	// Filter with created_at_from = now - 2 hours (should return 2 logs)
	createdAtFrom := now.Add(-2 * time.Hour)
	auditLogs, err := repo.List(ctx, 0, 10, &createdAtFrom, nil)
	require.NoError(t, err)
	require.Len(t, auditLogs, 2)

	// Verify only logs after the filter time are returned (newest first)
	assert.Equal(t, auditLog3.ID, auditLogs[0].ID)
	assert.Equal(t, auditLog2.ID, auditLogs[1].ID)
}

func TestPostgreSQLAuditLogRepository_List_WithCreatedAtToFilter(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test clients for foreign key constraints
	clientID1 := testutil.CreateTestClient(t, db, "postgres", "test-to-1")
	clientID2 := testutil.CreateTestClient(t, db, "postgres", "test-to-2")
	clientID3 := testutil.CreateTestClient(t, db, "postgres", "test-to-3")

	// Create audit logs with different timestamps
	now := time.Now().UTC()
	auditLog1 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID1,
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/before1",
		Metadata:   nil,
		CreatedAt:  now.Add(-3 * time.Hour),
	}
	auditLog2 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID2,
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/before2",
		Metadata:   nil,
		CreatedAt:  now.Add(-1 * time.Hour),
	}
	auditLog3 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID3,
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/after",
		Metadata:   nil,
		CreatedAt:  now,
	}

	require.NoError(t, repo.Create(ctx, auditLog1))
	require.NoError(t, repo.Create(ctx, auditLog2))
	require.NoError(t, repo.Create(ctx, auditLog3))

	// Filter with created_at_to = now - 30 minutes (should return 2 logs)
	createdAtTo := now.Add(-30 * time.Minute)
	auditLogs, err := repo.List(ctx, 0, 10, nil, &createdAtTo)
	require.NoError(t, err)
	require.Len(t, auditLogs, 2)

	// Verify only logs before the filter time are returned (newest first)
	assert.Equal(t, auditLog2.ID, auditLogs[0].ID)
	assert.Equal(t, auditLog1.ID, auditLogs[1].ID)
}

func TestPostgreSQLAuditLogRepository_List_WithBothFilters(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test clients for foreign key constraints
	clientID1 := testutil.CreateTestClient(t, db, "postgres", "test-both-1")
	clientID2 := testutil.CreateTestClient(t, db, "postgres", "test-both-2")
	clientID3 := testutil.CreateTestClient(t, db, "postgres", "test-both-3")
	clientID4 := testutil.CreateTestClient(t, db, "postgres", "test-both-4")

	// Create audit logs with different timestamps
	now := time.Now().UTC()
	auditLog1 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID1,
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/before-range",
		Metadata:   nil,
		CreatedAt:  now.Add(-5 * time.Hour),
	}
	auditLog2 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID2,
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/in-range1",
		Metadata:   nil,
		CreatedAt:  now.Add(-3 * time.Hour),
	}
	auditLog3 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID3,
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/in-range2",
		Metadata:   nil,
		CreatedAt:  now.Add(-2 * time.Hour),
	}
	auditLog4 := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   clientID4,
		Capability: authDomain.EncryptCapability,
		Path:       "/secrets/after-range",
		Metadata:   nil,
		CreatedAt:  now,
	}

	require.NoError(t, repo.Create(ctx, auditLog1))
	require.NoError(t, repo.Create(ctx, auditLog2))
	require.NoError(t, repo.Create(ctx, auditLog3))
	require.NoError(t, repo.Create(ctx, auditLog4))

	// Filter with date range: now - 4 hours to now - 1 hour (should return 2 logs)
	createdAtFrom := now.Add(-4 * time.Hour)
	createdAtTo := now.Add(-1 * time.Hour)
	auditLogs, err := repo.List(ctx, 0, 10, &createdAtFrom, &createdAtTo)
	require.NoError(t, err)
	require.Len(t, auditLogs, 2)

	// Verify only logs within the range are returned (newest first)
	assert.Equal(t, auditLog3.ID, auditLogs[0].ID)
	assert.Equal(t, auditLog2.ID, auditLogs[1].ID)
}

func TestPostgreSQLAuditLogRepository_List_NoFilters(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test client for foreign key constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-no-filters")

	// Create multiple audit logs
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		auditLog := &authDomain.AuditLog{
			ID:         uuid.Must(uuid.NewV7()),
			RequestID:  uuid.Must(uuid.NewV7()),
			ClientID:   clientID,
			Capability: authDomain.ReadCapability,
			Path:       "/secrets/test",
			Metadata:   nil,
			CreatedAt:  now.Add(time.Duration(-i) * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, auditLog))
	}

	// List all without filters
	auditLogs, err := repo.List(ctx, 0, 10, nil, nil)
	require.NoError(t, err)
	assert.Len(t, auditLogs, 5)

	// Verify sorting (newest first)
	for i := 0; i < len(auditLogs)-1; i++ {
		assert.True(t, auditLogs[i].CreatedAt.After(auditLogs[i+1].CreatedAt) ||
			auditLogs[i].CreatedAt.Equal(auditLogs[i+1].CreatedAt))
	}
}

func TestPostgreSQLAuditLogRepository_List_EmptyResult(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// List when no audit logs exist
	auditLogs, err := repo.List(ctx, 0, 10, nil, nil)
	require.NoError(t, err)
	assert.Len(t, auditLogs, 0)
	assert.NotNil(t, auditLogs) // Should return empty slice, not nil
}

func TestPostgreSQLAuditLogRepository_List_Pagination(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLAuditLogRepository(db)
	ctx := context.Background()

	// Create test client for foreign key constraint
	clientID := testutil.CreateTestClient(t, db, "postgres", "test-pagination")

	// Create 10 audit logs
	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		auditLog := &authDomain.AuditLog{
			ID:         uuid.Must(uuid.NewV7()),
			RequestID:  uuid.Must(uuid.NewV7()),
			ClientID:   clientID,
			Capability: authDomain.ReadCapability,
			Path:       "/secrets/test",
			Metadata:   nil,
			CreatedAt:  now.Add(time.Duration(-i) * time.Minute),
		}
		require.NoError(t, repo.Create(ctx, auditLog))
	}

	// First page (limit 3, offset 0)
	page1, err := repo.List(ctx, 0, 3, nil, nil)
	require.NoError(t, err)
	assert.Len(t, page1, 3)

	// Second page (limit 3, offset 3)
	page2, err := repo.List(ctx, 3, 3, nil, nil)
	require.NoError(t, err)
	assert.Len(t, page2, 3)

	// Verify pages don't overlap
	assert.NotEqual(t, page1[0].ID, page2[0].ID)

	// Verify page2 has older logs than page1
	assert.True(t, page2[0].CreatedAt.Before(page1[len(page1)-1].CreatedAt))
}
