package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/database"
	apperrors "github.com/allisson/secrets/internal/errors"
)

// MySQLClientPoliciesRepository implements ClientPolicies persistence for MySQL.
// Uses BINARY(16) for UUIDs with transaction support via database.GetTx().
type MySQLClientPoliciesRepository struct {
	db *sql.DB
}

// Create inserts a new ClientPolicies relationship into the MySQL database.
func (m *MySQLClientPoliciesRepository) Create(
	ctx context.Context,
	clientPolicies *authDomain.ClientPolicies,
) error {
	querier := database.GetTx(ctx, m.db)

	query := `INSERT INTO client_policies (client_id, policy_id) VALUES (?, ?)`

	clientID, err := clientPolicies.ClientID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	policyID, err := clientPolicies.PolicyID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy id")
	}

	_, err = querier.ExecContext(ctx, query, clientID, policyID)
	if err != nil {
		return apperrors.Wrap(err, "failed to create client policies")
	}
	return nil
}

// Delete removes a ClientPolicies relationship from the MySQL database.
func (m *MySQLClientPoliciesRepository) Delete(
	ctx context.Context,
	clientID uuid.UUID,
	policyID uuid.UUID,
) error {
	querier := database.GetTx(ctx, m.db)

	query := `DELETE FROM client_policies WHERE client_id = ? AND policy_id = ?`

	clientIDBytes, err := clientID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal client id")
	}

	policyIDBytes, err := policyID.MarshalBinary()
	if err != nil {
		return apperrors.Wrap(err, "failed to marshal policy id")
	}

	result, err := querier.ExecContext(ctx, query, clientIDBytes, policyIDBytes)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete client policies")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return authDomain.ErrClientPoliciesNotFound
	}

	return nil
}

// NewMySQLClientPoliciesRepository creates a new MySQL ClientPolicies repository.
func NewMySQLClientPoliciesRepository(db *sql.DB) *MySQLClientPoliciesRepository {
	return &MySQLClientPoliciesRepository{db: db}
}
