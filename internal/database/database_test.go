package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/allisson/secrets/internal/testutil"
)

func TestConnect_Success(t *testing.T) {
	testutil.SkipIfNoPostgres(t)

	cfg := Config{
		Driver:             "postgres",
		ConnectionString:   testutil.GetPostgresTestDSN(),
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
		ConnMaxLifetime:    time.Hour,
		ConnMaxIdleTime:    time.Minute,
	}

	db, err := Connect(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	defer func() {
		_ = db.Close()
	}()

	assert.Equal(t, 10, db.Stats().MaxOpenConnections)
}

func TestConnect_Error(t *testing.T) {
	cfg := Config{
		Driver:             "invalid",
		ConnectionString:   "invalid",
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
		ConnMaxLifetime:    time.Hour,
	}

	db, err := Connect(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "sql: unknown driver")
}
