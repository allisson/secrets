package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLPolicyRepository implements Policy persistence for MySQL.
// Uses BINARY(16) for UUIDs with transaction support via database.GetTx().
type MySQLPolicyRepository struct {
	db *sql.DB
}

// Create inserts a new Policy into the MySQL database.
func (m *MySQLPolicyRepository) Create(ctx context.Context, policy *authDomain.Policy) error {
	querier := database.GetTx(ctx, m.db)

	documentJSON, err := json.Marshal(policy.Document)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy document")
	}

	query := `INSERT INTO policies (id, name, document, created_at) 
			  VALUES (?, ?, ?, ?)`

	id, err := policy.ID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy id")
	}

	_, err = querier.ExecContext(
		ctx,
		query,
		id,
		policy.Name,
		documentJSON,
		policy.CreatedAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to create policy")
	}
	return nil
}

// Update modifies an existing Policy in the MySQL database.
func (m *MySQLPolicyRepository) Update(ctx context.Context, policy *authDomain.Policy) error {
	querier := database.GetTx(ctx, m.db)

	documentJSON, err := json.Marshal(policy.Document)
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy document")
	}

	query := `UPDATE policies 
			  SET document = ?, 
			  	  created_at = ?
			  WHERE name = ?`

	_, err = querier.ExecContext(
		ctx,
		query,
		documentJSON,
		policy.CreatedAt,
		policy.Name,
	)
	if err != nil {
		return apperrors.Wrap(err, "failed to update policy")
	}

	return nil
}

// Get retrieves a Policy by name from the MySQL database.
func (m *MySQLPolicyRepository) Get(ctx context.Context, name string) (*authDomain.Policy, error) {
	querier := database.GetTx(ctx, m.db)

	query := `SELECT id, name, document, created_at FROM policies WHERE name = ?`

	var policy authDomain.Policy
	var idBytes []byte
	var documentJSON []byte

	err := querier.QueryRowContext(ctx, query, name).Scan(
		&idBytes,
		&policy.Name,
		&documentJSON,
		&policy.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, authDomain.ErrPolicyNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get policy")
	}

	if err := policy.ID.UnmarshalBinary(idBytes); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal policy id")
	}

	if err := json.Unmarshal(documentJSON, &policy.Document); err != nil {
		return nil, apperrors.Wrap(err, "failed to unmarshal policy document")
	}

	return &policy, nil
}

// Delete removes a Policy by name from the MySQL database.
func (m *MySQLPolicyRepository) Delete(ctx context.Context, name string) error {
	querier := database.GetTx(ctx, m.db)

	query := `DELETE FROM policies WHERE name = ?`

	result, err := querier.ExecContext(ctx, query, name)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete policy")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return authDomain.ErrPolicyNotFound
	}

	return nil
}

// NewMySQLPolicyRepository creates a new MySQL Policy repository.
func NewMySQLPolicyRepository(db *sql.DB) *MySQLPolicyRepository {
	return &MySQLPolicyRepository{db: db}
}
