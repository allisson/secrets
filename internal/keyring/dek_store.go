package keyring

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

type dekStore interface {
	create(ctx context.Context, d *dek) error
	get(ctx context.Context, dekID uuid.UUID) (*dek, error)
	update(ctx context.Context, d *dek) error
	getBatchNotKekID(ctx context.Context, kekID uuid.UUID, limit int) ([]*dek, error)
}

type dekRepository struct {
	db *sql.DB
}

func newDekRepository(db *sql.DB) dekStore {
	return &dekRepository{db: db}
}

func (p *dekRepository) create(ctx context.Context, d *dek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO deks (id, kek_id, algorithm, encrypted_key, nonce, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := querier.ExecContext(
		ctx,
		query,
		d.id,
		d.kekID,
		d.algorithm,
		d.encryptedKey,
		d.nonce,
		d.createdAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create dek")
	}
	return nil
}

func (p *dekRepository) get(ctx context.Context, dekID uuid.UUID) (*dek, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at
			  FROM deks
			  WHERE id = $1`

	var d dek
	err := querier.QueryRowContext(ctx, query, dekID).Scan(
		&d.id,
		&d.kekID,
		&d.algorithm,
		&d.encryptedKey,
		&d.nonce,
		&d.createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDekNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get dek")
	}

	return &d, nil
}

func (p *dekRepository) update(ctx context.Context, d *dek) error {
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
		d.kekID,
		d.algorithm,
		d.encryptedKey,
		d.nonce,
		d.createdAt,
		d.id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update dek")
	}

	return nil
}

func (p *dekRepository) getBatchNotKekID(ctx context.Context, kekID uuid.UUID, limit int) ([]*dek, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, kek_id, algorithm, encrypted_key, nonce, created_at
			  FROM deks
			  WHERE kek_id != $1
			  ORDER BY created_at ASC
			  LIMIT $2`

	rows, err := querier.QueryContext(ctx, query, kekID, limit)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to query deks batch")
	}
	defer func() {
		_ = rows.Close()
	}()

	var deks []*dek
	for rows.Next() {
		var d dek
		if err := rows.Scan(
			&d.id,
			&d.kekID,
			&d.algorithm,
			&d.encryptedKey,
			&d.nonce,
			&d.createdAt,
		); err != nil {
			return nil, apperrors.Wrap(err, "failed to scan dek")
		}
		deks = append(deks, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err, "failed to iterate deks")
	}

	return deks, nil
}
