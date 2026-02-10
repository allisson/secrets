package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLKekRepository implements KEK persistence for MySQL.
// Uses BINARY(16) for UUIDs and BLOB for binary data with transaction support.
type MySQLKekRepository struct {
	db *sql.DB
}

// Create inserts a new KEK into the MySQL database.
func (m *MySQLKekRepository) Create(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	id, err := kek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create kek")
	}
	return nil
}

// Update modifies an existing KEK in the MySQL database.
func (m *MySQLKekRepository) Update(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE keks 
			  SET master_key_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  version = ?, 
				  created_at = ?
			  WHERE id = ?`

	id, err := kek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update kek")
	}

	return nil
}

// List retrieves all KEKs ordered by version descending (newest first).
func (m *MySQLKekRepository) List(ctx context.Context) ([]*cryptoDomain.Kek, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, master_key_id, algorithm, encrypted_key, nonce, version, created_at 
			  FROM keks ORDER BY version DESC`

	rows, err := querier.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var keks []*cryptoDomain.Kek
	for rows.Next() {
		var kek cryptoDomain.Kek
		var id []byte

		err := rows.Scan(
			&id,
			&kek.MasterKeyID,
			&kek.Algorithm,
			&kek.EncryptedKey,
			&kek.Nonce,
			&kek.Version,
			&kek.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := kek.ID.UnmarshalBinary(id); err != nil {
			return nil, apperrors.Wrap(err, "failed to unmarshal kek id")
		}

		keks = append(keks, &kek)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keks, nil
}

// NewMySQLKekRepository creates a new MySQL KEK repository.
func NewMySQLKekRepository(db *sql.DB) *MySQLKekRepository {
	return &MySQLKekRepository{db: db}
}
