package dto_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	"github.com/allisson/secrets/internal/tokenization/http/dto"
)

func TestMapTokenizationKeysToListResponse(t *testing.T) {
	now := time.Now().UTC()
	keys := []*tokenizationDomain.TokenizationKey{
		{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            "key-1",
			Version:         1,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			CreatedAt:       now,
		},
		{
			ID:              uuid.Must(uuid.NewV7()),
			Name:            "key-2",
			Version:         2,
			FormatType:      tokenizationDomain.FormatNumeric,
			IsDeterministic: true,
			CreatedAt:       now,
		},
	}

	response := dto.MapTokenizationKeysToListResponse(keys)

	assert.Len(t, response.Items, 2)
	assert.Equal(t, keys[0].ID.String(), response.Items[0].ID)
	assert.Equal(t, keys[0].Name, response.Items[0].Name)
	assert.Equal(t, keys[0].Version, response.Items[0].Version)
	assert.Equal(t, string(keys[0].FormatType), response.Items[0].FormatType)
	assert.Equal(t, keys[0].IsDeterministic, response.Items[0].IsDeterministic)
	assert.Equal(t, keys[0].CreatedAt, response.Items[0].CreatedAt)

	assert.Equal(t, keys[1].ID.String(), response.Items[1].ID)
	assert.Equal(t, keys[1].Name, response.Items[1].Name)
	assert.Equal(t, keys[1].Version, response.Items[1].Version)
	assert.Equal(t, string(keys[1].FormatType), response.Items[1].FormatType)
	assert.Equal(t, keys[1].IsDeterministic, response.Items[1].IsDeterministic)
	assert.Equal(t, keys[1].CreatedAt, response.Items[1].CreatedAt)
}
