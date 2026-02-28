package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apperrors "github.com/allisson/secrets/internal/errors"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	"github.com/allisson/secrets/internal/secrets/http/dto"
	"github.com/allisson/secrets/internal/secrets/usecase/mocks"
)

// setupTestHandler creates a test handler with mocked dependencies.
func setupTestHandler(t *testing.T) (*SecretHandler, *mocks.MockSecretUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockSecretUseCase := mocks.NewMockSecretUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewSecretHandler(mockSecretUseCase, nil, logger)

	return handler, mockSecretUseCase
}

func TestSecretHandler_CreateOrUpdateHandler(t *testing.T) {
	t.Parallel()
	t.Run("Success_ValidRequest", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		secretID := uuid.Must(uuid.NewV7())
		path := "database/password"
		value := []byte("super-secret-password")
		now := time.Now().UTC()

		request := dto.CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString(value),
		}

		expectedSecret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      path,
			Version:   1,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			CreateOrUpdate(mock.Anything, path, value).
			Return(expectedSecret, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/secrets/"+path, request)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.SecretResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, secretID.String(), response.ID)
		assert.Equal(t, path, response.Path)
		assert.Equal(t, uint(1), response.Version)
		assert.Empty(t, response.Value) // Value should not be included in create response
	})

	t.Run("Success_NestedPath", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		secretID := uuid.Must(uuid.NewV7())
		path := "my/nested/secret/path"
		value := []byte("nested-value")
		now := time.Now().UTC()

		request := dto.CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString(value),
		}

		expectedSecret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      path,
			Version:   2,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			CreateOrUpdate(mock.Anything, path, value).
			Return(expectedSecret, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/secrets/"+path, request)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.SecretResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, path, response.Path)
		assert.Equal(t, uint(2), response.Version)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/secrets/database/password", nil)
		c.Params = gin.Params{{Key: "path", Value: "/database/password"}}
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})

	t.Run("Error_EmptyValue", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		request := dto.CreateOrUpdateSecretRequest{
			Value: "",
		}

		c, w := createTestContext(http.MethodPost, "/v1/secrets/database/password", request)
		c.Params = gin.Params{{Key: "path", Value: "/database/password"}}

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidBase64", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		request := dto.CreateOrUpdateSecretRequest{
			Value: "not-valid-base64!@#$%",
		}

		c, w := createTestContext(http.MethodPost, "/v1/secrets/database/password", request)
		c.Params = gin.Params{{Key: "path", Value: "/database/password"}}

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_EmptyPath", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		request := dto.CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString([]byte("value")),
		}

		c, w := createTestContext(http.MethodPost, "/v1/secrets/", request)
		c.Params = gin.Params{{Key: "path", Value: "/"}}

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
		assert.Contains(t, response["message"], "path cannot be empty")
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		path := "database/password"
		value := []byte("value")

		request := dto.CreateOrUpdateSecretRequest{
			Value: base64.StdEncoding.EncodeToString(value),
		}

		mockUseCase.EXPECT().
			CreateOrUpdate(mock.Anything, path, value).
			Return(nil, fmt.Errorf("use case error")).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/secrets/"+path, request)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.CreateOrUpdateHandler(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "internal_error", response["error"])
	})
}

