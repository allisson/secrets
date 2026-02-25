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
	"github.com/allisson/secrets/internal/testutil"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// createKekAndDekMySQL creates a KEK and DEK for MySQL testing and returns their IDs
func createKekAndDekMySQL(t *testing.T, db *sql.DB) (kekID uuid.UUID, dekID uuid.UUID) {
	t.Helper()

	ctx := context.Background()

	// Create KEK
	kekID = uuid.Must(uuid.NewV7())
	kekRepo := cryptoRepository.NewMySQLKekRepository(db)
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
	dekRepo := cryptoRepository.NewMySQLDekRepository(db)
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

// createTokenizationKeyMySQL creates a tokenization key for MySQL testing and returns its ID
func createTokenizationKeyMySQL(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	_, dekID := createKekAndDekMySQL(t, db)

	keyRepo := NewMySQLTokenizationKeyRepository(db)
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

func TestNewMySQLTokenizationKeyRepository(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	assert.NotNil(t, repo)
	assert.IsType(t, &MySQLTokenizationKeyRepository{}, repo)
}

func TestMySQLTokenizationKeyRepository_Create(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	ctx := context.Background()

	// Create DEK dependency
	_, dekID := createKekAndDekMySQL(t, db)

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

func TestMySQLTokenizationKeyRepository_GetByName(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	ctx := context.Background()

	// Create DEK dependency
	_, dekID := createKekAndDekMySQL(t, db)

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

func TestMySQLTokenizationKeyRepository_Delete(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	ctx := context.Background()

	// Create DEK dependency
	_, dekID := createKekAndDekMySQL(t, db)

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

	tokenRepo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	// Create tokenization key dependency
	keyID := createTokenizationKeyMySQL(t, db)

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

func TestMySQLTokenRepository_GetByValueHash(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKeyMySQL(t, db)
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

func TestMySQLTokenRepository_Revoke(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKeyMySQL(t, db)

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

func TestMySQLTokenRepository_CountExpired(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKeyMySQL(t, db)
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

func TestMySQLTokenRepository_DeleteExpired(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	keyID := createTokenizationKeyMySQL(t, db)
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

func TestMySQLTokenRepository_CountExpired_ZeroTime(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	// Test with zero time
	count, err := repo.CountExpired(ctx, time.Time{})
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Contains(t, err.Error(), "olderThan timestamp cannot be zero")
}

func TestMySQLTokenRepository_DeleteExpired_ZeroTime(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenRepository(db)
	ctx := context.Background()

	// Test with zero time
	count, err := repo.DeleteExpired(ctx, time.Time{})
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Contains(t, err.Error(), "olderThan timestamp cannot be zero")
}

func TestMySQLTokenizationKeyRepository_List(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	ctx := context.Background()

	_, dekID := createKekAndDekMySQL(t, db)

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		key := &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            fmt.Sprintf("tok-key-%02d", i),
			Version:         1,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			DekID:           dekID,
			CreatedAt:       time.Now().UTC(),
		}
		err := repo.Create(ctx, key)
		require.NoError(t, err)

		time.Sleep(time.Millisecond)
		keyV2 := &tokenizationDomain.TokenizationKey{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            fmt.Sprintf("tok-key-%02d", i),
			Version:         2,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			DekID:           dekID,
			CreatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctx, keyV2)
		require.NoError(t, err)
	}

	keys, err := repo.List(ctx, 0, 3)
	require.NoError(t, err)
	assert.Len(t, keys, 3)
	assert.Equal(t, "tok-key-00", keys[0].Name)
	assert.Equal(t, uint(2), keys[0].Version)
	assert.Equal(t, "tok-key-01", keys[1].Name)
	assert.Equal(t, "tok-key-02", keys[2].Name)

	keys, err = repo.List(ctx, 3, 3)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, "tok-key-03", keys[0].Name)
	assert.Equal(t, "tok-key-04", keys[1].Name)
}

func TestMySQLTokenizationKeyRepository_Get(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	ctx := context.Background()
	_, dekID := createKekAndDekMySQL(t, db)

	key := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            "get-test-key",
		Version:         1,
		FormatType:      tokenizationDomain.FormatUUID,
		IsDeterministic: false,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)

	retrieved, err := repo.Get(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.ID, retrieved.ID)
	assert.Equal(t, key.Name, retrieved.Name)
}

func TestMySQLTokenizationKeyRepository_GetByNameAndVersion(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	repo := NewMySQLTokenizationKeyRepository(db)
	ctx := context.Background()
	_, dekID := createKekAndDekMySQL(t, db)

	key := &tokenizationDomain.TokenizationKey{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            "get-name-version-key",
		Version:         2,
		FormatType:      tokenizationDomain.FormatUUID,
		IsDeterministic: false,
		DekID:           dekID,
		CreatedAt:       time.Now().UTC(),
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)

	retrieved, err := repo.GetByNameAndVersion(ctx, key.Name, key.Version)
	require.NoError(t, err)
	assert.Equal(t, key.ID, retrieved.ID)
	assert.Equal(t, key.Version, retrieved.Version)
}

func TestMySQLTokenRepository_GetByToken(t *testing.T) {
	db := testutil.SetupMySQLDB(t)
	defer testutil.TeardownDB(t, db)
	defer testutil.CleanupMySQLDB(t, db)

	tokenRepo := NewMySQLTokenRepository(db)
	ctx := context.Background()
	keyID := createTokenizationKeyMySQL(t, db)

	token := &tokenizationDomain.Token{
		ID:                uuid.Must(uuid.NewV7()),
		TokenizationKeyID: keyID,
		Token:             "tok_getbytoken",
		Ciphertext:        []byte("encrypted"),
		Nonce:             []byte("nonce"),
		CreatedAt:         time.Now().UTC(),
	}
	err := tokenRepo.Create(ctx, token)
	require.NoError(t, err)

	retrieved, err := tokenRepo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.Equal(t, token.ID, retrieved.ID)
	assert.Equal(t, token.Token, retrieved.Token)
}
