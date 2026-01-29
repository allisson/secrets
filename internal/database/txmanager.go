// Package database provides database connection management and configuration.
package database

import (
	"context"
	"database/sql"
)

// txKey is a context key type for storing database transactions.
type txKey struct{}

// Querier is an interface that represents a database query executor.
// It can be either *sql.DB or *sql.Tx
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TxManager manages database transactions
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// sqlTxManager implements TxManager for SQL databases.
type sqlTxManager struct {
	db *sql.DB
}

// NewTxManager creates a new TxManager
func NewTxManager(db *sql.DB) TxManager {
	return &sqlTxManager{db: db}
}

// WithTx executes the provided function within a database transaction
func (m *sqlTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

// GetTx retrieves a transaction from context, or falls back to the DB connection
func GetTx(ctx context.Context, db *sql.DB) Querier {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}
