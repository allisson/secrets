package http

import (
	"github.com/gin-gonic/gin"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	nethttp "github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/metrics"
)

// Module registers the secret-management routes and owns the collaborators
// those routes need (authorizer + business metrics), bound at construction.
type Module struct {
	handler *SecretHandler
	authz   *authHTTP.Authorizer
	bm      metrics.BusinessMetrics
}

// NewModule creates a secrets route module.
func NewModule(handler *SecretHandler, authz *authHTTP.Authorizer, bm metrics.BusinessMetrics) *Module {
	return &Module{handler: handler, authz: authz, bm: bm}
}

// Register configures the v1 secret-related endpoints on the given group.
func (m *Module) Register(v1 *gin.RouterGroup, mw nethttp.RouteMiddlewares) {
	secrets := v1.Group("/secrets")
	secrets.Use(mw.Auth)
	if mw.RateLimit != nil {
		secrets.Use(mw.RateLimit)
	}
	{
		secrets.GET("",
			m.authz.Require(authDomain.ReadCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "secrets", "secret_list"),
			m.handler.ListHandler,
		)
		secrets.POST("/*path",
			m.authz.Require(authDomain.EncryptCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "secrets", "secret_create_or_update"),
			m.handler.CreateOrUpdateHandler,
		)
		secrets.GET("/*path",
			m.authz.Require(authDomain.DecryptCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "secrets", "secret_get"),
			m.handler.GetHandler,
		)
		secrets.DELETE("/*path",
			m.authz.Require(authDomain.DeleteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "secrets", "secret_delete"),
			m.handler.DeleteHandler,
		)
	}
}
