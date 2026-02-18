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
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

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
	dekID = uuid.Must(uuid.NewV7())
	dekRepo := cryptoRepository.NewPostgreSQLDekRepository(db)
	dek := &cryptoDomain.Dek{
		ID:           dekID,
		KekID:        kekID,
		Algorithm:    cryptoDomain.AESGCM,
		EncryptedKey: []byte("encrypted-dek-data"),
		Nonce:        []byte("dek-nonce-12345"),
		CreatedAt:    time.Now().UTC(),
	}
	err = dekRepo.Create(ctx, dek)
	require.NoError(t, err)

	return kekID, dekID
}

// createTokenizationKey creates a tokenization key for testing and returns its ID
func createTokenizationKey(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	_, dekID := createKekAndDek(t, db)

	keyRepo := NewPostgreSQLTokenizationKeyRepository(db)
	keyID := uuid.Must(uuid.NewV7())
	tokKey := &tokenizationDomain.TokenizationKey{
		ID:              keyID,
		Name:            "test-key",
		Version:         1,
		FormatType:      tokenizationDomain.FormatUUID,
		IsDeterministic: false,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
		DeletedAt:       nil,
	}
	err := keyRepo.Create(ctx, tokKey)
	require.NoError(t, err)

	return keyID
}

func TestNewPostgreSQLTokenizationKeyRepository(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewPostgreSQLTokenizationKeyRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgreSQLTokenizationKeyRepository{}, repo)
}

func TestPostgreSQLTokenizationKeyRepository_Create(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenizationKeyRepository(db)
	ctx := context.Background()

	// Create DEK dependency
	_, dekID := createKekAndDek(t, db)

	key := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            "test-key",
		Version:         1,
		FormatType:      tokenizationDomain.FormatUUID,
		IsDeterministic: false,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
		DeletedAt:       nil,
	}

	err := repo.Create(ctx, key)
	require.NoError(t, err)

	// Verify by fetching
	retrieved, err := repo.GetByName(ctx, key.Name)
	require.NoError(t, err)
	assert.Equal(t, key.ID, retrieved.ID)
	assert.Equal(t, key.Name, retrieved.Name)
	assert.Equal(t, key.Version, retrieved.Version)
	assert.Equal(t, key.FormatType, retrieved.FormatType)
	assert.Equal(t, key.IsDeterministic, retrieved.IsDeterministic)
}

func TestPostgreSQLTokenizationKeyRepository_GetByName(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenizationKeyRepository(db)
	ctx := context.Background()

	// Create DEK dependency
	_, dekID := createKekAndDek(t, db)

	// Create first version
	key1 := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            "test-key",
		Version:         1,
		FormatType:      tokenizationDomain.FormatNumeric,
		IsDeterministic: true,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
		DeletedAt:       nil,
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// Create second version (newer)
	time.Sleep(time.Millisecond)
	key2 := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            "test-key",
		Version:         2,
		FormatType:      tokenizationDomain.FormatNumeric,
		IsDeterministic: true,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
		DeletedAt:       nil,
	}
	err = repo.Create(ctx, key2)
	require.NoError(t, err)

	// GetByName should return the latest (highest version)
	retrieved, err := repo.GetByName(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, key2.ID, retrieved.ID)
	assert.Equal(t, uint(2), retrieved.Version)
}

func TestPostgreSQLTokenizationKeyRepository_Delete(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenizationKeyRepository(db)
	ctx := context.Background()

	// Create DEK dependency
	_, dekID := createKekAndDek(t, db)

	key := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            "delete-test",
		Version:         1,
		FormatType:      tokenizationDomain.FormatUUID,
		IsDeterministic: false,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
		DeletedAt:       nil,
	}

	err := repo.Create(ctx, key)
	require.NoError(t, err)

	// Delete the key
	err = repo.Delete(ctx, key.ID)
	require.NoError(t, err)

	// Verify soft delete - key should not be found
	_, err = repo.Get(ctx, key.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tokenization key not found")
}

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

	tokenRepo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	// Create tokenization key dependency
	keyID := createTokenizationKey(t, db)

	valueHash := "test-hash"
	token := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_test123",
		ValueHash:         &valueHash,
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce123"),
		Metadata:          map[string]any{"key": "value"},
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         nil,
		RevokedAt:         nil,
	}

	err := tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	// Verify by fetching
	retrieved, err := tokenRepo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.Equal(t, token.ID, retrieved.ID)
	assert.Equal(t, token.Token, retrieved.Token)
	assert.Equal(t, token.Ciphertext, retrieved.Ciphertext)
}

