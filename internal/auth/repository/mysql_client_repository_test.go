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

func TestNewMySQLClientRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLClientRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLClientRepository{}, repo)
}

func TestMySQLClientRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret-hash",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Verify the client was created by retrieving it
	retrievedClient, err := repo.Get(ctx, client.ID)
	require.NoError(t, err)

	assert.Equal(t, client.ID, retrievedClient.ID)
	assert.Equal(t, client.Secret, retrievedClient.Secret)
	assert.Equal(t, client.Name, retrievedClient.Name)
	assert.Equal(t, client.IsActive, retrievedClient.IsActive)
	assert.WithinDuration(t, client.CreatedAt, retrievedClient.CreatedAt, time.Second)
}

func TestMySQLClientRepository_Create_InactiveClient(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "inactive-secret",
		Name:      "inactive-client",
		IsActive:  false,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Verify the client was created with correct is_active status
	retrievedClient, err := repo.Get(ctx, client.ID)
	require.NoError(t, err)
	assert.False(t, retrievedClient.IsActive)
}

func TestMySQLClientRepository_Create_MultipleClients(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Create first client
	client1 := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "secret-1",
		Name:      "client-1",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client1)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create second client
	client2 := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "secret-2",
		Name:      "client-2",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err = repo.Create(ctx, client2)
	require.NoError(t, err)

	// Verify both clients can be retrieved
	retrievedClient1, err := repo.Get(ctx, client1.ID)
	require.NoError(t, err)
	assert.Equal(t, client1.ID, retrievedClient1.ID)

	retrievedClient2, err := repo.Get(ctx, client2.ID)
	require.NoError(t, err)
	assert.Equal(t, client2.ID, retrievedClient2.ID)
}

func TestMySQLClientRepository_Update(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Create initial client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "original-secret",
		Name:      "original-name",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Update the client
	client.Secret = "updated-secret"
	client.Name = "updated-name"
	client.IsActive = false

	err = repo.Update(ctx, client)
	require.NoError(t, err)

	// Verify the update
	retrievedClient, err := repo.Get(ctx, client.ID)
	require.NoError(t, err)

	assert.Equal(t, client.ID, retrievedClient.ID)
	assert.Equal(t, "updated-secret", retrievedClient.Secret)
	assert.Equal(t, "updated-name", retrievedClient.Name)
	assert.False(t, retrievedClient.IsActive)
}

func TestMySQLClientRepository_Update_NonExistent(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Try to update a non-existent client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "secret",
		Name:      "name",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	// Update should not return an error even if no rows are affected
	err := repo.Update(ctx, client)
	assert.NoError(t, err)
}

func TestMySQLClientRepository_Get_Success(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Create a client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Retrieve the client
	retrievedClient, err := repo.Get(ctx, client.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedClient)

	assert.Equal(t, client.ID, retrievedClient.ID)
	assert.Equal(t, client.Secret, retrievedClient.Secret)
	assert.Equal(t, client.Name, retrievedClient.Name)
	assert.Equal(t, client.IsActive, retrievedClient.IsActive)
	assert.WithinDuration(t, client.CreatedAt, retrievedClient.CreatedAt, time.Second)
}

func TestMySQLClientRepository_Get_NotFound(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Try to get a non-existent client
	nonExistentID := uuid.Must(uuid.NewV7())
	client, err := repo.Get(ctx, nonExistentID)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, authDomain.ErrClientNotFound)
}

func TestMySQLClientRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "secret",
		Name:      "client-name-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	// Test rollback behavior using a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Marshal client ID
	id, err := client.ID.MarshalBinary()
	require.NoError(t, err)

	// Create client within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO clients (id, secret, name, is_active, created_at) VALUES (?, ?, ?, ?, ?)`,
		id,
		client.Secret,
		client.Name,
		client.IsActive,
		client.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the client was not created (rollback worked)
	retrievedClient, err := repo.Get(ctx, client.ID)
	assert.Error(t, err)
	assert.Nil(t, retrievedClient)
	assert.ErrorIs(t, err, authDomain.ErrClientNotFound)
}

func TestMySQLClientRepository_Update_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Create initial client
	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "original-secret",
		Name:      "original-name-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Marshal client ID
	id, err := client.ID.MarshalBinary()
	require.NoError(t, err)

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE clients SET secret = ?, name = ?, is_active = ?, created_at = ? WHERE id = ?`,
		"updated-secret",
		"updated-name",
		false,
		client.CreatedAt,
		id,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the client was not updated (rollback worked)
	retrievedClient, err := repo.Get(ctx, client.ID)
	require.NoError(t, err)
	assert.Equal(t, "original-secret", retrievedClient.Secret)
	assert.Equal(t, "original-name-tx", retrievedClient.Name)
	assert.True(t, retrievedClient.IsActive)
}

func TestMySQLClientRepository_Get_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLClientRepository(db)
	ctx := context.Background()

	// Create a client outside transaction
	client1 := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "secret-1",
		Name:      "client-1-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	err := repo.Create(ctx, client1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create another client inside transaction
	time.Sleep(time.Millisecond)
	client2 := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "secret-2",
		Name:      "client-2-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	// Marshal client ID
	id2, err := client2.ID.MarshalBinary()
	require.NoError(t, err)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO clients (id, secret, name, is_active, created_at) VALUES (?, ?, ?, ?, ?)`,
		id2,
		client2.Secret,
		client2.Name,
		client2.IsActive,
		client2.CreatedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see both clients
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Get outside transaction should also see both clients
	retrievedClient1, err := repo.Get(ctx, client1.ID)
	require.NoError(t, err)
	assert.Equal(t, client1.ID, retrievedClient1.ID)

	retrievedClient2, err := repo.Get(ctx, client2.ID)
	require.NoError(t, err)
	assert.Equal(t, client2.ID, retrievedClient2.ID)
}
