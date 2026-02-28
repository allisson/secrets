package http

import (
	"bytes"
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

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	apperrors "github.com/allisson/secrets/internal/errors"
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
	"github.com/allisson/secrets/internal/transit/http/dto"
	"github.com/allisson/secrets/internal/transit/usecase/mocks"
)

// setupTestTransitKeyHandler creates a test handler with mocked dependencies.
func setupTestTransitKeyHandler(t *testing.T) (*TransitKeyHandler, *mocks.MockTransitKeyUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockTransitKeyUseCase := mocks.NewMockTransitKeyUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewTransitKeyHandler(mockTransitKeyUseCase, nil, logger)

	return handler, mockTransitKeyUseCase
}

func TestTransitKeyHandler_CreateHandler(t *testing.T) {
	t.Run("Success_ValidRequest_AESGCM", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		transitKeyID := uuid.Must(uuid.NewV7())
		dekID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		request := dto.CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "aes-gcm",
		}

		expectedTransitKey := &transitDomain.TransitKey{
			ID:        transitKeyID,
			Name:      "test-key",
			Version:   1,
			DekID:     dekID,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, "test-key", cryptoDomain.AESGCM).
			Return(expectedTransitKey, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TransitKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, transitKeyID.String(), response.ID)
		assert.Equal(t, "test-key", response.Name)
		assert.Equal(t, uint(1), response.Version)
		assert.Equal(t, dekID.String(), response.DekID)
	})

	t.Run("Success_ValidRequest_ChaCha20", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		transitKeyID := uuid.Must(uuid.NewV7())
		dekID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		request := dto.CreateTransitKeyRequest{
			Name:      "test-key-chacha",
			Algorithm: "chacha20-poly1305",
		}

		expectedTransitKey := &transitDomain.TransitKey{
			ID:        transitKeyID,
			Name:      "test-key-chacha",
			Version:   1,
			DekID:     dekID,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, "test-key-chacha", cryptoDomain.ChaCha20).
			Return(expectedTransitKey, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TransitKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, transitKeyID.String(), response.ID)
		assert.Equal(t, "test-key-chacha", response.Name)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})

	t.Run("Error_ValidationFailed_MissingName", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		request := dto.CreateTransitKeyRequest{
			Name:      "",
			Algorithm: "aes-gcm",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_ValidationFailed_InvalidAlgorithm", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		request := dto.CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "invalid-algorithm",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		request := dto.CreateTransitKeyRequest{
			Name:      "test-key",
			Algorithm: "aes-gcm",
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, "test-key", cryptoDomain.AESGCM).
			Return(nil, apperrors.ErrConflict).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestTransitKeyHandler_RotateHandler(t *testing.T) {
	t.Run("Success_ValidRequest", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		transitKeyID := uuid.Must(uuid.NewV7())
		dekID := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		request := dto.RotateTransitKeyRequest{
			Algorithm: "aes-gcm",
		}

		expectedTransitKey := &transitDomain.TransitKey{
			ID:        transitKeyID,
			Name:      "test-key",
			Version:   2,
			DekID:     dekID,
			CreatedAt: now,
		}

		mockUseCase.EXPECT().
			Rotate(mock.Anything, "test-key", cryptoDomain.AESGCM).
			Return(expectedTransitKey, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/rotate", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TransitKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, transitKeyID.String(), response.ID)
		assert.Equal(t, "test-key", response.Name)
		assert.Equal(t, uint(2), response.Version)
	})

	t.Run("Error_EmptyName", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		request := dto.RotateTransitKeyRequest{
			Algorithm: "aes-gcm",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys//rotate", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: ""}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/rotate", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})

	t.Run("Error_ValidationFailed_InvalidAlgorithm", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		request := dto.RotateTransitKeyRequest{
			Algorithm: "invalid",
		}

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/test-key/rotate", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "test-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_TransitKeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		request := dto.RotateTransitKeyRequest{
			Algorithm: "aes-gcm",
		}

		mockUseCase.EXPECT().
			Rotate(mock.Anything, "nonexistent-key", cryptoDomain.AESGCM).
			Return(nil, transitDomain.ErrTransitKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/transit/keys/nonexistent-key/rotate", request)
		c.Params = gin.Params{gin.Param{Key: "name", Value: "nonexistent-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTransitKeyHandler_DeleteHandler(t *testing.T) {
	t.Run("Success_ValidUUID", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		transitKeyID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Delete(mock.Anything, transitKeyID).
			Return(nil).
			Once()

		c, w := createTestContext(http.MethodDelete, fmt.Sprintf("/v1/transit/keys/%s", transitKeyID), nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: transitKeyID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		c, w := createTestContext(http.MethodDelete, "/v1/transit/keys/invalid-uuid", nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "invalid-uuid"}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})

	t.Run("Error_TransitKeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		transitKeyID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Delete(mock.Anything, transitKeyID).
			Return(transitDomain.ErrTransitKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodDelete, fmt.Sprintf("/v1/transit/keys/%s", transitKeyID), nil)
		c.Params = gin.Params{gin.Param{Key: "id", Value: transitKeyID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTransitKeyHandler_ListHandler(t *testing.T) {
	t.Run("Success_ListTransitKeys", func(t *testing.T) {
		handler, mockUseCase := setupTestTransitKeyHandler(t)

		now := time.Now().UTC()
		expectedKeys := []*transitDomain.TransitKey{
			{
				ID:        uuid.Must(uuid.NewV7()),
				Name:      "key-1",
				Version:   1,
				CreatedAt: now,
			},
		}

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 100).
			Return(expectedKeys, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/transit/keys?offset=0&limit=100", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListTransitKeysResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Data, 1)
		assert.Equal(t, "key-1", response.Data[0].Name)
	})

	t.Run("Error_InvalidPaginationParams", func(t *testing.T) {
		handler, _ := setupTestTransitKeyHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/transit/keys?offset=invalid", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "bad_request", response["error"])
	})
}
