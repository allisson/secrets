package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/allisson/secrets/internal/testutil"
)

func TestNewTxManager(t *testing.T) {
	testutil.SkipIfNoPostgres(t)
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	assert.NotNil(t, txManager)
	assert.IsType(t, &sqlTxManager{}, txManager)
}

func TestWithTx_Success(t *testing.T) {
	testutil.SkipIfNoPostgres(t)
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
	testutil.SkipIfNoPostgres(t)
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
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))
	mock.ExpectRollback() // Rollback should be called on commit error

	txManager := NewTxManager(db)
	ctx := context.Background()

	err = txManager.WithTx(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit error")
}

func TestWithTx_RollbackError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	testError := errors.New("original error")
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(errors.New("rollback error"))

	txManager := NewTxManager(db)
	ctx := context.Background()

	err = txManager.WithTx(ctx, func(ctx context.Context) error {
		return testError
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "original error")
	assert.Contains(t, err.Error(), "rollback error")
}

func TestWithTx_NestedTransaction(t *testing.T) {
	testutil.SkipIfNoPostgres(t)
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	ctx := context.Background()

	err := txManager.WithTx(ctx, func(ctx context.Context) error {
		tx1 := ctx.Value(txKey{})
		assert.NotNil(t, tx1)

		return txManager.WithTx(ctx, func(ctx context.Context) error {
			tx2 := ctx.Value(txKey{})
			assert.Equal(t, tx1, tx2) // Should be the same transaction
			return nil
		})
	})

	assert.NoError(t, err)
}

func TestGetTx_WithTransaction(t *testing.T) {
	testutil.SkipIfNoPostgres(t)
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
	testutil.SkipIfNoPostgres(t)
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	ctx := context.Background()
	querier := GetTx(ctx, db)

	assert.NotNil(t, querier)
	assert.Equal(t, db, querier)
}

func TestWithTx_Panic(t *testing.T) {
	testutil.SkipIfNoPostgres(t)
	db := testutil.SetupPostgresDB(t)
	defer testutil.TeardownDB(t, db)

	txManager := NewTxManager(db)
	ctx := context.Background()

	assert.Panics(t, func() {
		_ = txManager.WithTx(ctx, func(ctx context.Context) error {
			panic("something went wrong")
		})
	})
}
