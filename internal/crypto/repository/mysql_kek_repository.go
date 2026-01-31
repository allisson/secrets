package repository

import (
	"context"
	"database/sql"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLKekRepository implements KEK persistence for MySQL databases.
//
// This repository handles storing and retrieving Key Encryption Keys using
// MySQL's BINARY(16) for UUID storage and BLOB for binary data. UUIDs are
// marshaled/unmarshaled to/from binary format. It supports transaction-aware
// operations via database.GetTx().
type MySQLKekRepository struct {
	db *sql.DB
}

// Create inserts a new KEK into the MySQL database.
//
// The KEK's ID is marshaled to BINARY(16) format, and binary fields
// (EncryptedKey, Nonce) are stored as BLOBs. This method supports
// transaction context via database.GetTx().
func (m *MySQLKekRepository) Create(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, is_active, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

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
		kek.IsActive,
		kek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create kek")
	}
	return nil
}

// Update modifies an existing KEK in the MySQL database.
//
// This method updates all mutable fields of the KEK. The KEK ID is marshaled
// to BINARY(16) format for the WHERE clause. It's typically used to deactivate
// old KEKs during key rotation. The method supports transaction context via
// database.GetTx().
func (m *MySQLKekRepository) Update(ctx context.Context, kek *cryptoDomain.Kek) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE keks 
			  SET master_key_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  version = ?, 
			      is_active = ?,
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
		kek.IsActive,
		kek.CreatedAt,
		id,
	)

	return err
}

// List retrieves all KEKs from the MySQL database ordered by version descending.
//
// This method returns all KEKs (both active and inactive) sorted by version
// in descending order. KEK IDs are unmarshaled from BINARY(16) to uuid.UUID.
// This ordering is useful for key rotation scenarios where you need to identify
// the latest KEK version.
func (m *MySQLKekRepository) List(ctx context.Context) ([]*cryptoDomain.Kek, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, master_key_id, algorithm, encrypted_key, nonce, version, is_active, created_at 
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
			&kek.IsActive,
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

// NewMySQLKekRepository creates a new MySQL KEK repository instance.
func NewMySQLKekRepository(db *sql.DB) *MySQLKekRepository {
	return &MySQLKekRepository{db: db}
}
