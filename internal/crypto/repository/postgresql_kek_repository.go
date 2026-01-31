// Package repository implements data persistence for cryptographic key management.
//
// This package provides repository implementations for storing and retrieving
// Key Encryption Keys (KEKs) in PostgreSQL and MySQL databases. Repositories
// follow the Repository pattern and support both direct database operations
// and transactional operations.
package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLKekRepository implements KEK persistence for PostgreSQL databases.
//
// This repository handles storing and retrieving Key Encryption Keys using
// PostgreSQL's native UUID type and BYTEA for binary data. It supports
// transaction-aware operations via database.GetTx().
type PostgreSQLKekRepository struct {
	db *sql.DB
}

// Create inserts a new KEK into the PostgreSQL database.
//
// The KEK's ID is stored as a native UUID, and binary fields (EncryptedKey, Nonce)
// are stored as BYTEA. This method supports transaction context via database.GetTx().
func (p *PostgreSQLKekRepository) Create(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, is_active, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := querier.ExecContext(
		ctx,
		query,
		kek.ID,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.IsActive,
		kek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create kek")
	}
	return nil
}

// Update modifies an existing KEK in the PostgreSQL database.
//
// This method updates all mutable fields of the KEK. It's typically used
// to deactivate old KEKs during key rotation. The method supports
// transaction context via database.GetTx().
func (p *PostgreSQLKekRepository) Update(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE keks 
			  SET master_key_id = $1, 
			  	  algorithm = $2,
				  encrypted_key = $3,
				  nonce = $4,
				  version = $5, 
			      is_active = $6,
				  created_at = $7
			  WHERE id = $8`

	_, err := querier.ExecContext(
		ctx,
		query,
		kek.MasterKeyID,
		kek.Algorithm,
		kek.EncryptedKey,
		kek.Nonce,
		kek.Version,
		kek.IsActive,
		kek.CreatedAt,
		kek.ID,
	)

	return err
}

// List retrieves all KEKs from the PostgreSQL database ordered by version descending.
//
// This method returns all KEKs (both active and inactive) sorted by version
// in descending order, ensuring the newest KEK appears first. This ordering
// is useful for key rotation scenarios where you need to identify the latest
// KEK version.
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
			&kek.IsActive,
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

// NewPostgreSQLKekRepository creates a new PostgreSQL KEK repository instance.
func NewPostgreSQLKekRepository(db *sql.DB) *PostgreSQLKekRepository {
	return &PostgreSQLKekRepository{db: db}
}
