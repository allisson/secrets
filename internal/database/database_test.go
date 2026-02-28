package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
