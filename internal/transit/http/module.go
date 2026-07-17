package http

import (
	"github.com/gin-gonic/gin"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	nethttp "github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/metrics"
)

// Module registers the transit-encryption routes and owns the collaborators
// those routes need (authorizer + business metrics), bound at construction.
type Module struct {
	keyHandler    *TransitKeyHandler
	cryptoHandler *CryptoHandler
	authz         *authHTTP.Authorizer
	bm            metrics.BusinessMetrics
}

// NewModule creates a transit route module.
func NewModule(
	keyHandler *TransitKeyHandler,
	cryptoHandler *CryptoHandler,
	authz *authHTTP.Authorizer,
	bm metrics.BusinessMetrics,
) *Module {
	return &Module{keyHandler: keyHandler, cryptoHandler: cryptoHandler, authz: authz, bm: bm}
}

// Register configures the v1 transit-encryption-related endpoints on the given group.
func (m *Module) Register(v1 *gin.RouterGroup, mw nethttp.RouteMiddlewares) {
	transit := v1.Group("/transit")
	transit.Use(mw.Auth)
	if mw.RateLimit != nil {
		transit.Use(mw.RateLimit)
	}
	{
		keys := transit.Group("/keys")
		{
			// List transit keys
			keys.GET("",
				m.authz.Require(authDomain.ReadCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_key_list"),
				m.keyHandler.ListHandler,
			)

			// Get individual transit key
			keys.GET("/:name",
				m.authz.Require(authDomain.ReadCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_key_get"),
				m.keyHandler.GetHandler,
			)

			// Create new transit key
			keys.POST("",
				m.authz.Require(authDomain.WriteCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_key_create"),
				m.keyHandler.CreateHandler,
			)

			// Rotate transit key to new version
			keys.POST("/:name/rotate",
				m.authz.Require(authDomain.RotateCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_key_rotate"),
				m.keyHandler.RotateHandler,
			)

			// Delete transit key
			keys.DELETE("/:name",
				m.authz.Require(authDomain.DeleteCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_key_delete"),
				m.keyHandler.DeleteHandler,
			)

			// Encrypt plaintext with transit key
			keys.POST("/:name/encrypt",
				m.authz.Require(authDomain.EncryptCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_encrypt"),
				m.cryptoHandler.EncryptHandler,
			)

			// Decrypt ciphertext with transit key
			keys.POST("/:name/decrypt",
				m.authz.Require(authDomain.DecryptCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "transit", "transit_decrypt"),
				m.cryptoHandler.DecryptHandler,
			)
		}
	}
}
