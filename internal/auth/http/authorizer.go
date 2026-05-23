// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"log/slog"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

// Authorizer enforces capability-based authorization and emits audit logs
// for every authorization attempt. Construct once at server bootstrap; call
// Require(cap) per route.
type Authorizer struct {
	audit  authUseCase.AuditLogUseCase
	logger *slog.Logger
}

// NewAuthorizer binds the audit log use case and logger so per-route middleware
// only needs the capability.
func NewAuthorizer(audit authUseCase.AuditLogUseCase, logger *slog.Logger) *Authorizer {
	return &Authorizer{audit: audit, logger: logger}
}

// Require returns a Gin middleware that admits requests authorised for
// capability cap on the request path. MUST follow AuthenticationMiddleware.
//
// Path Matching:
//   - Exact: "/secrets/mykey" matches policy "/secrets/mykey"
//   - Wildcard: "*" matches all paths
//   - Prefix: "secret/*" matches paths starting with "secret/"
//
// Returns:
//   - 401 Unauthorized: No authenticated client in context
//   - 403 Forbidden: Insufficient permissions
func (a *Authorizer) Require(cap authDomain.Capability) gin.HandlerFunc {
	return func(c *gin.Context) {
		client, ok := GetClient(c.Request.Context())
		if !ok || client == nil {
			a.logger.Debug("authorization failed: no authenticated client in context")
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, a.logger)
			c.Abort()
			return
		}

		path := c.Request.URL.Path
		allowed := client.IsAllowed(path, cap)

		a.createAuditLog(c, client, cap, path, allowed)

		if !allowed {
			a.logger.Debug("authorization failed: insufficient permissions",
				slog.String("client_id", client.ID.String()),
				slog.String("client_name", client.Name),
				slog.String("path", path),
				slog.String("capability", string(cap)))
			httputil.HandleErrorGin(c, apperrors.ErrForbidden, a.logger)
			c.Abort()
			return
		}

		a.logger.Debug("authorization successful",
			slog.String("client_id", client.ID.String()),
			slog.String("client_name", client.Name),
			slog.String("path", path),
			slog.String("capability", string(cap)))

		ctx := WithPath(c.Request.Context(), path)
		ctx = WithCapability(ctx, cap)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// createAuditLog records an authorization attempt. Extracts the request ID
// from context (or generates a new UUIDv7), attaches client IP and user
// agent, and logs but does not block the request on audit failure.
func (a *Authorizer) createAuditLog(
	c *gin.Context,
	client *authDomain.Client,
	cap authDomain.Capability,
	path string,
	allowed bool,
) {
	requestIDStr := requestid.Get(c)
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		requestID = uuid.Must(uuid.NewV7())
		a.logger.Debug("invalid request ID, generated new one",
			slog.String("original", requestIDStr),
			slog.String("new", requestID.String()))
	}

	metadata := map[string]any{
		"allowed":    allowed,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}

	if err := a.audit.Create(
		c.Request.Context(),
		requestID,
		client.ID,
		cap,
		path,
		metadata,
	); err != nil {
		a.logger.Error("failed to create audit log",
			slog.Any("error", err),
			slog.String("client_id", client.ID.String()),
			slog.String("path", path))
	}
}
