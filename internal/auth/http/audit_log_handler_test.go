package http

import (
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

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/auth/http/dto"
	"github.com/allisson/secrets/internal/auth/usecase/mocks"
)

// setupTestAuditLogHandler creates a test handler with mocked dependencies.
func setupTestAuditLogHandler(t *testing.T) (*AuditLogHandler, *mocks.MockAuditLogUseCase) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	mockAuditLogUseCase := mocks.NewMockAuditLogUseCase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewAuditLogHandler(mockAuditLogUseCase, logger)

	return handler, mockAuditLogUseCase
}

func TestAuditLogHandler_ListHandler(t *testing.T) {
	t.Run("Success_DefaultPagination", func(t *testing.T) {
		handler, mockUseCase := setupTestAuditLogHandler(t)

		id1 := uuid.Must(uuid.NewV7())
		id2 := uuid.Must(uuid.NewV7())
		requestID1 := uuid.Must(uuid.NewV7())
		requestID2 := uuid.Must(uuid.NewV7())
		clientID1 := uuid.Must(uuid.NewV7())
		clientID2 := uuid.Must(uuid.NewV7())
		now := time.Now().UTC()

		expectedAuditLogs := []*authDomain.AuditLog{
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

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 50).
			Return(expectedAuditLogs, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListAuditLogsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.AuditLogs, 2)
		assert.Equal(t, id1.String(), response.AuditLogs[0].ID)
		assert.Equal(t, requestID1.String(), response.AuditLogs[0].RequestID)
		assert.Equal(t, clientID1.String(), response.AuditLogs[0].ClientID)
		assert.Equal(t, string(authDomain.ReadCapability), response.AuditLogs[0].Capability)
		assert.Equal(t, "/v1/secrets/test", response.AuditLogs[0].Path)
		assert.NotNil(t, response.AuditLogs[0].Metadata)
		assert.Equal(t, id2.String(), response.AuditLogs[1].ID)
	})

	t.Run("Success_CustomPagination", func(t *testing.T) {
		handler, mockUseCase := setupTestAuditLogHandler(t)

		expectedAuditLogs := []*authDomain.AuditLog{}

		mockUseCase.EXPECT().
			List(mock.Anything, 10, 25).
			Return(expectedAuditLogs, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?offset=10&limit=25", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListAuditLogsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.AuditLogs, 0)
	})

	t.Run("Success_MaxLimit", func(t *testing.T) {
		handler, mockUseCase := setupTestAuditLogHandler(t)

		expectedAuditLogs := []*authDomain.AuditLog{}

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 100).
			Return(expectedAuditLogs, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?limit=100", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListAuditLogsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.AuditLogs, 0)
	})

	t.Run("Success_EmptyResults", func(t *testing.T) {
		handler, mockUseCase := setupTestAuditLogHandler(t)

		expectedAuditLogs := []*authDomain.AuditLog{}

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 50).
			Return(expectedAuditLogs, nil).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.ListAuditLogsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.AuditLogs, 0)
	})

	t.Run("Error_InvalidOffset_Negative", func(t *testing.T) {
		handler, _ := setupTestAuditLogHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?offset=-1", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "validation_error")
	})

	t.Run("Error_InvalidOffset_NotNumber", func(t *testing.T) {
		handler, _ := setupTestAuditLogHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?offset=abc", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "validation_error")
	})

	t.Run("Error_InvalidLimit_TooLow", func(t *testing.T) {
		handler, _ := setupTestAuditLogHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?limit=0", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "validation_error")
	})

	t.Run("Error_InvalidLimit_TooHigh", func(t *testing.T) {
		handler, _ := setupTestAuditLogHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?limit=101", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "validation_error")
	})

	t.Run("Error_InvalidLimit_NotNumber", func(t *testing.T) {
		handler, _ := setupTestAuditLogHandler(t)

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs?limit=xyz", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "validation_error")
	})

	t.Run("Error_UseCaseError", func(t *testing.T) {
		handler, mockUseCase := setupTestAuditLogHandler(t)

		mockUseCase.EXPECT().
			List(mock.Anything, 0, 50).
			Return(nil, errors.New("database error")).
			Once()

		c, w := createTestContext(http.MethodGet, "/v1/audit-logs", nil)

		handler.ListHandler(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "internal_error")
	})
}
