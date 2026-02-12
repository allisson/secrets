package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/auth/http/dto"
	httpMocks "github.com/allisson/secrets/internal/auth/http/mocks"
)

// setupTokenTestHandler creates a test token handler with mocked dependencies.
func setupTokenTestHandler(t *testing.T) (*TokenHandler, *httpMocks.MockTokenUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockTokenUseCase := &httpMocks.MockTokenUseCase{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewTokenHandler(mockTokenUseCase, logger)

	return handler, mockTokenUseCase
}

func TestTokenHandler_IssueTokenHandler(t *testing.T) {
	t.Run("Success_ValidCredentials", func(t *testing.T) {
		handler, mockUseCase := setupTokenTestHandler(t)

		clientID := uuid.Must(uuid.NewV7())
		plainToken := "tok_1234567890abcdef"
		expiresAt := time.Now().UTC().Add(1 * time.Hour)

		request := dto.IssueTokenRequest{
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

		mockUseCase.On("Issue", nil, expectedInput).
			Return(expectedOutput, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/token", request)

		handler.IssueTokenHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.IssueTokenResponse
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

		request := dto.IssueTokenRequest{
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
		request := dto.IssueTokenRequest{
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

		request := dto.IssueTokenRequest{
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
		request := dto.IssueTokenRequest{
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
		request := dto.IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "wrong_secret",
		}

		mockUseCase.On("Issue", nil, nil).
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
		request := dto.IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		mockUseCase.On("Issue", nil, nil).
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
		request := dto.IssueTokenRequest{
			ClientID:     clientID.String(),
			ClientSecret: "test_secret_123",
		}

		mockUseCase.On("Issue", nil, nil).
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