func TestSecretHandler_GetHandler(t *testing.T) {
	t.Parallel()
	t.Run("Success_GetLatestVersion", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		secretID := uuid.Must(uuid.NewV7())
		path := "database/password"
		plaintext := []byte("super-secret-password")
		expectedPlaintext := []byte("super-secret-password") // Copy for comparison after zeroing
		now := time.Now().UTC()

		expectedSecret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      path,
			Version:   1,
			Plaintext: plaintext,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			Get(mock.Anything, path).
			Return(expectedSecret, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/secrets/"+path, nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.GetHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SecretResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, secretID.String(), response.ID)
		assert.Equal(t, path, response.Path)
		assert.Equal(t, uint(1), response.Version)
		assert.Equal(t, base64.StdEncoding.EncodeToString(expectedPlaintext), response.Value)
	})

	t.Run("Success_GetSpecificVersion", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		secretID := uuid.Must(uuid.NewV7())
		path := "database/password"
		plaintext := []byte("old-password")
		expectedPlaintext := []byte("old-password") // Copy for comparison after zeroing
		version := uint(2)
		now := time.Now().UTC()

		expectedSecret := &secretsDomain.Secret{
			ID:        secretID,
			Path:      path,
			Version:   version,
			Plaintext: plaintext,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			GetByVersion(mock.Anything, path, version).
			Return(expectedSecret, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/secrets/"+path+"?version=2", nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}
		c.Request.URL.RawQuery = "version=2"

		handler.GetHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SecretResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, secretID.String(), response.ID)
		assert.Equal(t, path, response.Path)
		assert.Equal(t, version, response.Version)
		assert.Equal(t, base64.StdEncoding.EncodeToString(expectedPlaintext), response.Value)
	})

	t.Run("Error_InvalidVersionParameter", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		path := "database/password"

		c, w := createTestContext(http.MethodGet, "/v1/secrets/"+path+"?version=invalid", nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}
		c.Request.URL.RawQuery = "version=invalid"

		handler.GetHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
		assert.Contains(t, response["message"], "invalid version parameter")
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		path := "nonexistent/secret"

		mockUseCase.EXPECT().
			Get(mock.Anything, path).
			Return(nil, apperrors.ErrNotFound).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/secrets/"+path, nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.GetHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response["error"])
	})

	t.Run("Error_EmptyPath", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/secrets/", nil)
		c.Params = gin.Params{{Key: "path", Value: "/"}}

		handler.GetHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
		assert.Contains(t, response["message"], "path cannot be empty")
	})
}

func TestSecretHandler_DeleteHandler(t *testing.T) {
	t.Parallel()
	t.Run("Success_DeleteSecret", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		path := "database/password"

		mockUseCase.EXPECT().
			Delete(mock.Anything, path).
			Return(nil).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/secrets/"+path, nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("Success_NestedPath", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		path := "my/nested/secret/path"

		mockUseCase.EXPECT().
			Delete(mock.Anything, path).
			Return(nil).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/secrets/"+path, nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		path := "nonexistent/secret"

		mockUseCase.EXPECT().
			Delete(mock.Anything, path).
			Return(apperrors.ErrNotFound).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/secrets/"+path, nil)
		c.Params = gin.Params{{Key: "path", Value: "/" + path}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response["error"])
	})

	t.Run("Error_EmptyPath", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodDelete, "/v1/secrets/", nil)
		c.Params = gin.Params{{Key: "path", Value: "/"}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
		assert.Contains(t, response["message"], "path cannot be empty")
	})
}

func TestSecretHandler_ListHandler(t *testing.T) {
	t.Parallel()
	t.Run("Success_ListSecrets", func(t *testing.T) {
		t.Parallel()
		handler, mockUseCase := setupTestHandler(t)

		now := time.Now().UTC()
		expectedSecrets := []*secretsDomain.Secret{
			{
				ID:        uuid.Must(uuid.NewV7()),
				Path:      "a/a",
				Version:   1,
				CreatedAt: now,
			},
			{
				ID:        uuid.Must(uuid.NewV7()),
				Path:      "b/b",
				Version:   2,
				CreatedAt: now,
			},
		}

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 100).
			Return(expectedSecrets, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/secrets?offset=0&limit=100", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListSecretsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Data, 2)
		assert.Equal(t, "a/a", response.Data[0].Path)
		assert.Equal(t, "b/b", response.Data[1].Path)
	})

	t.Run("Error_InvalidPaginationParams", func(t *testing.T) {
		t.Parallel()
		handler, _ := setupTestHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/secrets?offset=invalid", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})
}
