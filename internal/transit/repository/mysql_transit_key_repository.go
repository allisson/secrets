package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// MySQLTransitKeyRepository implements transit key persistence for MySQL databases.
//
// This repository handles storing and retrieving transit encryption keys using
// MySQL's BINARY(16) for UUID storage and DATETIME for date fields. UUIDs are
// marshaled/unmarshaled to/from binary format using uuid.MarshalBinary() and
// uuid.UnmarshalBinary(). It supports transaction-aware operations via
// database.GetTx(), enabling atomic operations across multiple transit key
// modifications.
//
// Database schema requirements:
//   - id: BINARY(16) PRIMARY KEY (UUID in binary format)
//   - name: VARCHAR(255) (unique with version, identifies the key)
//   - version: INTEGER (for tracking key versions during rotation)
//   - dek_id: BINARY(16) (reference to the data encryption key)
//   - created_at: DATETIME/TIMESTAMP
//   - deleted_at: DATETIME/TIMESTAMP (nullable, for soft deletion)
//   - UNIQUE KEY on (name, version)
//
// UUID handling:
//
//	MySQL doesn't have a native UUID type, so UUIDs are stored as BINARY(16).
//	The repository handles marshaling/unmarshaling automatically using
//	uuid.MarshalBinary() and uuid.UnmarshalBinary() methods.
//
// Soft deletion:
//
//	Transit keys are soft-deleted by setting the deleted_at timestamp.
//	Queries automatically filter out soft-deleted records.
//
// Transaction support:
//
//	The repository automatically detects transaction context using database.GetTx().
//	All methods work both within and outside of transactions seamlessly.
//
// Example usage:
//
//	repo := NewMySQLTransitKeyRepository(db)
//
//	// Create a transit key outside transaction
//	err := repo.Create(ctx, transitKey)
//
//	// Or within a transaction
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    // Both operations use the same transaction
//	    if err := repo.Create(txCtx, transitKey1); err != nil {
//	        return err
//	    }
//	    return repo.Create(txCtx, transitKey2)
//	})
type MySQLTransitKeyRepository struct {
	db *sql.DB
}

// Create inserts a new transit key into the MySQL database.
//
// The transit key's ID and DekID are marshaled to BINARY(16) format using
// uuid.MarshalBinary(). This method supports transaction context via
// database.GetTx(), enabling atomic multi-step operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - transitKey: The transit key to insert (must have all required fields populated)
//
// Returns:
//   - An error if marshaling the UUID fails or the insert fails
//
// Example:
//
//	transitKey := &transitDomain.TransitKey{
//	    ID:        uuid.Must(uuid.NewV7()),
//	    Name:      "payment-encryption",
//	    Version:   1,
//	    DekID:     dekID,
//	    CreatedAt: time.Now().UTC(),
//	}
//	err := repo.Create(ctx, transitKey)
func (m *MySQLTransitKeyRepository) Create(ctx context.Context, transitKey *transitDomain.TransitKey) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	id, err := transitKey.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal transit key id")
	}

	dekID, err := transitKey.DekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		transitKey.Name,
		transitKey.Version,
		dekID,
		transitKey.CreatedAt,
		transitKey.DeletedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create transit key")
	}
	return nil
}

// Delete soft-deletes a transit key by setting its deleted_at timestamp.
//
// This method performs a soft delete by updating the deleted_at field to the
// current timestamp. The transit key ID is marshaled to BINARY(16) format for
// the WHERE clause using uuid.MarshalBinary(). The transit key remains in the
// database but is filtered out from GetByName queries.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - transitKeyID: The UUID of the transit key to soft-delete
//
// Returns:
//   - An error if marshaling the UUID fails or the update fails
//
// Example:
//
//	// Soft delete a transit key
//	err := repo.Delete(ctx, transitKeyID)
func (m *MySQLTransitKeyRepository) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE transit_keys SET deleted_at = NOW() WHERE id = ?`

	id, err := transitKeyID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal transit key id")
	}

	_, err = querier.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete transit key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a transit key by name.
//
// This method returns the transit key with the highest version number for the
// given name, excluding any soft-deleted keys (where deleted_at IS NOT NULL).
// Transit key IDs are automatically unmarshaled from BINARY(16) to uuid.UUID
// using uuid.UnmarshalBinary(). This ensures clients always use the most recent
// active version of a key.
//
// The method supports transaction context via database.GetTx(), allowing
// consistent reads within a transaction.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - name: The name of the transit key to retrieve
//
// Returns:
//   - A pointer to the transit key with the highest version number
//   - transitDomain.ErrTransitKeyNotFound if no matching key exists or all versions are deleted
//   - An error if the query fails or UUID unmarshaling fails
//
// Example:
//
//	// Get the latest version of a transit key
//	transitKey, err := repo.GetByName(ctx, "payment-encryption")
//	if errors.Is(err, transitDomain.ErrTransitKeyNotFound) {
//	    // Handle key not found
//	}
func (m *MySQLTransitKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = ? AND deleted_at IS NULL 
			  ORDER BY version DESC 
			  LIMIT 1`

	var transitKey transitDomain.TransitKey
	var id []byte
	var dekID []byte

	err := querier.QueryRowContext(ctx, query, name).Scan(
		&id,
		&transitKey.Name,
		&transitKey.Version,
		&dekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name")
	}

	if err := transitKey.ID.UnmarshalBinary(id); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal transit key id")
	}

	if err := transitKey.DekID.UnmarshalBinary(dekID); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	return &transitKey, nil
}

// NewMySQLTransitKeyRepository creates a new MySQL transit key repository instance.
//
// Parameters:
//   - db: A MySQL database connection
//
// Returns:
//   - A new MySQLTransitKeyRepository ready for use
//
// Example:
//
//	db, err := sql.Open("mysql", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewMySQLTransitKeyRepository(db)
func NewMySQLTransitKeyRepository(db *sql.DB) *MySQLTransitKeyRepository {
	return &MySQLTransitKeyRepository{db: db}
}
