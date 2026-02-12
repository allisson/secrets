package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
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

// MockTokenUseCase is a mock implementation of TokenUseCase for testing.
type MockTokenUseCase struct {
	mock.Mock
}

func (m *MockTokenUseCase) Issue(
	ctx context.Context,
	input *authDomain.IssueTokenInput,
) (*authDomain.IssueTokenOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.IssueTokenOutput), args.Error(1)
}

func (m *MockTokenUseCase) Authenticate(ctx context.Context, tokenHash string) (*authDomain.Client, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authDomain.Client), args.Error(1)
}

// setupTokenTestHandler creates a test token handler with mocked dependencies.
func setupTokenTestHandler(t *testing.T) (*TokenHandler, *MockTokenUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockTokenUseCase := &MockTokenUseCase{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewTokenHandler(mockTokenUseCase, logger)

	return handler, mockTokenUseCase
}

// createTestContext creates a test Gin context with the given request.
func createTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	return c, w
}

func TestClientHandler_CreateHandler(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		handler, mockUseCase := setupTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		plainSecret := "sec_1234567890abcdef"

		request := CreateClientRequest{
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

		var response CreateClientResponse
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

		request := CreateClientRequest{
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

		request := CreateClientRequest{
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

		request := CreateClientRequest{
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

		var response ClientResponse
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
		request := UpdateClientRequest{
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

		var response ClientResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, clientID.String(), response.ID)
		assert.Equal(t, "Updated Client", response.Name)
		assert.False(t, response.IsActive)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		handler, _ := setupTestHandler(t)

		request := UpdateClientRequest{
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
		request := UpdateClientRequest{
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
		request := UpdateClientRequest{
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

func TestValidatePolicyDocument(t *testing.T) {
	t.Run("Success_ValidPolicy", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "/v1/secrets/*",
			Capabilities: []authDomain.Capability{authDomain.ReadCapability, authDomain.WriteCapability},
		}

		err := validatePolicyDocument(policy)
		assert.NoError(t, err)
	})

	t.Run("Error_EmptyPath", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "",
			Capabilities: []authDomain.Capability{authDomain.ReadCapability},
		}

		err := validatePolicyDocument(policy)
		assert.Error(t, err)
	})

	t.Run("Error_EmptyCapabilities", func(t *testing.T) {
		policy := authDomain.PolicyDocument{
			Path:         "/v1/secrets/*",
			Capabilities: []authDomain.Capability{},
		}

		err := validatePolicyDocument(policy)
		assert.Error(t, err)
	})

	t.Run("Error_InvalidType", func(t *testing.T) {
		err := validatePolicyDocument("not a policy")
		assert.Error(t, err)
	})
}

func TestMapClientToResponse(t *testing.T) {
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

	response := mapClientToResponse(client)

	assert.Equal(t, clientID.String(), response.ID)
	assert.Equal(t, "Test Client", response.Name)
	assert.True(t, response.IsActive)
	assert.Len(t, response.Policies, 1)
	assert.Equal(t, "/v1/secrets/*", response.Policies[0].Path)
	assert.Equal(t, now, response.CreatedAt)
}

func TestTokenHandler_IssueTokenHandler(t *testing.T) {
	t.Run("Success_ValidCredentials", func(t *testing.T) {
		handler, mockUseCase := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		plainToken := "tok_1234567890abcdef"
		expiresAt := time.Now().UTC().Add(1 * time.Hour)

		request := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		expectedInput := &authDomain.IssueTokenInput{
			ClientID:     clientID,
			ClientSecret: "test_secret_123",
		}

		expectedOutput := &authDomain.IssueTokenOutput{
			PlainToken: plainToken,
			ExpiresAt:  expiresAt,
		}

		mockUseCase.On("Issue", mock.Anything, expectedInput).
			Return(expectedOutput, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response IssueTokenResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, plainToken, response.Token)
		assert.Equal(t, expiresAt.Unix(), response.ExpiresAt.Unix())

		mockUseCase.AssertExpectations(t)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTokenTestHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/token", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_MissingClientID", func(t *testing.T) {
		handler, _ := setupTokenTestHandler(t)

		request := IssueTokenRequest{
			ClientID:     "",
			ClientSecret: "test_secret_123",
		}

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_MissingClientSecret", func(t *testing.T) {
		handler, _ := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidUUIDFormat", func(t *testing.T) {
		handler, _ := setupTokenTestHandler(t)

		request := IssueTokenRequest{
			ClientID:     "invalid-uuid",
			ClientSecret: "test_secret_123",
		}

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_BlankClientSecret", func(t *testing.T) {
		handler, _ := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "   ",
		}

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidCredentials", func(t *testing.T) {
		handler, mockUseCase := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "wrong_secret",
		}

		mockUseCase.On("Issue", mock.Anything, mock.Anything).
			Return(nil, authDomain.ErrInvalidCredentials).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "unauthorized", response["error"])

		mockUseCase.AssertExpectations(t)
	})

	t.Run("Error_ClientInactive", func(t *testing.T) {
		handler, mockUseCase := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		mockUseCase.On("Issue", mock.Anything, mock.Anything).
			Return(nil, authDomain.ErrClientInactive).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "forbidden", response["error"])

		mockUseCase.AssertExpectations(t)
	})

	t.Run("Error_RepositoryError", func(t *testing.T) {
		handler, mockUseCase := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		request := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		mockUseCase.On("Issue", mock.Anything, mock.Anything).
			Return(nil, errors.New("database connection failed")).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "internal_error", response["error"])

		mockUseCase.AssertExpectations(t)
	})
}

func TestIssueTokenRequest_Validate(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		req := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Error_MissingClientID", func(t *testing.T) {
		req := IssueTokenRequest{
			ClientID:     "",
			ClientSecret: "test_secret_123",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_MissingClientSecret", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		req := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankClientID", func(t *testing.T) {
		req := IssueTokenRequest{
			ClientID:     "   ",
			ClientSecret: "test_secret_123",
		}

		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Error_BlankClientSecret", func(t *testing.T) {
		clientID := uuid.Must(uuid.NewV7())
		req := IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "   ",
		}

		err := req.Validate()
		assert.Error(t, err)
	})
}
