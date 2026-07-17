package http

import (
	"github.com/gin-gonic/gin"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authHTTP "github.com/allisson/secrets/internal/auth/http"
	nethttp "github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/metrics"
)

// Module registers the tokenization routes and owns the collaborators those
// routes need (authorizer + business metrics), bound at construction.
type Module struct {
	keyHandler          *TokenizationKeyHandler
	tokenizationHandler *TokenizationHandler
	authz               *authHTTP.Authorizer
	bm                  metrics.BusinessMetrics
}

// NewModule creates a tokenization route module.
func NewModule(
	keyHandler *TokenizationKeyHandler,
	tokenizationHandler *TokenizationHandler,
	authz *authHTTP.Authorizer,
	bm metrics.BusinessMetrics,
) *Module {
	return &Module{keyHandler: keyHandler, tokenizationHandler: tokenizationHandler, authz: authz, bm: bm}
}

// Register configures the v1 tokenization-related endpoints on the given group.
func (m *Module) Register(v1 *gin.RouterGroup, mw nethttp.RouteMiddlewares) {
	tokenization := v1.Group("/tokenization")
	tokenization.Use(mw.Auth)
	if mw.RateLimit != nil {
		tokenization.Use(mw.RateLimit)
	}
	{
		keys := tokenization.Group("/keys")
		{
			// List tokenization keys
			keys.GET("",
				m.authz.Require(authDomain.ReadCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenization_key_list"),
				m.keyHandler.ListHandler,
			)

			// Get individual tokenization key
			keys.GET("/:name",
				m.authz.Require(authDomain.ReadCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenization_key_get"),
				m.keyHandler.GetByNameHandler,
			)

			// Create new tokenization key
			keys.POST("",
				m.authz.Require(authDomain.WriteCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenization_key_create"),
				m.keyHandler.CreateHandler,
			)

			// Rotate tokenization key to new version
			keys.POST("/:name/rotate",
				m.authz.Require(authDomain.RotateCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenization_key_rotate"),
				m.keyHandler.RotateHandler,
			)

			// Delete tokenization key
			keys.DELETE("/:name",
				m.authz.Require(authDomain.DeleteCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenization_key_delete"),
				m.keyHandler.DeleteHandler,
			)

			// Tokenize plaintext with tokenization key
			keys.POST("/:name/tokenize",
				m.authz.Require(authDomain.EncryptCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenize"),
				m.tokenizationHandler.TokenizeHandler,
			)

			// Tokenize batch of plaintexts with tokenization key
			keys.POST("/:name/tokenize-batch",
				m.authz.Require(authDomain.EncryptCapability),
				nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenize_batch"),
				m.tokenizationHandler.TokenizeBatchHandler,
			)
		}

		// Detokenize token to retrieve plaintext
		tokenization.POST("/detokenize",
			m.authz.Require(authDomain.DecryptCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "detokenize"),
			m.tokenizationHandler.DetokenizeHandler,
		)

		// Detokenize batch of tokens to retrieve plaintexts
		tokenization.POST("/detokenize-batch",
			m.authz.Require(authDomain.DecryptCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "detokenize_batch"),
			m.tokenizationHandler.DetokenizeBatchHandler,
		)

		// Validate token existence and validity
		tokenization.POST("/validate",
			m.authz.Require(authDomain.ReadCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenize_validate"),
			m.tokenizationHandler.ValidateHandler,
		)

		// Revoke token to prevent further detokenization
		tokenization.POST("/revoke",
			m.authz.Require(authDomain.DeleteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "tokenization", "tokenize_revoke"),
			m.tokenizationHandler.RevokeHandler,
		)
	}
}
