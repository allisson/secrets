package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// MySQLSecretRepository implements Secret persistence for MySQL databases.
//
// This repository handles storing and retrieving secrets using MySQL's BINARY(16)
// for UUID storage and BLOB for binary data. UUIDs are marshaled/unmarshaled to/from
// binary format using uuid.MarshalBinary() and uuid.UnmarshalBinary(). It supports
// transaction-aware operations via database.GetTx(), enabling atomic operations such
// as secret creation and updates.
//
// Database schema requirements:
//   - id: BINARY(16) PRIMARY KEY (UUID in binary format)
//   - path: VARCHAR(255) (secret path identifier)
//   - version: INTEGER (for tracking secret versions)
//   - dek_id: BINARY(16) (foreign key reference to DEK)
//   - ciphertext: BLOB (encrypted secret data)
//   - nonce: BLOB (encryption nonce)
//   - created_at: DATETIME(6)
//   - deleted_at: DATETIME(6) (nullable, for soft deletes)
//   - UNIQUE KEY uk_secret_versions (path, version)
//
// UUID handling:
//
//	MySQL doesn't have a native UUID type, so UUIDs are stored as BINARY(16).
//	The repository handles marshaling/unmarshaling automatically using
//	uuid.MarshalBinary() and uuid.UnmarshalBinary() methods.
//
// Transaction support:
//
//	The repository automatically detects transaction context using database.GetTx().
//	All methods work both within and outside of transactions seamlessly.
//
// Example usage:
//
//	repo := NewMySQLSecretRepository(db)
//
//	// Create a secret outside transaction
//	err := repo.Create(ctx, secret)
//
//	// Or within a transaction
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    return repo.Create(txCtx, secret)
//	})
type MySQLSecretRepository struct {
	db *sql.DB
}

// Create inserts a new secret into the MySQL database.
//
// The secret's ID and DekID are marshaled to BINARY(16) format using uuid.MarshalBinary(),
// and binary fields (Ciphertext, Nonce) are stored as BLOBs. This method supports
// transaction context via database.GetTx(), enabling atomic multi-step operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - secret: The Secret to insert (must have all required fields populated)
//
// Returns:
//   - An error if marshaling the UUIDs fails or the insert fails
//
// Example:
//
//	secret := &secretsDomain.Secret{
//	    ID:         uuid.Must(uuid.NewV7()),
//	    Path:       "/app/database/password",
//	    Version:    1,
//	    DekID:      dekID,
//	    Ciphertext: encryptedBytes,
//	    Nonce:      nonceBytes,
//	    CreatedAt:  time.Now().UTC(),
//	}
//	err := repo.Create(ctx, secret)
func (m *MySQLSecretRepository) Create(ctx context.Context, secret *secretsDomain.Secret) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO secrets (id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	id, err := secret.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal secret id")
	}

	dekID, err := secret.DekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		secret.Path,
		secret.Version,
		dekID,
		secret.Ciphertext,
		secret.Nonce,
		secret.CreatedAt,
		secret.DeletedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create secret")
	}

	return nil
}

// GetByPath retrieves the latest non-deleted version of a secret by its path from the MySQL database.
//
// This method fetches a secret using its path identifier, returning only non-deleted versions.
// The secret is returned with all fields populated, including encrypted data. UUIDs are unmarshaled from BINARY(16)
// format. This method supports transaction context via database.GetTx(), enabling
// consistent reads within a transaction.
//
// Soft-deleted secrets (with deleted_at set) are excluded from results. If all versions of a secret
// at the given path are deleted, this method returns ErrNotFound.
//
// Note: This method returns the most recent non-deleted version of the secret at the given path.
// If multiple versions exist, it returns the one with the highest version number that has not been deleted.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - path: The secret path identifier (e.g., "/app/database/password")
//
// Returns:
//   - The Secret if found with all encrypted fields populated
//   - ErrNotFound if the secret doesn't exist at the specified path or all versions are deleted
//   - An error if the database query or UUID unmarshaling fails
//
// Example:
//
//	secret, err := repo.GetByPath(ctx, "/app/api-key")
//	if err != nil {
//	    if errors.Is(err, apperrors.ErrNotFound) {
//	        return nil, fmt.Errorf("secret not found")
//	    }
//	    return nil, err
//	}
//	// Use secret.DekID to retrieve the DEK for decryption
func (m *MySQLSecretRepository) GetByPath(
	ctx context.Context,
	path string,
) (*secretsDomain.Secret, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at 
			  FROM secrets 
			  WHERE path = ? AND deleted_at IS NULL
			  ORDER BY version DESC 
			  LIMIT 1`

	var secret secretsDomain.Secret
	var id, dekID []byte

	err := querier.QueryRowContext(ctx, query, path).Scan(
		&id,
		&secret.Path,
		&secret.Version,
		&dekID,
		&secret.Ciphertext,
		&secret.Nonce,
		&secret.CreatedAt,
		&secret.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get secret by path")
	}

	if err := secret.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal secret id")
	}

	if err := secret.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	return &secret, nil
}

// Delete performs a soft delete on a secret by setting the DeletedAt timestamp.
//
// This method does not physically remove the secret from the database. Instead, it sets
// the deleted_at field to the current timestamp, marking the secret as deleted while
// preserving it for audit purposes or potential recovery. The secretID is marshaled to
// BINARY(16) format for the query.
//
// This method supports transaction context via database.GetTx(), enabling atomic
// operations where multiple secrets might be deleted together.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - secretID: The UUID of the secret to soft delete
//
// Returns:
//   - An error if UUID marshaling fails or the update fails
//
// Example:
//
//	// Soft delete a secret
//	err := repo.Delete(ctx, secretID)
func (m *MySQLSecretRepository) Delete(ctx context.Context, secretID uuid.UUID) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE secrets 
			  SET deleted_at = ?
			  WHERE id = ?`

	id, err := secretID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal secret id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		time.Now().UTC(),
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete secret")
	}

	return nil
}

// NewMySQLSecretRepository creates a new MySQL Secret repository instance.
//
// Parameters:
//   - db: A MySQL database connection
//
// Returns:
//   - A new MySQLSecretRepository ready for use
//
// Example:
//
//	db, err := sql.Open("mysql", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewMySQLSecretRepository(db)
func NewMySQLSecretRepository(db *sql.DB) *MySQLSecretRepository {
	return &MySQLSecretRepository{db: db}
}
