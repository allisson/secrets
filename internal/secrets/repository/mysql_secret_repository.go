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
type MySQLSecretRepository struct {
	db *sql.DB
}

// Create inserts a new secret into the MySQL database.
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

// GetByPath retrieves the latest non-deleted version of a secret by its path.
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

// GetByPathAndVersion retrieves a specific version of a secret by its path and version number.
func (m *MySQLSecretRepository) GetByPathAndVersion(
	ctx context.Context,
	path string,
	version uint,
) (*secretsDomain.Secret, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, path, version, dek_id, ciphertext, nonce, created_at, deleted_at 
			  FROM secrets 
			  WHERE path = ? AND version = ? AND deleted_at IS NULL
			  LIMIT 1`

	var secret secretsDomain.Secret
	var id, dekID []byte

	err := querier.QueryRowContext(ctx, query, path, version).Scan(
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
		return nil, apperrors.Wrap(err, "failed to get secret by path and version")
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
func NewMySQLSecretRepository(db *sql.DB) *MySQLSecretRepository {
	return &MySQLSecretRepository{db: db}
}
