package dto_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	"github.com/allisson/secrets/internal/transit/http/dto"
)

func TestMapTransitKeysToListResponse(t *testing.T) {
	now := time.Now().UTC()
	keys := []*transitDomain.TransitKey{
		{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      "key-1",
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:        uuid.Must(uuid.NewV7()),
			Name:      "key-2",
			Version:   2,
			CreatedAt: now,
		},
	}

	response := dto.MapTransitKeysToListResponse(keys)

	assert.Len(t, response.Data, 2)
	assert.Equal(t, keys[0].ID.String(), response.Data[0].ID)
	assert.Equal(t, keys[0].Name, response.Data[0].Name)
	assert.Equal(t, keys[0].Version, response.Data[0].Version)
	assert.Equal(t, keys[0].CreatedAt, response.Data[0].CreatedAt)

	assert.Equal(t, keys[1].ID.String(), response.Data[1].ID)
	assert.Equal(t, keys[1].Name, response.Data[1].Name)
	assert.Equal(t, keys[1].Version, response.Data[1].Version)
	assert.Equal(t, keys[1].CreatedAt, response.Data[1].CreatedAt)
}
