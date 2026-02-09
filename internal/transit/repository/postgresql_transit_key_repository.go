// Package repository implements data persistence for transit encryption key management.
//
// This package provides repository implementations for storing and retrieving
// transit encryption keys in PostgreSQL and MySQL databases. Transit keys enable
// cryptographic operations (encrypt/decrypt) without exposing the actual key material
// to clients. Repositories follow the Repository pattern and support both direct
// database operations and transactional operations.
//
// # Key Components
//
// The package includes repositories for:
//   - TransitKey: Versioned encryption keys for transit encryption operations
//
// # Database Support
//
// Each repository type has two implementations:
//   - PostgreSQL: Uses native UUID type and BYTEA for binary data
//   - MySQL: Uses BINARY(16) for UUIDs and BLOB for binary data
//
// # Transaction Support
//
// All repositories support transaction-aware operations via database.GetTx(),
// enabling atomic multi-step operations. When called within a transaction context,
// repositories automatically use the transaction connection.
//
// # Transit Key Versioning
//
// Transit keys support versioning to enable key rotation without breaking existing
// encrypted data. Multiple versions of a key with the same name can coexist, but
// only the latest (highest version number) is returned by GetByName.
//
// # Soft Deletion
//
// Transit keys use soft deletion via the deleted_at timestamp. Deleted keys remain
// in the database but are filtered out from queries, allowing historical audit and
// recovery while preventing accidental reuse.
//
// # Usage Example
//
//	// Create transit key repository
//	transitKeyRepo := repository.NewPostgreSQLTransitKeyRepository(db)
//
//	// Use within a transaction
//	txManager := database.NewTxManager(db)
//	err := txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    return transitKeyRepo.Create(txCtx, transitKey)
//	})
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

// PostgreSQLTransitKeyRepository implements transit key persistence for PostgreSQL databases.
//
// This repository handles storing and retrieving transit encryption keys using
// PostgreSQL's native UUID type and timestamp with timezone for date fields.
// It supports transaction-aware operations via database.GetTx(), enabling atomic
// operations across multiple transit key modifications.
//
// Database schema requirements:
//   - id: UUID PRIMARY KEY
//   - name: TEXT (unique with version, identifies the key)
//   - version: INTEGER (for tracking key versions during rotation)
//   - dek_id: UUID (reference to the data encryption key)
//   - created_at: TIMESTAMP WITH TIME ZONE
//   - deleted_at: TIMESTAMP WITH TIME ZONE (nullable, for soft deletion)
//   - UNIQUE constraint on (name, version)
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
//	repo := NewPostgreSQLTransitKeyRepository(db)
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
type PostgreSQLTransitKeyRepository struct {
	db *sql.DB
}

// Create inserts a new transit key into the PostgreSQL database.
//
// The transit key's ID and DekID are stored as native UUIDs, and timestamps
// use TIMESTAMPTZ. This method supports transaction context via database.GetTx(),
// enabling atomic multi-step operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - transitKey: The transit key to insert (must have all required fields populated)
//
// Returns:
//   - An error if the insert fails (e.g., duplicate key, constraint violation)
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
func (p *PostgreSQLTransitKeyRepository) Create(
	ctx context.Context,
	transitKey *transitDomain.TransitKey,
) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO transit_keys (id, name, version, dek_id, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := querier.ExecContext(
		ctx,
		query,
		transitKey.ID,
		transitKey.Name,
		transitKey.Version,
		transitKey.DekID,
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
// current timestamp. The transit key remains in the database but is filtered
// out from GetByName queries. This approach preserves historical data while
// preventing the key from being used for new operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - transitKeyID: The UUID of the transit key to soft-delete
//
// Returns:
//   - An error if the update fails
//
// Example:
//
//	// Soft delete a transit key
//	err := repo.Delete(ctx, transitKeyID)
func (p *PostgreSQLTransitKeyRepository) Delete(ctx context.Context, transitKeyID uuid.UUID) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE transit_keys SET deleted_at = NOW() WHERE id = $1`

	_, err := querier.ExecContext(ctx, query, transitKeyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete transit key")
	}

	return nil
}

// GetByName retrieves the latest non-deleted version of a transit key by name.
//
// This method returns the transit key with the highest version number for the
// given name, excluding any soft-deleted keys (where deleted_at IS NOT NULL).
// This ensures clients always use the most recent active version of a key.
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
//   - An error if the query fails
//
// Example:
//
//	// Get the latest version of a transit key
//	transitKey, err := repo.GetByName(ctx, "payment-encryption")
//	if errors.Is(err, transitDomain.ErrTransitKeyNotFound) {
//	    // Handle key not found
//	}
func (p *PostgreSQLTransitKeyRepository) GetByName(
	ctx context.Context,
	name string,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = $1 AND deleted_at IS NULL 
			  ORDER BY version DESC 
			  LIMIT 1`

	var transitKey transitDomain.TransitKey
	err := querier.QueryRowContext(ctx, query, name).Scan(
		&transitKey.ID,
		&transitKey.Name,
		&transitKey.Version,
		&transitKey.DekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name")
	}

	return &transitKey, nil
}

// GetByNameAndVersion retrieves a specific version of a transit key by name and version.
//
// This method returns the transit key with the exact name and version number,
// excluding any soft-deleted keys (where deleted_at IS NOT NULL).
//
// The method supports transaction context via database.GetTx(), allowing
// consistent reads within a transaction.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - name: The name of the transit key to retrieve
//   - version: The specific version number to retrieve
//
// Returns:
//   - A pointer to the transit key matching the name and version
//   - transitDomain.ErrTransitKeyNotFound if no matching key exists or it is deleted
//   - An error if the query fails
//
// Example:
//
//	// Get version 2 of a transit key
//	transitKey, err := repo.GetByNameAndVersion(ctx, "payment-encryption", 2)
//	if errors.Is(err, transitDomain.ErrTransitKeyNotFound) {
//	    // Handle key not found
//	}
func (p *PostgreSQLTransitKeyRepository) GetByNameAndVersion(
	ctx context.Context,
	name string,
	version uint,
) (*transitDomain.TransitKey, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, name, version, dek_id, created_at, deleted_at 
			  FROM transit_keys 
			  WHERE name = $1 AND version = $2 AND deleted_at IS NULL`

	var transitKey transitDomain.TransitKey
	err := querier.QueryRowContext(ctx, query, name, version).Scan(
		&transitKey.ID,
		&transitKey.Name,
		&transitKey.Version,
		&transitKey.DekID,
		&transitKey.CreatedAt,
		&transitKey.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transitDomain.ErrTransitKeyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get transit key by name and version")
	}

	return &transitKey, nil
}

// NewPostgreSQLTransitKeyRepository creates a new PostgreSQL transit key repository instance.
//
// Parameters:
//   - db: A PostgreSQL database connection
//
// Returns:
//   - A new PostgreSQLTransitKeyRepository ready for use
//
// Example:
//
//	db, err := sql.Open("postgres", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewPostgreSQLTransitKeyRepository(db)
func NewPostgreSQLTransitKeyRepository(db *sql.DB) *PostgreSQLTransitKeyRepository {
	return &PostgreSQLTransitKeyRepository{db: db}
}
