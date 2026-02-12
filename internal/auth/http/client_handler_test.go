package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/auth/http/dto"
	"github.com/allisson/secrets/internal/auth/usecase/mocks"
)

// setupTestHandler creates a test handler with mocked dependencies.
func setupTestHandler(t *testing.T) (*ClientHandler, *mocks.MockClientUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockClientUseCase := mocks.NewMockClientUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewClientHandler(mockClientUseCase, nil, logger)

	return handler, mockClientUseCase
}

func TestClientHandler_CreateHandler(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		plainSecret := "sec_1234567890abcdef"

		request := dto.CreateClientRequest{
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path: "/v1/secrets/*",
					Capabilities: []authDomain.Capability{
						authDomain.ReadCapability,
						authDomain.WriteCapability,
					},
				},
			},
		}

		expectedInput := &authDomain.CreateClientInput{
			Name:     request.Name,
			IsActive: request.IsActive,
			Policies: request.Policies,
		}

		expectedOutput := &authDomain.CreateClientOutput{
			ID:          clientID,
			PlainSecret: plainSecret,
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, expectedInput).
			Return(expectedOutput, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/clients", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.CreateClientResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, clientID.String(), response.ID)
		assert.Equal(t, plainSecret, response.Secret)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/clients", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ValidationFailed_MissingName", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		request := dto.CreateClientRequest{
			Name:     "",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		c, w := createTestContext(http.MethodPost, "/v1/clients", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ValidationFailed_EmptyPolicies", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		request := dto.CreateClientRequest{
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{},
		}

		c, w := createTestContext(http.MethodPost, "/v1/clients", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		request := dto.CreateClientRequest{
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/clients", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response["error"])
	})
}

func TestClientHandler_GetHandler(t *testing.T) {
	t.Run("Success_ValidUUID", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		expectedClient := &authDomain.Client{
			ID:       clientID,
			Name:     "Test Client",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
			CreatedAt: time.Now().UTC(),
		}

		mockUseCase.EXPECT().
			Get(mock.Anything, clientID).
			Return(expectedClient, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/clients/"+clientID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.GetHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ClientResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, clientID.String(), response.ID)
		assert.Equal(t, "Test Client", response.Name)
		assert.True(t, response.IsActive)
		assert.Len(t, response.Policies, 1)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/clients/invalid-uuid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.GetHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Get(mock.Anything, clientID).
			Return(nil, authDomain.ErrClientNotFound).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/clients/"+clientID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.GetHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response["error"])
	})
}

func TestClientHandler_UpdateHandler(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := dto.UpdateClientRequest{
			Name:     "Updated Client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/prod/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		expectedInput := &authDomain.UpdateClientInput{
			Name:     request.Name,
			IsActive: request.IsActive,
			Policies: request.Policies,
		}

		updatedClient := &authDomain.Client{
			ID:        clientID,
			Name:      request.Name,
			IsActive:  request.IsActive,
			Policies:  request.Policies,
			CreatedAt: time.Now().UTC(),
		}

		mockUseCase.EXPECT().
			Update(mock.Anything, clientID, expectedInput).
			Return(nil).
			Once()

		mockUseCase.EXPECT().
			Get(mock.Anything, clientID).
			Return(updatedClient, nil).
			Once()

		c, w := createTestContext(http.MethodPut, "/v1/clients/"+clientID.String(), request)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.UpdateHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ClientResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, clientID.String(), response.ID)
		assert.Equal(t, "Updated Client", response.Name)
		assert.False(t, response.IsActive)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		request := dto.UpdateClientRequest{
			Name:     "Updated Client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		c, w := createTestContext(http.MethodPut, "/v1/clients/invalid-uuid", request)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.UpdateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())

		c, w := createTestContext(http.MethodPut, "/v1/clients/"+clientID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.UpdateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ValidationFailed", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := dto.UpdateClientRequest{
			Name:     "",
			IsActive: true,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		c, w := createTestContext(http.MethodPut, "/v1/clients/"+clientID.String(), request)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.UpdateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := dto.UpdateClientRequest{
			Name:     "Updated Client",
			IsActive: false,
			Policies: []authDomain.PolicyDocument{
				{
					Path:         "/v1/secrets/*",
					Capabilities: []authDomain.Capability{authDomain.ReadCapability},
				},
			},
		}

		mockUseCase.EXPECT().
			Update(mock.Anything, clientID, mock.Anything).
			Return(authDomain.ErrClientNotFound).
			Once()

		c, w := createTestContext(http.MethodPut, "/v1/clients/"+clientID.String(), request)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.UpdateHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response["error"])
	})
}

func TestClientHandler_DeleteHandler(t *testing.T) {
	t.Run("Success_ValidUUID", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Delete(mock.Anything, clientID).
			Return(nil).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/clients/"+clientID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodDelete, "/v1/clients/invalid-uuid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ClientNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Delete(mock.Anything, clientID).
			Return(authDomain.ErrClientNotFound).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/clients/"+clientID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: clientID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response["error"])
	})
}
