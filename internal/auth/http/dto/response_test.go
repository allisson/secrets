package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

func TestMapClientToResponse(t *testing.T) {
	t.Run("Success_MapAllFields", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed_secret",
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
			CreatedAt: now,
		}

		response := MapClientToResponse(client)

		assert.Equal(t, clientID.String(), response.ID)
		assert.Equal(t, "Test Client", response.Name)
		assert.True(t, response.IsActive)
		assert.Len(t, response.Policies, 1)
		assert.Equal(t, "/v1/secrets/*", response.Policies[0].Path)
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_InactiveClient", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed_secret",
			Name:     "Inactive Client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/read-only/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
			CreatedAt: now,
		}

		response := MapClientToResponse(client)

		assert.Equal(t, clientID.String(), response.ID)
		assert.Equal(t, "Inactive Client", response.Name)
		assert.False(t, response.IsActive)
		assert.Len(t, response.Policies, 1)
	})

	t.Run("Success_MultiplePolicies", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		client := &authDomain.Client{
			ID:       clientID,
			Secret:   "hashed_secret",
			Name:     "Multi Policy Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path: "/v1/secrets/*",
					Capabilities: []authDomain.Capability{
						authDomain.ReadCapability,
						authDomain.WriteCapability,
					},
				},
				{
					Path:         "/v1/clients/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
			CreatedAt: now,
		}

		response := MapClientToResponse(client)

		assert.Equal(t, clientID.String(), response.ID)
		assert.Len(t, response.Policies, 2)
		assert.Equal(t, "/v1/secrets/*", response.Policies[0].Path)
		assert.Equal(t, "/v1/clients/*", response.Policies[1].Path)
	})
}
