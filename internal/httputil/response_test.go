package httputil_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

func TestHandleErrorGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		err                error
		expectedStatus     int
		expectedErrCode    string
		expectedErrMessage string
	}{
		{
			name:           "nil error",
			err:            nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:               "not found error",
			err:                apperrors.ErrNotFound,
			expectedStatus:     http.StatusNotFound,
			expectedErrCode:    "not_found",
			expectedErrMessage: "The requested resource was not found",
		},
		{
			name:               "conflict error",
			err:                apperrors.ErrConflict,
			expectedStatus:     http.StatusConflict,
			expectedErrCode:    "conflict",
			expectedErrMessage: "A conflict occurred with existing data",
		},
		{
			name:               "invalid input error",
			err:                errors.Join(apperrors.ErrInvalidInput, errors.New("custom detail")),
			expectedStatus:     http.StatusUnprocessableEntity,
			expectedErrCode:    "invalid_input",
			expectedErrMessage: "invalid input\ncustom detail",
		},
		{
			name:               "unauthorized error",
			err:                apperrors.ErrUnauthorized,
			expectedStatus:     http.StatusUnauthorized,
			expectedErrCode:    "unauthorized",
			expectedErrMessage: "Authentication is required",
		},
		{
			name:               "locked error",
			err:                apperrors.ErrLocked,
			expectedStatus:     http.StatusLocked,
			expectedErrCode:    "client_locked",
			expectedErrMessage: "Account is locked due to too many failed authentication attempts",
		},
		{
			name:               "forbidden error",
			err:                apperrors.ErrForbidden,
			expectedStatus:     http.StatusForbidden,
			expectedErrCode:    "forbidden",
			expectedErrMessage: "You don't have permission to access this resource",
		},
		{
			name:               "internal error",
			err:                apperrors.ErrInternal,
			expectedStatus:     http.StatusInternalServerError,
			expectedErrCode:    "internal_error",
			expectedErrMessage: "An internal error occurred",
		},
		{
			name:               "unknown error",
			err:                errors.New("something went wrong"),
			expectedStatus:     http.StatusInternalServerError,
			expectedErrCode:    "internal_error",
			expectedErrMessage: "An internal error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			httputil.HandleErrorGin(c, tt.err, logger)

			if tt.err != nil {
				assert.Equal(t, tt.expectedStatus, w.Code)
				assert.True(t, c.IsAborted())

				var resp httputil.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedErrCode, resp.Error)
				assert.Equal(t, tt.expectedErrMessage, resp.Message)

				// Verify logging
				assert.Contains(t, buf.String(), tt.expectedErrCode)
				assert.Contains(t, buf.String(), "request failed")
			}
		})
	}
}

func TestHandleBadRequestGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	errMsg := "bad json"
	err := errors.New(errMsg)
	httputil.HandleBadRequestGin(c, err, logger)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.True(t, c.IsAborted())

	var resp httputil.ErrorResponse
	errJson := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, errJson)
	assert.Equal(t, "bad_request", resp.Error)
	assert.Equal(t, errMsg, resp.Message)

	assert.Contains(t, buf.String(), "bad request")
	assert.Contains(t, buf.String(), errMsg)
}

func TestHandleValidationErrorGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	errMsg := "validation failed"
	err := errors.New(errMsg)
	httputil.HandleValidationErrorGin(c, err, logger)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.True(t, c.IsAborted())

	var resp httputil.ErrorResponse
	errJson := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, errJson)
	assert.Equal(t, "invalid_input", resp.Error)
	assert.Equal(t, errMsg, resp.Message)

	assert.Contains(t, buf.String(), "validation failed")
	assert.Contains(t, buf.String(), errMsg)
}
