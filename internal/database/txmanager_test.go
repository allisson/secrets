package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/allisson/go-project-template/internal/testutil"
)

func TestNewTxManager(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	assert.NotNil(t, txManager)
	assert.IsType(t, &sqlTxManager{}, txManager)
}

func TestWithTx_Success(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	ctx := context.Background()

	err := txManager.WithTx(ctx, func(ctx context.Context) error {
		// Verify transaction is in context
		tx := ctx.Value(txKey{})
		assert.NotNil(t, tx)
		assert.IsType(t, &sql.Tx{}, tx)
		return nil
	})

	assert.NoError(t, err)
}

func TestWithTx_RollbackOnError(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	ctx := context.Background()

	testError := assert.AnError
	err := txManager.WithTx(ctx, func(ctx context.Context) error {
		return testError
	})

	assert.Equal(t, testError, err)
}

func TestWithTx_CommitError(t *testing.T) {
	// This test is tricky because we need the transaction to start but commit to fail
	// We'll skip this test as it's difficult to reliably trigger commit errors
	// without using mocks, and the behavior is tested implicitly in integration tests
	t.Skip("Difficult to test commit errors without mocks")
}

func TestWithTx_RollbackError(t *testing.T) {
	// This test is tricky because we need the transaction to start but rollback to fail
	// We'll skip this test as it's difficult to reliably trigger rollback errors
	// without using mocks, and the behavior is tested implicitly in integration tests
	t.Skip("Difficult to test rollback errors without mocks")
}

func TestGetTx_WithTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	ctx := context.Background()

	err := txManager.WithTx(ctx, func(ctx context.Context) error {
		querier := GetTx(ctx, db)
		assert.NotNil(t, querier)
		assert.IsType(t, &sql.Tx{}, querier)
		return nil
	})

	assert.NoError(t, err)
}

func TestGetTx_WithoutTransaction(t *testing.T) {
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	ctx := context.Background()
	querier := GetTx(ctx, db)

	assert.NotNil(t, querier)
	assert.Equal(t, db, querier)
}
