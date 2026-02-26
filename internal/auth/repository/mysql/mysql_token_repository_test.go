package mysql

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

func TestNewMySQLTokenRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLTokenRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLTokenRepository{}, repo)
}

func TestMySQLTokenRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Now create a token
	tokenRepo := NewMySQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "test-token-hash-mysql-1",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Verify the token was created by retrieving it
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)

	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Equal(t, token.TokenHash, retrievedToken.TokenHash)
	assert.Equal(t, token.ClientID, retrievedToken.ClientID)
	assert.WithinDuration(t, token.ExpiresAt, retrievedToken.ExpiresAt, time.Second)
	assert.Nil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, token.CreatedAt, retrievedToken.CreatedAt, time.Second)
}

func TestMySQLTokenRepository_Create_WithRevokedAt(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-revoked-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a revoked token
	tokenRepo := NewMySQLTokenRepository(db)
	revokedAt := time.Now().UTC()

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "revoked-token-hash-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: &revokedAt,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Verify the token was created with revoked_at set
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, revokedAt, *retrievedToken.RevokedAt, time.Second)
}

func TestMySQLTokenRepository_Create_MultipleTokens(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-multiple-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewMySQLTokenRepository(db)

	// Create first token
	//nolint:gosec // test fixture data
	token1 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-hash-mysql-1",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token1)
	require.NoError(t, err)

	time.Sleep(time.Millisecond) // Ensure different timestamp for UUIDv7

	// Create second token
	//nolint:gosec // test fixture data
	token2 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-hash-mysql-2",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token2)
	require.NoError(t, err)

	// Verify both tokens can be retrieved
	retrievedToken1, err := tokenRepo.Get(ctx, token1.ID)
	require.NoError(t, err)
	assert.Equal(t, token1.ID, retrievedToken1.ID)

	retrievedToken2, err := tokenRepo.Get(ctx, token2.ID)
	require.NoError(t, err)
	assert.Equal(t, token2.ID, retrievedToken2.ID)
}

func TestMySQLTokenRepository_Update(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-update-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create initial token
	tokenRepo := NewMySQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "original-hash-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Update the token (revoke it)
	revokedAt := time.Now().UTC()
	token.RevokedAt = &revokedAt
	token.TokenHash = "updated-hash-mysql"

	err = tokenRepo.Update(ctx, token)
	require.NoError(t, err)

	// Verify the update
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)

	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Equal(t, "updated-hash-mysql", retrievedToken.TokenHash)
	require.NotNil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, revokedAt, *retrievedToken.RevokedAt, time.Second)
}

func TestMySQLTokenRepository_Update_NonExistent(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-nonexist-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewMySQLTokenRepository(db)

	// Try to update a non-existent token
	//nolint:gosec // test fixture data
	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "hash-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Update should not return an error even if no rows are affected
	err = tokenRepo.Update(ctx, token)
	assert.NoError(t, err)
}

func TestMySQLTokenRepository_Get_Success(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a token
	tokenRepo := NewMySQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "get-test-hash-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Retrieve the token
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedToken)

	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Equal(t, token.TokenHash, retrievedToken.TokenHash)
	assert.Equal(t, token.ClientID, retrievedToken.ClientID)
	assert.WithinDuration(t, token.ExpiresAt, retrievedToken.ExpiresAt, time.Second)
	assert.Nil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, token.CreatedAt, retrievedToken.CreatedAt, time.Second)
}

func TestMySQLTokenRepository_Get_NotFound(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	tokenRepo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	// Try to get a non-existent token
	nonExistentID := uuid.Must(uuid.NewV7())
	token, err := tokenRepo.Get(ctx, nonExistentID)

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
}

func TestMySQLTokenRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-tx-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewMySQLTokenRepository(db)

	//nolint:gosec // test fixture data
	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "tx-test-hash-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Test rollback behavior using a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Marshal UUIDs
	id, err := token.ID.MarshalBinary()
	require.NoError(t, err)

	clientID, err := token.ClientID.MarshalBinary()
	require.NoError(t, err)

	// Create token within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id,
		token.TokenHash,
		clientID,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the token was not created (rollback worked)
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	assert.Error(t, err)
	assert.Nil(t, retrievedToken)
	assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
}

