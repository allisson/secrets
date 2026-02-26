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

		assert.Len(t, response.Data, 2)
		assert.Equal(t, client1ID.String(), response.Data[0].ID)
		assert.Equal(t, "Client 1", response.Data[0].Name)
		assert.True(t, response.Data[0].IsActive)
		assert.Equal(t, client2ID.String(), response.Data[1].ID)
		assert.Equal(t, "Client 2", response.Data[1].Name)
		assert.False(t, response.Data[1].IsActive)
	})

	t.Run("Success_EmptyList", func(t *testing.T) {
		clients := []*authDomain.Client{}

		response := MapClientsToListResponse(clients)

		assert.NotNil(t, response.Data)
		assert.Empty(t, response.Data)
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

		assert.Len(t, response.Data, 1)
		assert.Equal(t, clientID.String(), response.Data[0].ID)
		assert.Equal(t, "Single Client", response.Data[0].Name)
	})
}

func TestMapAuditLogToResponse(t *testing.T) {
	t.Run("Success_MapAllFields", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		auditLog := &authDomain.AuditLog{
			ID:         id,
			RequestID:  requestID,
			ClientID:   clientID,
			Capability: authDomain.ReadCapability,
			Path:       "/v1/secrets/test",
			Metadata:   map[string]any{"key": "value", "count": 42},
			CreatedAt:  now,
		}

		response := MapAuditLogToResponse(auditLog)

		assert.Equal(t, id.String(), response.ID)
		assert.Equal(t, requestID.String(), response.RequestID)
		assert.Equal(t, clientID.String(), response.ClientID)
		assert.Equal(t, string(authDomain.ReadCapability), response.Capability)
		assert.Equal(t, "/v1/secrets/test", response.Path)
		assert.NotNil(t, response.Metadata)
		assert.Equal(t, "value", response.Metadata["key"])
		assert.Equal(t, 42, response.Metadata["count"])
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_NilMetadata", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		auditLog := &authDomain.AuditLog{
			ID:         id,
			RequestID:  requestID,
			ClientID:   clientID,
			Capability: authDomain.WriteCapability,
			Path:       "/v1/clients",
			Metadata:   nil,
			CreatedAt:  now,
		}

		response := MapAuditLogToResponse(auditLog)

		assert.Equal(t, id.String(), response.ID)
		assert.Equal(t, requestID.String(), response.RequestID)
		assert.Equal(t, clientID.String(), response.ClientID)
		assert.Equal(t, string(authDomain.WriteCapability), response.Capability)
		assert.Equal(t, "/v1/clients", response.Path)
		assert.Nil(t, response.Metadata)
		assert.Equal(t, now, response.CreatedAt)
	})

	t.Run("Success_DifferentCapabilities", func(t *testing.T) {
		testCases := []struct {
			name       string
			capability authDomain.Capability
		}{
			{"ReadCapability", authDomain.ReadCapability},
			{"WriteCapability", authDomain.WriteCapability},
			{"DeleteCapability", authDomain.DeleteCapability},
			{"EncryptCapability", authDomain.EncryptCapability},
			{"DecryptCapability", authDomain.DecryptCapability},
			{"RotateCapability", authDomain.RotateCapability},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				id := uuid.Must(uuid.NewV7())
				requestID := uuid.Must(uuid.NewV7())
				clientID := uuid.Must(uuid.NewV7())
				now := time.Now().UTC()

				auditLog := &authDomain.AuditLog{
					ID:         id,
					RequestID:  requestID,
					ClientID:   clientID,
					Capability: tc.capability,
					Path:       "/test",
					Metadata:   nil,
					CreatedAt:  now,
				}

				response := MapAuditLogToResponse(auditLog)

				assert.Equal(t, string(tc.capability), response.Capability)
			})
		}
	})
}

func TestMapAuditLogsToListResponse(t *testing.T) {
	t.Run("Success_MapMultipleAuditLogs", func(t *testing.T) {
		now := time.Now().UTC()
		id1 := uuid.Must(uuid.NewV7())
		id2 := uuid.Must(uuid.NewV7())
		requestID1 := uuid.Must(uuid.NewV7())
		requestID2 := uuid.Must(uuid.NewV7())
		clientID1 := uuid.Must(uuid.NewV7())
		clientID2 := uuid.Must(uuid.NewV7())

		auditLogs := []*authDomain.AuditLog{
			{
				ID:         id1,
				RequestID:  requestID1,
				ClientID:   clientID1,
				Capability: authDomain.ReadCapability,
				Path:       "/v1/secrets/test",
				Metadata:   map[string]any{"key": "value"},
				CreatedAt:  now,
			},
			{
				ID:         id2,
				RequestID:  requestID2,
				ClientID:   clientID2,
				Capability: authDomain.WriteCapability,
				Path:       "/v1/clients",
				Metadata:   nil,
				CreatedAt:  now.Add(-1 * time.Hour),
			},
		}

		response := MapAuditLogsToListResponse(auditLogs)

		assert.Len(t, response.Data, 2)
		assert.Equal(t, id1.String(), response.Data[0].ID)
		assert.Equal(t, requestID1.String(), response.Data[0].RequestID)
		assert.Equal(t, clientID1.String(), response.Data[0].ClientID)
		assert.Equal(t, string(authDomain.ReadCapability), response.Data[0].Capability)
		assert.NotNil(t, response.Data[0].Metadata)
		assert.Equal(t, id2.String(), response.Data[1].ID)
		assert.Nil(t, response.Data[1].Metadata)
	})

	t.Run("Success_EmptyList", func(t *testing.T) {
		auditLogs := []*authDomain.AuditLog{}

		response := MapAuditLogsToListResponse(auditLogs)

		assert.NotNil(t, response.Data)
		assert.Empty(t, response.Data)
	})

	t.Run("Success_SingleAuditLog", func(t *testing.T) {
		now := time.Now().UTC()
		id := uuid.Must(uuid.NewV7())
		requestID := uuid.Must(uuid.NewV7())
		clientID := uuid.Must(uuid.NewV7())

		auditLogs := []*authDomain.AuditLog{
			{
				ID:         id,
				RequestID:  requestID,
				ClientID:   clientID,
				Capability: authDomain.EncryptCapability,
				Path:       "/v1/transit/keys/test/encrypt",
				Metadata:   map[string]any{"algorithm": "aes-gcm"},
				CreatedAt:  now,
			},
		}

		response := MapAuditLogsToListResponse(auditLogs)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, id.String(), response.Data[0].ID)
		assert.Equal(t, string(authDomain.EncryptCapability), response.Data[0].Capability)
		assert.NotNil(t, response.Data[0].Metadata)
	})
}
