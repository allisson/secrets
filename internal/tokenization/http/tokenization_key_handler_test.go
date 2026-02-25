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
	"github.com/stretchr/testify/mock"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
	"github.com/allisson/secrets/internal/tokenization/http/dto"
	"github.com/allisson/secrets/internal/tokenization/usecase/mocks"
)

// setupTestKeyHandler creates a test handler with mocked dependencies.
func setupTestKeyHandler(t *testing.T) (*TokenizationKeyHandler, *mocks.MockTokenizationKeyUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockKeyUseCase := mocks.NewMockTokenizationKeyUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewTokenizationKeyHandler(mockKeyUseCase, logger)

	return handler, mockKeyUseCase
}

func TestTokenizationKeyHandler_CreateHandler(t *testing.T) {
	t.Run("Success_CreateKeyWithUUID", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		keyID := uuid.Must(uuid.NewV7())
		request := dto.CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		expectedKey := &tokenizationDomain.TokenizationKey{
			ID:              keyID,
			Name:            "test-key",
			Version:         1,
			FormatType:      tokenizationDomain.FormatUUID,
			IsDeterministic: false,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       time.Now().UTC(),
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, "test-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
			Return(expectedKey, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TokenizationKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, keyID.String(), response.ID)
		assert.Equal(t, "test-key", response.Name)
		assert.Equal(t, uint(1), response.Version)
		assert.Equal(t, "uuid", response.FormatType)
		assert.False(t, response.IsDeterministic)
	})

	t.Run("Success_CreateKeyWithLuhnPreserving", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		keyID := uuid.Must(uuid.NewV7())
		request := dto.CreateTokenizationKeyRequest{
			Name:            "cc-tokenizer",
			FormatType:      "luhn-preserving",
			IsDeterministic: true,
			Algorithm:       "chacha20-poly1305",
		}

		expectedKey := &tokenizationDomain.TokenizationKey{
			ID:              keyID,
			Name:            "cc-tokenizer",
			Version:         1,
			FormatType:      tokenizationDomain.FormatLuhnPreserving,
			IsDeterministic: true,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       time.Now().UTC(),
		}

		mockUseCase.EXPECT().
			Create(
				mock.Anything,
				"cc-tokenizer",
				tokenizationDomain.FormatLuhnPreserving,
				true,
				cryptoDomain.ChaCha20,
			).
			Return(expectedKey, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TokenizationKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, keyID.String(), response.ID)
		assert.Equal(t, "cc-tokenizer", response.Name)
		assert.Equal(t, "luhn-preserving", response.FormatType)
		assert.True(t, response.IsDeterministic)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		request := dto.CreateTokenizationKeyRequest{
			Name:            "",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidFormatType", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		request := dto.CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "invalid-format",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_InvalidAlgorithm", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		request := dto.CreateTokenizationKeyRequest{
			Name:            "test-key",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "invalid-alg",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})

	t.Run("Error_KeyAlreadyExists", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		request := dto.CreateTokenizationKeyRequest{
			Name:            "existing-key",
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		mockUseCase.EXPECT().
			Create(mock.Anything, "existing-key", tokenizationDomain.FormatUUID, false, cryptoDomain.AESGCM).
			Return(nil, tokenizationDomain.ErrTokenizationKeyAlreadyExists).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys", request)

		handler.CreateHandler(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestTokenizationKeyHandler_RotateHandler(t *testing.T) {
	t.Run("Success_RotateKey", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		keyID := uuid.Must(uuid.NewV7())
		request := dto.RotateTokenizationKeyRequest{
			FormatType:      "numeric",
			IsDeterministic: true,
			Algorithm:       "aes-gcm",
		}

		expectedKey := &tokenizationDomain.TokenizationKey{
			ID:              keyID,
			Name:            "existing-key",
			Version:         2,
			FormatType:      tokenizationDomain.FormatNumeric,
			IsDeterministic: true,
			DekID:           uuid.Must(uuid.NewV7()),
			CreatedAt:       time.Now().UTC(),
		}

		mockUseCase.EXPECT().
			Rotate(
				mock.Anything,
				"existing-key",
				tokenizationDomain.FormatNumeric,
				true,
				cryptoDomain.AESGCM,
			).
			Return(expectedKey, nil).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/existing-key/rotate", request)
		c.Params = gin.Params{{Key: "name", Value: "existing-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.TokenizationKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, keyID.String(), response.ID)
		assert.Equal(t, "existing-key", response.Name)
		assert.Equal(t, uint(2), response.Version)
		assert.Equal(t, "numeric", response.FormatType)
		assert.True(t, response.IsDeterministic)
	})

	t.Run("Error_MissingKeyNameInURL", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		request := dto.RotateTokenizationKeyRequest{
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys//rotate", request)
		c.Params = gin.Params{{Key: "name", Value: ""}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/test-key/rotate", nil)
		c.Request.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))
		c.Params = gin.Params{{Key: "name", Value: "test-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_KeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		request := dto.RotateTokenizationKeyRequest{
			FormatType:      "uuid",
			IsDeterministic: false,
			Algorithm:       "aes-gcm",
		}

		mockUseCase.EXPECT().
			Rotate(
				mock.Anything,
				"nonexistent-key",
				tokenizationDomain.FormatUUID,
				false,
				cryptoDomain.AESGCM,
			).
			Return(nil, tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodPost, "/v1/tokenization/keys/nonexistent-key/rotate", request)
		c.Params = gin.Params{{Key: "name", Value: "nonexistent-key"}}

		handler.RotateHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTokenizationKeyHandler_DeleteHandler(t *testing.T) {
	t.Run("Success_DeleteKey", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		keyID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Delete(mock.Anything, keyID).
			Return(nil).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/tokenization/keys/"+keyID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: keyID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		c, w := createTestContext(http.MethodDelete, "/v1/tokenization/keys/invalid-uuid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
		assert.Contains(t, response["message"], "invalid key ID format")
	})

	t.Run("Error_KeyNotFound", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		keyID := uuid.Must(uuid.NewV7())

		mockUseCase.EXPECT().
			Delete(mock.Anything, keyID).
			Return(tokenizationDomain.ErrTokenizationKeyNotFound).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/tokenization/keys/"+keyID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: keyID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		keyID := uuid.Must(uuid.NewV7())
		dbError := errors.New("database error")

		mockUseCase.EXPECT().
			Delete(mock.Anything, keyID).
			Return(dbError).
			Once()

		c, w := createTestContext(http.MethodDelete, "/v1/tokenization/keys/"+keyID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: keyID.String()}}

		handler.DeleteHandler(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestTokenizationKeyHandler_ListHandler(t *testing.T) {
	t.Run("Success_ListTokenizationKeys", func(t *testing.T) {
		handler, mockUseCase := setupTestKeyHandler(t)

		now := time.Now().UTC()
		expectedKeys := []*tokenizationDomain.TokenizationKey{
			{
				ID:        uuid.Must(uuid.NewV7()),
				Name:      "tok-key-1",
				Version:   1,
				CreatedAt: now,
			},
		}

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 100).
			Return(expectedKeys, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/tokenization/keys?offset=0&limit=100", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListTokenizationKeysResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Items, 1)
		assert.Equal(t, "tok-key-1", response.Items[0].Name)
	})

	t.Run("Error_InvalidPaginationParams", func(t *testing.T) {
		handler, _ := setupTestKeyHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/tokenization/keys?offset=invalid", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "validation_error", response["error"])
	})
}
