// Package repository implements data persistence for secret management.
// Repositories support both PostgreSQL and MySQL with automatic versioning and soft deletion.
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
type PostgreSQLSecretRepository struct {
	db *sql.DB
}

// Create inserts a new secret into the PostgreSQL database.
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

// GetByPath retrieves the latest non-deleted version of a secret by its path.
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

// GetByPathAndVersion retrieves a specific version of a secret by its path and version number.
func (p *PostgreSQLSecretRepository) GetByPathAndVersion(
	ctx context.Context,
	path string,
	version uint,
) (*secretsDomain.Secret, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at 
			  FROM secrets 
			  WHERE path = $1 AND version = $2 AND deleted_at IS NULL
			  LIMIT 1`

	var secret secretsDomain.Secret
	err := querier.QueryRowContext(ctx, query, path, version).Scan(
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
		return nil, apperrors.Wrap(err, "failed to get secret by path and version")
	}

	return &secret, nil
}

// Delete performs a soft delete on a secret by setting the DeletedAt timestamp.
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
func NewPostgreSQLSecretRepository(db *sql.DB) *PostgreSQLSecretRepository {
	return &PostgreSQLSecretRepository{db: db}
}
