//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_Health_BasicChecks validates infrastructure health and readiness endpoints.
// Tests health check and database connectivity verification against both PostgreSQL and MySQL.
func TestIntegration_Health_BasicChecks(t *testing.T) {
	// Skip if short mode (integration tests can be slow)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name     string
		dbDriver string
	}{
		{"PostgreSQL", "postgres"},
		{"MySQL", "mysql"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			ctx := setupIntegrationTest(t, tc.dbDriver)
			defer teardownIntegrationTest(t, ctx)

			// [1/2] Test GET /health - Health check endpoint
			t.Run("01_HealthCheck", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/health", nil, false)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response map[string]string
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "healthy", response["status"])
			})

			// [2/2] Test GET /ready - Readiness check endpoint
			t.Run("02_ReadinessCheck", func(t *testing.T) {
				resp, body := ctx.makeRequest(t, http.MethodGet, "/ready", nil, false)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "ready", response["status"])
			})

			t.Logf("All 2 health endpoint tests passed for %s", tc.dbDriver)
		})
	}
}
