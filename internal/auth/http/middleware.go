// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"log/slog"
	"strings"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

// AuthenticationMiddleware validates Bearer tokens and stores authenticated clients in request context.
//
// Extracts Bearer token from Authorization header, hashes it via tokenService.HashToken(),
// validates via tokenUseCase.Authenticate(), and stores the client for downstream handlers.
//
// Authorization header format: "Bearer <token>" (case-insensitive)
//
// Returns:
//   - 401 Unauthorized: Missing/malformed/invalid token
//   - 403 Forbidden: Inactive client
//   - 500 Internal Server Error: Other errors
func AuthenticationMiddleware(
	tokenUseCase authUseCase.TokenUseCase,
	tokenService authService.TokenService,
	logger *slog.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Debug("authentication failed: missing authorization header")
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, logger)
			c.Abort()
			return
		}

		// Parse Bearer token (case-insensitive)
		const bearerPrefix = "bearer "
		if len(authHeader) < len(bearerPrefix) ||
			!strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
			logger.Debug("authentication failed: malformed authorization header",
				slog.String("header", authHeader))
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, logger)
			c.Abort()
			return
		}

		plainToken := authHeader[len(bearerPrefix):]
		if plainToken == "" {
			logger.Debug("authentication failed: empty bearer token")
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, logger)
			c.Abort()
			return
		}

		// Hash the token for lookup
		tokenHash := tokenService.HashToken(plainToken)

		// Authenticate using the token hash
		client, err := tokenUseCase.Authenticate(c.Request.Context(), tokenHash)
		if err != nil {
			logger.Debug("authentication failed",
				slog.String("error", err.Error()))
			httputil.HandleErrorGin(c, err, logger)
			c.Abort()
			return
		}

		// Store authenticated client in context
		ctx := WithClient(c.Request.Context(), client)
		c.Request = c.Request.WithContext(ctx)

		logger.Debug("authentication successful",
			slog.String("client_id", client.ID.String()),
			slog.String("client_name", client.Name))

		// Continue to next handler
		c.Next()
	}
}

// AuthorizationMiddleware enforces capability-based authorization for authenticated clients.
//
// MUST be used after AuthenticationMiddleware. Retrieves authenticated client from context,
// extracts request path, and checks if Client.IsAllowed(path, capability) permits access.
// Creates audit logs for all authorization attempts (both successful and failed).
//
// Path Matching:
//   - Exact: "/secrets/mykey" matches policy "/secrets/mykey"
//   - Wildcard: "*" matches all paths
//   - Prefix: "secret/*" matches paths starting with "secret/"
//
// Returns:
//   - 401 Unauthorized: No authenticated client in context
//   - 403 Forbidden: Insufficient permissions
func AuthorizationMiddleware(
	capability authDomain.Capability,
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve authenticated client from context
		client, ok := GetClient(c.Request.Context())
		if !ok || client == nil {
			logger.Debug("authorization failed: no authenticated client in context")
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, logger)
			c.Abort()
			return
		}

		// Extract request path
		path := c.Request.URL.Path

		// Check if client is allowed to perform the capability on the path
		allowed := client.IsAllowed(path, capability)

		// Create audit log for the authorization attempt
		createAuditLog(c, auditLogUseCase, client, capability, path, allowed, logger)

		if !allowed {
			logger.Debug("authorization failed: insufficient permissions",
				slog.String("client_id", client.ID.String()),
				slog.String("client_name", client.Name),
				slog.String("path", path),
				slog.String("capability", string(capability)))
			httputil.HandleErrorGin(c, apperrors.ErrForbidden, logger)
			c.Abort()
			return
		}

		logger.Debug("authorization successful",
			slog.String("client_id", client.ID.String()),
			slog.String("client_name", client.Name),
			slog.String("path", path),
			slog.String("capability", string(capability)))

		// Store path and capability in context for audit logging
		ctx := WithPath(c.Request.Context(), path)
		ctx = WithCapability(ctx, capability)
		c.Request = c.Request.WithContext(ctx)

		// Continue to next handler
		c.Next()
	}
}

// createAuditLog creates an audit log entry for an authorization attempt. Extracts the
// request ID from context or generates a new UUIDv7 if missing. Includes metadata with
// authorization outcome, client IP, and user agent. Logs errors but does not block the
// request if audit log creation fails.
func createAuditLog(
	c *gin.Context,
	auditLogUseCase authUseCase.AuditLogUseCase,
	client *authDomain.Client,
	capability authDomain.Capability,
	path string,
	allowed bool,
	logger *slog.Logger,
) {
	// Extract or generate request ID
	requestIDStr := requestid.Get(c)
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		requestID = uuid.Must(uuid.NewV7())
		logger.Debug("invalid request ID, generated new one",
			slog.String("original", requestIDStr),
			slog.String("new", requestID.String()))
	}

	// Build metadata
	metadata := map[string]any{
		"allowed":    allowed,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}

	// Create audit log (non-blocking on error)
	if err := auditLogUseCase.Create(
		c.Request.Context(),
		requestID,
		client.ID,
		capability,
		path,
		metadata,
	); err != nil {
		logger.Error("failed to create audit log",
			slog.Any("error", err),
			slog.String("client_id", client.ID.String()),
			slog.String("path", path))
	}
}
