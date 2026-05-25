package keyring

import (
	"context"
	"database/sql"

	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

type kekStore interface {
	create(ctx context.Context, k *kek) error
	update(ctx context.Context, k *kek) error
	list(ctx context.Context) ([]*kek, error)
}

type kekRepository struct {
	db *sql.DB
}

func newKekRepository(db *sql.DB) kekStore {
	return &kekRepository{db: db}
}

func (p *kekRepository) create(ctx context.Context, k *kek) error {
	querier := database.GetTx(ctx, p.db)

	query := `INSERT INTO keks (id, master_key_id, algorithm, encrypted_key, nonce, version, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := querier.ExecContext(
		ctx,
		query,
		k.id,
		k.masterKeyID,
		k.algorithm,
		k.encryptedKey,
		k.nonce,
		k.version,
		k.createdAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create kek")
	}
	return nil
}

func (p *kekRepository) update(ctx context.Context, k *kek) error {
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
		k.masterKeyID,
		k.algorithm,
		k.encryptedKey,
		k.nonce,
		k.version,
		k.createdAt,
		k.id,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update kek")
	}

	return nil
}

func (p *kekRepository) list(ctx context.Context) ([]*kek, error) {
	querier := database.GetTx(ctx, p.db)

	query := `SELECT id, master_key_id, algorithm, encrypted_key, nonce, version, created_at
			  FROM keks
			  ORDER BY version DESC`

	rows, err := querier.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var keks []*kek
	for rows.Next() {
		var k kek
		err := rows.Scan(
			&k.id,
			&k.masterKeyID,
			&k.algorithm,
			&k.encryptedKey,
			&k.nonce,
			&k.version,
			&k.createdAt,
		)
		if err != nil {
			return nil, err
		}
		keks = append(keks, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keks, nil
}
