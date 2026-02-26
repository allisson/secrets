package httputil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/allisson/secrets/internal/httputil"
)

func TestParsePagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		url            string
		expectedOffset int
		expectedLimit  int
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "default values",
			url:            "/",
			expectedOffset: 0,
			expectedLimit:  50,
			expectError:    false,
		},
		{
			name:           "valid custom values",
			url:            "/?offset=10&limit=20",
			expectedOffset: 10,
			expectedLimit:  20,
			expectError:    false,
		},
		{
			name:           "max limit",
			url:            "/?limit=100",
			expectedOffset: 0,
			expectedLimit:  100,
			expectError:    false,
		},
		{
			name:        "offset negative",
			url:         "/?offset=-1",
			expectError: true,
			errorMsg:    "invalid offset parameter: must be a non-negative integer",
		},
		{
			name:        "offset not an integer",
			url:         "/?offset=abc",
			expectError: true,
			errorMsg:    "invalid offset parameter: must be a non-negative integer",
		},
		{
			name:        "limit zero",
			url:         "/?limit=0",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be between 1 and 100",
		},
		{
			name:        "limit exceeds max",
			url:         "/?limit=101",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be between 1 and 100",
		},
		{
			name:        "limit not an integer",
			url:         "/?limit=xyz",
			expectError: true,
			errorMsg:    "invalid limit parameter: must be between 1 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			c.Request = req

			offset, limit, err := httputil.ParsePagination(c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				// Check that values are 0 on error
				assert.Equal(t, 0, offset)
				assert.Equal(t, 0, limit)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOffset, offset)
				assert.Equal(t, tt.expectedLimit, limit)
			}
		})
	}
}