func TestPostgreSQLTokenRepository_GetByValueHash(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKey(t, db)
	valueHash := "deterministic-hash"

	token := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_deterministic",
		ValueHash:         &valueHash,
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce"),
		Metadata:          nil,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         nil,
		RevokedAt:         nil,
	}

	err := repo.Create(ctx, token)
	require.NoError(t, err)

	// Retrieve by value hash
	retrieved, err := repo.GetByValueHash(ctx, keyID, valueHash)
	require.NoError(t, err)
	assert.Equal(t, token.ID, retrieved.ID)
	assert.Equal(t, token.Token, retrieved.Token)
}

func TestPostgreSQLTokenRepository_Revoke(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKey(t, db)

	token := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_revoke_test",
		ValueHash:         nil,
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce"),
		Metadata:          nil,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         nil,
		RevokedAt:         nil,
	}

	err := repo.Create(ctx, token)
	require.NoError(t, err)

	// Revoke the token
	err = repo.Revoke(ctx, token.Token)
	require.NoError(t, err)

	// Verify revocation
	retrieved, err := repo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.RevokedAt)
}

func TestPostgreSQLTokenRepository_CountExpired(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKey(t, db)
	pastTime := time.Now().UTC().Add(-2 * time.Hour)

	// Create expired token
	expiredToken := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_expired",
		ValueHash:         nil,
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce"),
		Metadata:          nil,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         &pastTime,
		RevokedAt:         nil,
	}
	err := repo.Create(ctx, expiredToken)
	require.NoError(t, err)

	// Create non-expired token
	futureTime := time.Now().UTC().Add(2 * time.Hour)
	validToken := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_valid",
		ValueHash:         nil,
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce"),
		Metadata:          nil,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         &futureTime,
		RevokedAt:         nil,
	}
	err = repo.Create(ctx, validToken)
	require.NoError(t, err)

	// Count expired tokens (check before current time minus 1 hour)
	beforeTimestamp := time.Now().UTC().Add(-1 * time.Hour)
	count, err := repo.CountExpired(ctx, beforeTimestamp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestPostgreSQLTokenRepository_DeleteExpired(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKey(t, db)
	pastTime := time.Now().UTC().Add(-2 * time.Hour)

	// Create expired token
	expiredToken := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_to_delete",
		ValueHash:         nil,
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce"),
		Metadata:          nil,
		CreatedAt:         time.Now().UTC(),
		ExpiresAt:         &pastTime,
		RevokedAt:         nil,
	}
	err := repo.Create(ctx, expiredToken)
	require.NoError(t, err)

	// Delete expired tokens (before current time minus 1 hour)
	beforeTimestamp := time.Now().UTC().Add(-1 * time.Hour)
	count, err := repo.DeleteExpired(ctx, beforeTimestamp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify deletion
	_, err = repo.GetByToken(ctx, expiredToken.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token not found")
}

func TestPostgreSQLTokenRepository_CountExpired_ZeroTime(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	// Test with zero time
	count, err := repo.CountExpired(ctx, time.Time{})
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Contains(t, err.Error(), "olderThan timestamp cannot be zero")
}

func TestPostgreSQLTokenRepository_DeleteExpired_ZeroTime(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupPostgresDB(t, db)

	repo := NewPostgreSQLTokenRepository(db)
	ctx := context.Background()

	// Test with zero time
	count, err := repo.DeleteExpired(ctx, time.Time{})
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Contains(t, err.Error(), "olderThan timestamp cannot be zero")
}
