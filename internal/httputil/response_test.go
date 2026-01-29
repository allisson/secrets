package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		body         interface{}
		statusCode   int
		expectedBody string
	}{
		{
			name:         "success response",
			body:         map[string]string{"status": "ok"},
			statusCode:   http.StatusOK,
			expectedBody: `{"status":"ok"}`,
		},
		{
			name:         "error response",
			body:         map[string]string{"error": "something went wrong"},
			statusCode:   http.StatusInternalServerError,
			expectedBody: `{"error":"something went wrong"}`,
		},
		{
			name: "complex object",
			body: map[string]interface{}{
				"id":   1,
				"name": "Test",
				"data": map[string]string{"key": "value"},
			},
			statusCode:   http.StatusOK,
			expectedBody: `{"data":{"key":"value"},"id":1,"name":"Test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			MakeJSONResponse(w, tt.statusCode, tt.body)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}
