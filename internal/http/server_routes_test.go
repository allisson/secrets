package http_test

import (
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authHTTP "github.com/allisson/secrets/internal/auth/http"
	"github.com/allisson/secrets/internal/config"
	nethttp "github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/metrics"
	secretsHTTP "github.com/allisson/secrets/internal/secrets/http"
	tokenizationHTTP "github.com/allisson/secrets/internal/tokenization/http"
	transitHTTP "github.com/allisson/secrets/internal/transit/http"
)

// goldenRoutes is the complete method+path inventory the router must expose.
// It is the guardrail for behavior-preserving route refactors: any route that is
// dropped, renamed, or moved to a different verb fails TestRouteManifest.
var goldenRoutes = []string{
	"DELETE /v1/clients/:id",
	"DELETE /v1/clients/:id/tokens",
	"DELETE /v1/secrets/*path",
	"DELETE /v1/token",
	"DELETE /v1/tokenization/keys/:name",
	"DELETE /v1/transit/keys/:name",
	"GET /health",
	"GET /ready",
	"GET /v1/audit-logs",
	"GET /v1/clients",
	"GET /v1/clients/:id",
	"GET /v1/secrets",
	"GET /v1/secrets/*path",
	"GET /v1/tokenization/keys",
	"GET /v1/tokenization/keys/:name",
	"GET /v1/transit/keys",
	"GET /v1/transit/keys/:name",
	"POST /v1/clients",
	"POST /v1/clients/:id/rotate-secret",
	"POST /v1/clients/:id/unlock",
	"POST /v1/secrets/*path",
	"POST /v1/token",
	"POST /v1/tokenization/detokenize",
	"POST /v1/tokenization/detokenize-batch",
	"POST /v1/tokenization/keys",
	"POST /v1/tokenization/keys/:name/rotate",
	"POST /v1/tokenization/keys/:name/tokenize",
	"POST /v1/tokenization/keys/:name/tokenize-batch",
	"POST /v1/tokenization/revoke",
	"POST /v1/tokenization/validate",
	"POST /v1/transit/keys",
	"POST /v1/transit/keys/:name/decrypt",
	"POST /v1/transit/keys/:name/encrypt",
	"POST /v1/transit/keys/:name/rotate",
	"PUT /v1/clients/:id",
}

// buildManifestRouter constructs the full router with nil-backed handlers.
// Route registration only binds handler method values; it never invokes them,
// so nil handler pointers are safe here and let the manifest test avoid mocks.
func buildManifestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := nethttp.NewServer(nil, "localhost", 0, time.Second, time.Second, time.Second, logger)

	bm := metrics.NewNopBusinessMetrics()
	authz := authHTTP.NewAuthorizer(nil, logger)
	registrars := []nethttp.RouteRegistrar{
		authHTTP.NewModule(nil, nil, nil, authz, bm, nil),
		secretsHTTP.NewModule(nil, authz, bm),
		transitHTTP.NewModule(nil, nil, authz, bm),
		tokenizationHTTP.NewModule(nil, nil, authz, bm),
	}
	srv.SetupRouter(&config.Config{}, registrars, nethttp.RouteMiddlewares{}, nil, "")

	eng, ok := srv.GetHandler().(*gin.Engine)
	require.True(t, ok, "expected *gin.Engine from GetHandler")
	return eng
}

// actualRoutes returns the sorted "METHOD path" inventory of the router.
func actualRoutes(eng *gin.Engine) []string {
	routes := eng.Routes()
	got := make([]string, 0, len(routes))
	for _, r := range routes {
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)
	return got
}

// TestRouteManifest asserts the registered route set matches the golden inventory.
func TestRouteManifest(t *testing.T) {
	got := actualRoutes(buildManifestRouter(t))
	assert.Equal(t, goldenRoutes, got)
}
