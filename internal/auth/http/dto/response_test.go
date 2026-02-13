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

func TestMapClientsToListResponse(t *testing.T) {
	t.Run("Success_MapMultipleClients", func(t *testing.T) {
		now := time.Now().UTC()
		client1ID := uuid.Must(uuid.NewV7())
		client2ID := uuid.Must(uuid.NewV7())

		clients := []*authDomain.Client{
			{
				ID:       client1ID,
				Secret:   "hashed_secret_1",
				Name:     "Client 1",
				IsActive: true,
				Policies: []authDomain.PolicyDocument{
					{
						Path:         "/v1/secrets/*",
						Capabilities: []authDomain.Capability{authDomain.ReadCapability},
					},
				},
				CreatedAt: now,
			},
			{
				ID:       client2ID,
				Secret:   "hashed_secret_2",
				Name:     "Client 2",
				IsActive: false,
				Policies: []authDomain.PolicyDocument{
					{
						Path:         "/v1/clients/*",
						Capabilities: []authDomain.Capability{authDomain.WriteCapability},
					},
				},
				CreatedAt: now.Add(time.Hour),
			},
		}

		response := MapClientsToListResponse(clients)

		assert.Len(t, response.Clients, 2)
		assert.Equal(t, client1ID.String(), response.Clients[0].ID)
		assert.Equal(t, "Client 1", response.Clients[0].Name)
		assert.True(t, response.Clients[0].IsActive)
		assert.Equal(t, client2ID.String(), response.Clients[1].ID)
		assert.Equal(t, "Client 2", response.Clients[1].Name)
		assert.False(t, response.Clients[1].IsActive)
	})

	t.Run("Success_EmptyList", func(t *testing.T) {
		clients := []*authDomain.Client{}

		response := MapClientsToListResponse(clients)

		assert.NotNil(t, response.Clients)
		assert.Empty(t, response.Clients)
	})

	t.Run("Success_SingleClient", func(t *testing.T) {
		now := time.Now().UTC()
		clientID := uuid.Must(uuid.NewV7())

		clients := []*authDomain.Client{
			{
				ID:        clientID,
				Secret:    "hashed_secret",
				Name:      "Single Client",
				IsActive:  true,
				Policies:  []authDomain.PolicyDocument{},
				CreatedAt: now,
			},
		}

		response := MapClientsToListResponse(clients)

		assert.Len(t, response.Clients, 1)
		assert.Equal(t, clientID.String(), response.Clients[0].ID)
		assert.Equal(t, "Single Client", response.Clients[0].Name)
	})
}
