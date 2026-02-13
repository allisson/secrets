// Package http provides HTTP handlers for transit key management and cryptographic operations.
package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

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
