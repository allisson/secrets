package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// PostgreSQLDekRepository implements DEK persistence for PostgreSQL.
// Uses native UUID and BYTEA types with transaction support via database.GetTx().
type PostgreSQLDekRepository struct {
	db *sql.DB
}

// Create inserts a new DEK into the PostgreSQL database.
func (p *PostgreSQLDekRepository) Create(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := querier.ExecContext(
		ctx,
		query,
		dek.ID,
		dek.KekID,
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

// Get retrieves a DEK by its ID from the PostgreSQL database.
func (p *PostgreSQLDekRepository) Get(
	ctx context.Context,
	dekID uuid.UUID,
) (*cryptoDomain.Dek, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at 
			  FROM deks 
			  WHERE id = $1`

	var dek cryptoDomain.Dek
	err := querier.QueryRowContext(ctx, query, dekID).Scan(
		&dek.ID,
		&dek.KekID,
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

	return &dek, nil
}

// Update modifies an existing DEK in the PostgreSQL database.
func (p *PostgreSQLDekRepository) Update(ctx context.Context, dek *cryptoDomain.Dek) error {
	querier := database.GetTx(ctx, p.db)

	query := `UPDATE deks 
			  SET kek_id = $1, 
			  	  algorithm = $2,
				  encrypted_key = $3,
				  nonce = $4,
				  created_at = $5
			  WHERE id = $6`

	_, err := querier.ExecContext(
		ctx,
		query,
		dek.KekID,
		dek.Algorithm,
		dek.EncryptedKey,
		dek.Nonce,
		dek.CreatedAt,
		dek.ID,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update dek")
	}

	return nil
}

// NewPostgreSQLDekRepository creates a new PostgreSQL DEK repository.
func NewPostgreSQLDekRepository(db *sql.DB) *PostgreSQLDekRepository {
	return &PostgreSQLDekRepository{db: db}
}
