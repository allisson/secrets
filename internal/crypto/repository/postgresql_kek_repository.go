// Package repository implements data persistence for KEKs and DEKs.
//
// Provides PostgreSQL and MySQL implementations with transaction support via database.GetTx().
// PostgreSQL uses native UUID and BYTEA types, MySQL uses BINARY(16) and BLOB types.
package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLKekRepository implements KEK persistence for PostgreSQL.
// Uses native UUID and BYTEA types with transaction support via database.GetTx().
type PostgreSQLKekRepository struct {
	db *sql.DB
}

// Create inserts a new KEK into the PostgreSQL database.
func (p *PostgreSQLKekRepository) Create(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := querier.ExecContext(
		ctx,
		query,
		kek.ID,
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

// Update modifies an existing KEK in the PostgreSQL database.
func (p *PostgreSQLKekRepository) Update(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE keks 
			  SET master_key_id = $1, 
			  	  algorithm = $2,
				  encrypted_key = $3,
				  nonce = $4,
				  version = $5, 
				  created_at = $6
			  WHERE id = $7`

	_, err := querier.ExecContext(
		ctx,
		query,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.CreatedAt,
		kek.ID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update kek")
	}

	return nil
}

// List retrieves all KEKs ordered by version descending (newest first).
func (p *PostgreSQLKekRepository) List(ctx context.Context) ([]*cryptoDomain.Kek, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT * FROM keks ORDER BY version DESC`

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

		err := rows.Scan(
			&kek.ID,
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

		keks = append(keks, &kek)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keks, nil
}

// NewPostgreSQLKekRepository creates a new PostgreSQL KEK repository.
func NewPostgreSQLKekRepository(db *sql.DB) *PostgreSQLKekRepository {
	return &PostgreSQLKekRepository{db: db}
}
