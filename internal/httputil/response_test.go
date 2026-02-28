package httputil_test

import (
	"errors"
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
			expectedStatus: http.StatusOK, // Context status remains unchanged or is 200 by default in test context
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
			expectedErrMessage: "invalid input: custom detail",
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
			name:               "unknown error",
			err:                errors.New("something went wrong"),
			expectedStatus:     http.StatusInternalServerError,
			expectedErrCode:    "internal_error",
			expectedErrMessage: "An internal error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			httputil.HandleErrorGin(c, tt.err, nil)

			if tt.err != nil {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandleBadRequestGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.New("bad json")
	httputil.HandleBadRequestGin(c, err, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleValidationErrorGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.New("validation failed")
	httputil.HandleValidationErrorGin(c, err, nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
