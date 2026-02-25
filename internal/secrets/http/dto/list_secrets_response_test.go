package dto_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	"github.com/allisson/secrets/internal/secrets/http/dto"
)

func TestMapSecretsToListResponse(t *testing.T) {
	now := time.Now().UTC()
	secrets := []*secretsDomain.Secret{
		{
			ID:        uuid.Must(uuid.NewV7()),
			Path:      "/test/1",
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:        uuid.Must(uuid.NewV7()),
			Path:      "/test/2",
			Version:   2,
			CreatedAt: now,
		},
	}

	response := dto.MapSecretsToListResponse(secrets)

	assert.Len(t, response.Data, 2)
	assert.Equal(t, secrets[0].ID.String(), response.Data[0].ID)
	assert.Equal(t, secrets[0].Path, response.Data[0].Path)
	assert.Equal(t, secrets[0].Version, response.Data[0].Version)
	assert.Equal(t, secrets[0].CreatedAt, response.Data[0].CreatedAt)

	assert.Equal(t, secrets[1].ID.String(), response.Data[1].ID)
	assert.Equal(t, secrets[1].Path, response.Data[1].Path)
	assert.Equal(t, secrets[1].Version, response.Data[1].Version)
	assert.Equal(t, secrets[1].CreatedAt, response.Data[1].CreatedAt)
}