func TestMySQLTokenRepository_Update_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-update-tx-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create initial token
	tokenRepo := NewMySQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "original-hash-tx-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	revokedAt := time.Now().UTC()

	// Marshal UUIDs
	id, err := token.ID.MarshalBinary()
	require.NoError(t, err)

	clientID, err := token.ClientID.MarshalBinary()
	require.NoError(t, err)

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE tokens SET token_hash = ?, client_id = ?, expires_at = ?, revoked_at = ?, 
		 created_at = ? WHERE id = ?`,
		"updated-hash-tx-mysql",
		clientID,
		token.ExpiresAt,
		revokedAt,
		token.CreatedAt,
		id,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the token was not updated (rollback worked)
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)
	assert.Equal(t, "original-hash-tx-mysql", retrievedToken.TokenHash)
	assert.Nil(t, retrievedToken.RevokedAt)
}

func TestMySQLTokenRepository_Get_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get-tx-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewMySQLTokenRepository(db)

	// Create a token outside transaction
	//nolint:gosec // test fixture data
	token1 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-1-tx-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create another token inside transaction
	time.Sleep(time.Millisecond)
	//nolint:gosec // test fixture data
	token2 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-2-tx-mysql",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Marshal UUIDs
	id2, err := token2.ID.MarshalBinary()
	require.NoError(t, err)

	clientID, err := token2.ClientID.MarshalBinary()
	require.NoError(t, err)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id2,
		token2.TokenHash,
		clientID,
		token2.ExpiresAt,
		token2.RevokedAt,
		token2.CreatedAt,
	)
	require.NoError(t, err)

	// Marshal client ID for query
	clientIDQuery, err := client.ID.MarshalBinary()
	require.NoError(t, err)

	// Query within transaction should see both tokens
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tokens WHERE client_id = ?`, clientIDQuery).
		Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Get outside transaction should also see both tokens
	retrievedToken1, err := tokenRepo.Get(ctx, token1.ID)
	require.NoError(t, err)
	assert.Equal(t, token1.ID, retrievedToken1.ID)

	retrievedToken2, err := tokenRepo.Get(ctx, token2.ID)
	require.NoError(t, err)
	assert.Equal(t, token2.ID, retrievedToken2.ID)
}

func TestMySQLTokenRepository_GetByTokenHash_Success(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get-by-hash-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a token
	tokenRepo := NewMySQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "unique-token-hash-mysql-123",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Retrieve the token by hash
	retrievedToken, err := tokenRepo.GetByTokenHash(ctx, token.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, retrievedToken)

	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Equal(t, token.TokenHash, retrievedToken.TokenHash)
	assert.Equal(t, token.ClientID, retrievedToken.ClientID)
	assert.WithinDuration(t, token.ExpiresAt, retrievedToken.ExpiresAt, time.Second)
	assert.Nil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, token.CreatedAt, retrievedToken.CreatedAt, time.Second)
}

func TestMySQLTokenRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	tokenRepo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	// Try to get a token with a non-existent hash
	token, err := tokenRepo.GetByTokenHash(ctx, "non-existent-hash-mysql")

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
}

func TestMySQLTokenRepository_GetByTokenHash_WithRevokedToken(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-revoked-hash-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a revoked token
	tokenRepo := NewMySQLTokenRepository(db)
	revokedAt := time.Now().UTC()

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "revoked-token-hash-mysql-456",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: &revokedAt,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Retrieve the revoked token by hash
	retrievedToken, err := tokenRepo.GetByTokenHash(ctx, token.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, retrievedToken)

	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Equal(t, token.TokenHash, retrievedToken.TokenHash)
	assert.Equal(t, token.ClientID, retrievedToken.ClientID)
	require.NotNil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, revokedAt, *retrievedToken.RevokedAt, time.Second)
}

func TestMySQLTokenRepository_GetByTokenHash_WithTransaction(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewMySQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-hash-tx-mysql",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewMySQLTokenRepository(db)

	// Create a token outside transaction
	//nolint:gosec // test fixture data
	token1 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-hash-tx-mysql-1",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	err = tokenRepo.Create(ctx, token1)
	require.NoError(t, err)

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create another token inside transaction
	time.Sleep(time.Millisecond)
	//nolint:gosec // test fixture data
	token2 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-hash-tx-mysql-2",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Marshal UUIDs
	id2, err := token2.ID.MarshalBinary()
	require.NoError(t, err)

	clientID, err := token2.ClientID.MarshalBinary()
	require.NoError(t, err)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id2,
		token2.TokenHash,
		clientID,
		token2.ExpiresAt,
		token2.RevokedAt,
		token2.CreatedAt,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Token1 should be retrievable by hash
	retrievedToken1, err := tokenRepo.GetByTokenHash(ctx, token1.TokenHash)
	require.NoError(t, err)
	assert.Equal(t, token1.ID, retrievedToken1.ID)
	assert.Equal(t, token1.TokenHash, retrievedToken1.TokenHash)

	// Token2 should not be retrievable (transaction was rolled back)
	retrievedToken2, err := tokenRepo.GetByTokenHash(ctx, token2.TokenHash)
	assert.Error(t, err)
	assert.Nil(t, retrievedToken2)
	assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
}
