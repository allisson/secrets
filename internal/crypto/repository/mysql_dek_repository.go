package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLDekRepository implements DEK persistence for MySQL.
// Uses BINARY(16) for UUIDs and BLOB for binary data with transaction support.
type MySQLDekRepository struct {
	db *sql.DB
}

// Create inserts a new DEK into the MySQL database.
func (m *MySQLDekRepository) Create(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	id, err := dek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	kekID, err := dek.KekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		kekID,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create dek")
	}
	return nil
}

// Get retrieves a DEK by its ID from the MySQL database.
func (m *MySQLDekRepository) Get(ctx context.Context, dekID uuid.UUID) (*cryptoDomain.Dek, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at 
			  FROM deks 
			  WHERE id = ?`

	id, err := dekID.MarshalBinary()
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to marshal dek id")
	}

	var dek cryptoDomain.Dek
	var idBytes, kekIDBytes []byte

	err = querier.QueryRowContext(ctx, query, id).Scan(
		&idBytes,
		&kekIDBytes,
		&dek.Algorithm,
		&dek.EncryptedKey,
		&dek.Nonce,
		&dek.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, cryptoDomain.ErrDekNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get dek")
	}

	if err := dek.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal dek id")
	}

	if err := dek.KekID.UnmarshalBinary(kekIDBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal kek id")
	}

	return &dek, nil
}

// Update modifies an existing DEK in the MySQL database.
func (m *MySQLDekRepository) Update(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, m.db)

	query := `UPDATE deks 
			  SET kek_id = ?, 
			  	  algorithm = ?,
				  encrypted_key = ?,
				  nonce = ?,
				  created_at = ?
			  WHERE id = ?`

	kekID, err := dek.KekID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal kek id")
	}

	id, err := dek.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal dek id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		kekID,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
		id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update dek")
	}

	return nil
}

// NewMySQLDekRepository creates a new MySQL DEK repository.
func NewMySQLDekRepository(db *sql.DB) *MySQLDekRepository {
	return &MySQLDekRepository{db: db}
}
