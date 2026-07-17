package http

import (
	"github.com/gin-gonic/gin"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	nethttp "github.com/allisson/secrets/internal/http"
	"github.com/allisson/secrets/internal/metrics"
)

// Module registers the authentication routes (token, clients, audit logs) and
// owns the collaborators those routes need, bound at construction. It also owns
// the token endpoint's irregular layout: /token skips the shared authentication
// middleware and uses its own IP-based rate limiter.
type Module struct {
	clientHandler   *ClientHandler
	tokenHandler    *TokenHandler
	auditLogHandler *AuditLogHandler
	authz           *Authorizer
	bm              metrics.BusinessMetrics
	tokenRateLimit  gin.HandlerFunc
}

// NewModule creates an auth route module. tokenRateLimit may be nil when
// IP-based rate limiting of the token endpoint is disabled.
func NewModule(
	clientHandler *ClientHandler,
	tokenHandler *TokenHandler,
	auditLogHandler *AuditLogHandler,
	authz *Authorizer,
	bm metrics.BusinessMetrics,
	tokenRateLimit gin.HandlerFunc,
) *Module {
	return &Module{
		clientHandler:   clientHandler,
		tokenHandler:    tokenHandler,
		auditLogHandler: auditLogHandler,
		authz:           authz,
		bm:              bm,
		tokenRateLimit:  tokenRateLimit,
	}
}

// Register configures the v1 authentication-related endpoints on the given group.
func (m *Module) Register(v1 *gin.RouterGroup, mw nethttp.RouteMiddlewares) {
	// Token issuance endpoint (no authentication required, IP-based rate limiting)
	if m.tokenRateLimit != nil {
		v1.POST(
			"/token",
			m.tokenRateLimit,
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "token_issue"),
			m.tokenHandler.IssueTokenHandler,
		)
	} else {
		v1.POST(
			"/token",
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "token_issue"),
			m.tokenHandler.IssueTokenHandler,
		)
	}

	// Token revocation endpoint (requires authentication)
	v1.DELETE(
		"/token",
		mw.Auth,
		nethttp.BusinessMetricsMiddleware(m.bm, "auth", "token_revoke"),
		m.tokenHandler.RevokeTokenHandler,
	)

	// Client management endpoints
	clients := v1.Group("/clients")
	clients.Use(mw.Auth)
	if mw.RateLimit != nil {
		clients.Use(mw.RateLimit)
	}
	{
		clients.POST("",
			m.authz.Require(authDomain.WriteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_create"),
			m.clientHandler.CreateHandler,
		)
		clients.GET("",
			m.authz.Require(authDomain.ReadCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_list"),
			m.clientHandler.ListHandler,
		)
		clients.GET("/:id",
			m.authz.Require(authDomain.ReadCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_get"),
			m.clientHandler.GetHandler,
		)
		clients.PUT("/:id",
			m.authz.Require(authDomain.WriteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_update"),
			m.clientHandler.UpdateHandler,
		)
		clients.DELETE("/:id",
			m.authz.Require(authDomain.DeleteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_delete"),
			m.clientHandler.DeleteHandler,
		)
		clients.POST("/:id/unlock",
			m.authz.Require(authDomain.WriteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_unlock"),
			m.clientHandler.UnlockHandler,
		)
		clients.POST("/:id/rotate-secret",
			m.authz.Require(authDomain.RotateCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_rotate_secret"),
			m.clientHandler.RotateSecretHandler,
		)
		clients.DELETE("/:id/tokens",
			m.authz.Require(authDomain.DeleteCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "client_revoke_tokens"),
			m.clientHandler.RevokeTokensHandler,
		)
	}

	// Audit log endpoints
	auditLogs := v1.Group("/audit-logs")
	auditLogs.Use(mw.Auth)
	if mw.RateLimit != nil {
		auditLogs.Use(mw.RateLimit)
	}
	{
		auditLogs.GET("",
			m.authz.Require(authDomain.ReadCapability),
			nethttp.BusinessMetricsMiddleware(m.bm, "auth", "audit_log_list"),
			m.auditLogHandler.ListHandler,
		)
	}
}
