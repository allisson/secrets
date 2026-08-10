// Package repository implements data persistence for secret management.
// Repositories use PostgreSQL with automatic versioning and soft deletion.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
)

// SecretRepository implements Secret persistence for PostgreSQL databases.
type SecretRepository struct {
	db *sql.DB
}

// Create inserts a new secret into the PostgreSQL database.
func (p *SecretRepository) Create(ctx context.Context, secret *secretsDomain.Secret) error {
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
func (p *SecretRepository) GetByPath(
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, secretsDomain.ErrSecretNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get secret by path")
	}

	return &secret, nil
}

// GetByPathAndVersion retrieves a specific version of a secret by its path and version number.
func (p *SecretRepository) GetByPathAndVersion(
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, secretsDomain.ErrSecretNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get secret by path and version")
	}

	return &secret, nil
}

// Delete performs a soft delete on all versions of a secret by path.
func (p *SecretRepository) Delete(ctx context.Context, path string) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE secrets 
			  SET deleted_at = $1
			  WHERE path = $2 AND deleted_at IS NULL`

	_, err := querier.ExecContext(
		ctx,
		query,
		time.Now().UTC(),
		path,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete secret")
	}

	// Note: We intentionally don't check rowsAffected to make Delete idempotent.
	// Deleting a non-existent or already-deleted secret is not an error.
	return nil
}

// ListCursor retrieves secrets ordered by path ascending using cursor-based pagination.
func (p *SecretRepository) ListCursor(
	ctx context.Context,
	afterPath *string,
	limit int,
) ([]*secretsDomain.Secret, error) {
	records, err := database.ListLatestCursor(
		ctx,
		database.GetTx(ctx, p.db),
		"secrets",
		"path",
		"t.id, t.path, t.version, t.dek_id, t.ciphertext, t.nonce, t.created_at, t.deleted_at",
		afterPath,
		limit,
		func(rows *sql.Rows) (*secretsDomain.Secret, error) {
			var secret secretsDomain.Secret
			if err := rows.Scan(
				&secret.ID,
				&secret.Path,
				&secret.Version,
				&secret.DekID,
				&secret.Ciphertext,
				&secret.Nonce,
				&secret.CreatedAt,
				&secret.DeletedAt,
			); err != nil {
				return nil, err
			}
			return &secret, nil
		},
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list secrets with cursor")
	}
	return records, nil
}

// HardDelete permanently removes soft-deleted secrets older than the specified time.
// Only affects secrets where deleted_at IS NOT NULL.
// If dryRun is true, returns count without performing deletion.
// Returns the number of secrets that were (or would be) deleted.
func (p *SecretRepository) HardDelete(
	ctx context.Context,
	olderThan time.Time,
	dryRun bool,
) (int64, error) {
	count, err := database.HardDeleteOlderThan(ctx, database.GetTx(ctx, p.db), "secrets", olderThan, dryRun)
	if err != nil {
		return 0, apperrors.Wrap(err, "failed to hard delete secrets")
	}
	return count, nil
}

// NewSecretRepository creates a new PostgreSQL Secret repository instance.
func NewSecretRepository(db *sql.DB) *SecretRepository {
	return &SecretRepository{db: db}
}
