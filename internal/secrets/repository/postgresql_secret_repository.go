// Package repository implements data persistence for secret management.
//
// This package provides repository implementations for storing and retrieving
// encrypted secrets in PostgreSQL and MySQL databases. Repositories follow the
// Repository pattern and support both direct database operations and transactional
// operations with automatic versioning.
//
// # Key Components
//
// The package includes repositories for:
//   - Secret storage with automatic version tracking
//   - Soft deletion with timestamp preservation
//   - Path-based retrieval with version ordering
//
// # Database Support
//
// Each repository has two implementations:
//   - PostgreSQL: Uses native UUID type and BYTEA for binary data
//   - MySQL: Uses BINARY(16) for UUIDs and BLOB for binary data
//
// # Versioning System
//
// Secrets use an immutable versioning model:
//   - Each update creates a new database row with incremented version
//   - GetByPath returns the latest version (highest version number)
//   - Old versions remain in database for audit trail
//   - Each version has its own DEK for cryptographic isolation
//
// # Soft Deletion
//
// Secrets are soft-deleted by setting deleted_at timestamp:
//   - Data remains in database for audit purposes
//   - GetByPath still returns deleted secrets (with deleted_at set)
//   - Application logic determines whether to use deleted secrets
//
// # Transaction Support
//
// All repositories support transaction-aware operations via database.GetTx(),
// enabling atomic multi-step operations such as secret creation with version
// tracking. When called within a transaction context, repositories automatically
// use the transaction connection.
//
// # Usage Example
//
//	// Create secret repository
//	secretRepo := repository.NewPostgreSQLSecretRepository(db)
//
//	// Create a new secret version
//	secret := &domain.Secret{
//	    ID:      uuid.Must(uuid.NewV7()),
//	    Path:    "/app/api-key",
//	    Version: 1,
//	    DekID:   dekID,
//	    // ... encrypted data ...
//	}
//	err := secretRepo.Create(ctx, secret)
//
//	// Get latest version
//	latest, err := secretRepo.GetByPath(ctx, "/app/api-key")
//
//	// Use within a transaction
//	txManager := database.NewTxManager(db)
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    return secretRepo.Create(txCtx, newVersion)
//	})
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

// PostgreSQLSecretRepository implements Secret persistence for PostgreSQL databases.
//
// This repository handles storing and retrieving secrets using PostgreSQL's native
// UUID type and BYTEA for binary data. It supports transaction-aware operations via
// database.GetTx(), enabling atomic operations such as secret creation and updates.
//
// Database schema requirements:
//   - id: UUID PRIMARY KEY
//   - path: TEXT (secret path identifier)
//   - version: INTEGER (for tracking secret versions)
//   - dek_id: UUID FOREIGN KEY (reference to DEK used for encryption)
//   - ciphertext: BYTEA (encrypted secret data)
//   - nonce: BYTEA (encryption nonce)
//   - created_at: TIMESTAMP WITH TIME ZONE
//   - deleted_at: TIMESTAMP WITH TIME ZONE (nullable, for soft deletes)
//
// Transaction support:
//
//	The repository automatically detects transaction context using database.GetTx().
//	All methods work both within and outside of transactions seamlessly.
//
// Example usage:
//
//	repo := NewPostgreSQLSecretRepository(db)
//
//	// Create a secret outside transaction
//	err := repo.Create(ctx, secret)
//
//	// Or within a transaction
//	err = txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    return repo.Create(txCtx, secret)
//	})
type PostgreSQLSecretRepository struct {
	db *sql.DB
}

// Create inserts a new secret into the PostgreSQL database.
//
// The secret's ID is stored as a native UUID, and binary fields (Ciphertext, Nonce)
// are stored as BYTEA. This method supports transaction context via database.GetTx(),
// enabling atomic multi-step operations.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - secret: The Secret to insert (must have all required fields populated)
//
// Returns:
//   - An error if the insert fails (e.g., duplicate key, constraint violation)
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
func (p *PostgreSQLSecretRepository) Create(ctx context.Context, secret *secretsDomain.Secret) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO secrets (id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := querier.ExecContext(
		ctx,
		query,
		secret.ID,
		secret.Path,
		secret.Version,
		secret.DekID,
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

// GetByPath retrieves the latest non-deleted version of a secret by its path from the PostgreSQL database.
//
// This method fetches a secret using its path identifier, returning only non-deleted versions.
// The secret is returned with all fields populated, including encrypted data. This method supports
// transaction context via database.GetTx(), enabling consistent reads within a transaction.
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
//   - An error if the database query fails
//
// Example:
//
//	secret, err := repo.GetByPath(ctx, "/app/api-key")
//	if err != nil {
//	    if errors.Is(err, errors.ErrNotFound) {
//	        return nil, fmt.Errorf("secret not found")
//	    }
//	    return nil, err
//	}
//	// Use secret.DekID to retrieve the DEK for decryption
func (p *PostgreSQLSecretRepository) GetByPath(
	ctx context.Context,
	path string,
) (*secretsDomain.Secret, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at 
			  FROM secrets 
			  WHERE path = $1 AND deleted_at IS NULL
			  ORDER BY version DESC 
			  LIMIT 1`

	var secret secretsDomain.Secret
	err := querier.QueryRowContext(ctx, query, path).Scan(
		&secret.ID,
		&secret.Path,
		&secret.Version,
		&secret.DekID,
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

	return &secret, nil
}

// Delete performs a soft delete on a secret by setting the DeletedAt timestamp.
//
// This method does not physically remove the secret from the database. Instead,
// it sets the deleted_at field to the current timestamp, marking the secret as
// deleted while preserving it for audit purposes or potential recovery.
//
// This method supports transaction context via database.GetTx(), enabling atomic
// operations where multiple secrets might be deleted together.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and transaction propagation
//   - secretID: The UUID of the secret to soft delete
//
// Returns:
//   - An error if the update fails (e.g., secret not found)
//
// Example:
//
//	// Soft delete a secret
//	err := repo.Delete(ctx, secretID)
func (p *PostgreSQLSecretRepository) Delete(ctx context.Context, secretID uuid.UUID) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE secrets 
			  SET deleted_at = $1
			  WHERE id = $2`

	_, err := querier.ExecContext(
		ctx,
		query,
		time.Now().UTC(),
		secretID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete secret")
	}

	return nil
}

// NewPostgreSQLSecretRepository creates a new PostgreSQL Secret repository instance.
//
// Parameters:
//   - db: A PostgreSQL database connection
//
// Returns:
//   - A new PostgreSQLSecretRepository ready for use
//
// Example:
//
//	db, err := sql.Open("postgres", dsn)
//	if err != nil {
//	    return nil, err
//	}
//	repo := NewPostgreSQLSecretRepository(db)
func NewPostgreSQLSecretRepository(db *sql.DB) *PostgreSQLSecretRepository {
	return &PostgreSQLSecretRepository{db: db}
}
