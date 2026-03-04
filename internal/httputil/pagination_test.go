package httputil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/allisson/secrets/internal/httputil"
)

func TestParseUUIDCursorPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID := uuid.New()

	tests := []struct {
		name           string
		url            string
		cursorParam    string
		expectedCursor *uuid.UUID
		expectedLimit  int
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "default values no cursor",
			url:            "/",
			cursorParam:    "after_id",
			expectedCursor: nil,
			expectedLimit:  50,
			expectError:    false,
		},
		{
			name:           "valid cursor and limit",
			url:            "/?after_id=" + validUUID.String() + "&limit=20",
			cursorParam:    "after_id",
			expectedCursor: &validUUID,
			expectedLimit:  20,
			expectError:    false,
		},
		{
			name:           "cursor only with default limit",
			url:            "/?after_id=" + validUUID.String(),
			cursorParam:    "after_id",
			expectedCursor: &validUUID,
			expectedLimit:  50,
			expectError:    false,
		},
		{
			name:           "max limit",
			url:            "/?limit=1000",
			cursorParam:    "after_id",
			expectedCursor: nil,
			expectedLimit:  1000,
			expectError:    false,
		},
		{
			name:           "limit exceeds max gets clamped",
			url:            "/?limit=5000",
			cursorParam:    "after_id",
			expectedCursor: nil,
			expectedLimit:  1000,
			expectError:    false,
		},
		{
			name:        "invalid uuid cursor",
			url:         "/?after_id=invalid-uuid",
			cursorParam: "after_id",
			expectError: true,
			errorMsg:    "invalid after_id parameter: must be a valid UUID",
		},
		{
			name:        "limit zero",
			url:         "/?limit=0",
			cursorParam: "after_id",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be a positive integer",
		},
		{
			name:        "limit negative",
			url:         "/?limit=-1",
			cursorParam: "after_id",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be a positive integer",
		},
		{
			name:        "limit not an integer",
			url:         "/?limit=abc",
			cursorParam: "after_id",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			c.Request = req

			cursor, limit, err := httputil.ParseUUIDCursorPagination(c, tt.cursorParam)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Nil(t, cursor)
				assert.Equal(t, 0, limit)
			} else {
				assert.NoError(t, err)
				if tt.expectedCursor == nil {
					assert.Nil(t, cursor)
				} else {
					assert.NotNil(t, cursor)
					assert.Equal(t, *tt.expectedCursor, *cursor)
				}
				assert.Equal(t, tt.expectedLimit, limit)
			}
		})
	}
}

func TestParseStringCursorPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		url            string
		cursorParam    string
		expectedCursor *string
		expectedLimit  int
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "default values no cursor",
			url:            "/",
			cursorParam:    "after_path",
			expectedCursor: nil,
			expectedLimit:  50,
			expectError:    false,
		},
		{
			name:           "valid cursor and limit",
			url:            "/?after_path=production/db/password&limit=20",
			cursorParam:    "after_path",
			expectedCursor: stringPtr("production/db/password"),
			expectedLimit:  20,
			expectError:    false,
		},
		{
			name:           "cursor only with default limit",
			url:            "/?after_name=my-key&limit=50",
			cursorParam:    "after_name",
			expectedCursor: stringPtr("my-key"),
			expectedLimit:  50,
			expectError:    false,
		},
		{
			name:           "max limit",
			url:            "/?limit=1000",
			cursorParam:    "after_path",
			expectedCursor: nil,
			expectedLimit:  1000,
			expectError:    false,
		},
		{
			name:           "limit exceeds max gets clamped",
			url:            "/?limit=5000",
			cursorParam:    "after_path",
			expectedCursor: nil,
			expectedLimit:  1000,
			expectError:    false,
		},
		{
			name:           "cursor with special characters",
			url:            "/?after_path=prod%2Fdb%2Fpass&limit=10",
			cursorParam:    "after_path",
			expectedCursor: stringPtr("prod/db/pass"),
			expectedLimit:  10,
			expectError:    false,
		},
		{
			name:        "limit zero",
			url:         "/?limit=0",
			cursorParam: "after_path",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be a positive integer",
		},
		{
			name:        "limit negative",
			url:         "/?limit=-1",
			cursorParam: "after_path",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be a positive integer",
		},
		{
			name:        "limit not an integer",
			url:         "/?limit=xyz",
			cursorParam: "after_path",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			c.Request = req

			cursor, limit, err := httputil.ParseStringCursorPagination(c, tt.cursorParam)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Nil(t, cursor)
				assert.Equal(t, 0, limit)
			} else {
				assert.NoError(t, err)
				if tt.expectedCursor == nil {
					assert.Nil(t, cursor)
				} else {
					assert.NotNil(t, cursor)
					assert.Equal(t, *tt.expectedCursor, *cursor)
				}
				assert.Equal(t, tt.expectedLimit, limit)
			}
		})
	}
}

// stringPtr returns a pointer to a string value
func stringPtr(s string) *string {
	return &s
}
