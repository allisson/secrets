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

func TestNewPostgreSQLTokenRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLTokenRepository{}, repo)
}

func TestPostgreSQLTokenRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Now create a token
	tokenRepo := NewPostgreSQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "test-token-hash-1",
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

func TestPostgreSQLTokenRepository_Create_WithRevokedAt(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-revoked",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a revoked token
	tokenRepo := NewPostgreSQLTokenRepository(db)
	revokedAt := time.Now().UTC()

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "revoked-token-hash",
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

func TestPostgreSQLTokenRepository_Create_MultipleTokens(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-multiple",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewPostgreSQLTokenRepository(db)

	// Create first token
	token1 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-hash-1",
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
		TokenHash: "token-hash-2",
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

func TestPostgreSQLTokenRepository_Update(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-update",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create initial token
	tokenRepo := NewPostgreSQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "original-hash",
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
	token.TokenHash = "updated-hash"

	err = tokenRepo.Update(ctx, token)
	require.NoError(t, err)

	// Verify the update
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)

	assert.Equal(t, token.ID, retrievedToken.ID)
	assert.Equal(t, "updated-hash", retrievedToken.TokenHash)
	require.NotNil(t, retrievedToken.RevokedAt)
	assert.WithinDuration(t, revokedAt, *retrievedToken.RevokedAt, time.Second)
}

func TestPostgreSQLTokenRepository_Update_NonExistent(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-nonexist",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewPostgreSQLTokenRepository(db)

	// Try to update a non-existent token
	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "hash",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Update should not return an error even if no rows are affected
	err = tokenRepo.Update(ctx, token)
	assert.NoError(t, err)
}

func TestPostgreSQLTokenRepository_Get_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a token
	tokenRepo := NewPostgreSQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "get-test-hash",
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

func TestPostgreSQLTokenRepository_Get_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	tokenRepo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	// Try to get a non-existent token
	nonExistentID := uuid.Must(uuid.NewV7())
	token, err := tokenRepo.Get(ctx, nonExistentID)

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
}

func TestPostgreSQLTokenRepository_Create_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewPostgreSQLTokenRepository(db)

	//nolint:gosec // test fixture data
	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "tx-test-hash",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	// Test rollback behavior using a transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Create token within transaction
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID,
		token.TokenHash,
		token.ClientID,
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

func TestPostgreSQLTokenRepository_Update_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-update-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create initial token
	tokenRepo := NewPostgreSQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "original-hash-tx",
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

	// Update within transaction
	_, err = tx.ExecContext(
		ctx,
		`UPDATE tokens SET token_hash = $1, client_id = $2, expires_at = $3, revoked_at = $4, 
		 created_at = $5 WHERE id = $6`,
		"updated-hash-tx",
		token.ClientID,
		token.ExpiresAt,
		revokedAt,
		token.CreatedAt,
		token.ID,
	)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify the token was not updated (rollback worked)
	retrievedToken, err := tokenRepo.Get(ctx, token.ID)
	require.NoError(t, err)
	assert.Equal(t, "original-hash-tx", retrievedToken.TokenHash)
	assert.Nil(t, retrievedToken.RevokedAt)
}

func TestPostgreSQLTokenRepository_Get_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewPostgreSQLTokenRepository(db)

	// Create a token outside transaction
	token1 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-1-tx",
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
		TokenHash: "token-2-tx",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token2.ID,
		token2.TokenHash,
		token2.ClientID,
		token2.ExpiresAt,
		token2.RevokedAt,
		token2.CreatedAt,
	)
	require.NoError(t, err)

	// Query within transaction should see both tokens
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tokens WHERE client_id = $1`, client.ID).Scan(&count)
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

func TestPostgreSQLTokenRepository_GetByTokenHash_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-get-by-hash",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a token
	tokenRepo := NewPostgreSQLTokenRepository(db)

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "unique-token-hash-123",
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

func TestPostgreSQLTokenRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	tokenRepo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	// Try to get a token with a non-existent hash
	token, err := tokenRepo.GetByTokenHash(ctx, "non-existent-hash")

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.ErrorIs(t, err, authDomain.ErrTokenNotFound)
}

func TestPostgreSQLTokenRepository_GetByTokenHash_WithRevokedToken(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-revoked-hash",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	// Create a revoked token
	tokenRepo := NewPostgreSQLTokenRepository(db)
	revokedAt := time.Now().UTC()

	token := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "revoked-token-hash-456",
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

func TestPostgreSQLTokenRepository_GetByTokenHash_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	// First create a client for the foreign key
	clientRepo := NewPostgreSQLClientRepository(db)
	ctx := context.Background()

	client := &authDomain.Client{
		ID:        uuid.Must(uuid.NewV7()),
		Secret:    "test-secret",
		Name:      "test-client-hash-tx",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	err := clientRepo.Create(ctx, client)
	require.NoError(t, err)

	tokenRepo := NewPostgreSQLTokenRepository(db)

	// Create a token outside transaction
	//nolint:gosec // test fixture data
	token1 := &authDomain.Token{
		ID:        uuid.Must(uuid.NewV7()),
		TokenHash: "token-hash-tx-1",
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
		TokenHash: "token-hash-tx-2",
		ClientID:  client.ID,
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO tokens (id, token_hash, client_id, expires_at, revoked_at, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token2.ID,
		token2.TokenHash,
		token2.ClientID,
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
