// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authService "github.com/allisson/secrets/internal/auth/service"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

// AuthenticationMiddleware provides authentication via Bearer token in the Authorization header.
//
// The middleware:
// 1. Extracts the Bearer token from the Authorization header (case-insensitive)
// 2. Hashes the token using tokenService.HashToken()
// 3. Validates the token using tokenUseCase.Authenticate()
// 4. Stores the authenticated client in the request context
// 5. Allows downstream handlers to access the client via GetClient()
//
// Authorization header format: "Bearer <token>" (case-insensitive "bearer")
//
// Error handling:
//   - Missing Authorization header → 401 Unauthorized
//   - Malformed Authorization header → 401 Unauthorized
//   - Invalid/expired/revoked token → 401 Unauthorized (from TokenUseCase.Authenticate)
//   - Inactive client → 403 Forbidden (from TokenUseCase.Authenticate)
//   - Other errors → 500 Internal Server Error
//
// Usage:
//
//	router.Use(AuthenticationMiddleware(tokenUseCase, tokenService, logger))
//	router.GET("/protected", func(c *gin.Context) {
//	    client, ok := GetClient(c.Request.Context())
//	    if !ok {
//	        // Should never happen if middleware is working correctly
//	        c.JSON(401, gin.H{"error": "unauthorized"})
//	        return
//	    }
//	    // Use client for authorization checks
//	})
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

// AuthorizationMiddleware provides capability-based authorization for authenticated clients.
//
// This middleware MUST be used after AuthenticationMiddleware, as it requires an authenticated
// client to be present in the request context. It checks if the client's policies permit the
// requested capability on the current request path.
//
// The middleware:
// 1. Retrieves the authenticated client from the request context via GetClient()
// 2. Extracts the request path from c.Request.URL.Path
// 3. Checks if the client is allowed to perform the specified capability on the path
// 4. Uses Client.IsAllowed(path, capability) for policy-based authorization
// 5. Returns 403 Forbidden if the client lacks the required permission
// 6. Returns 401 Unauthorized if no authenticated client is found in context
//
// Path Matching:
// The authorization check uses the Client.IsAllowed() method which supports:
//   - Exact path matching: "/secrets/mykey" matches policy "/secrets/mykey"
//   - Wildcard matching: "*" matches all paths (admin mode)
//   - Prefix matching: "secret/*" matches any path starting with "secret/"
//
// Error handling:
//   - No client in context → 401 Unauthorized (AuthenticationMiddleware not run)
//   - Client lacks capability → 403 Forbidden
//   - Path doesn't match any policy → 403 Forbidden
//
// Usage:
//
//	// Require read capability for GET /api/v1/secrets/:id
//	router.GET("/api/v1/secrets/:id",
//	    AuthenticationMiddleware(tokenUseCase, tokenService, logger),
//	    AuthorizationMiddleware(authDomain.ReadCapability, logger),
//	    handler)
//
//	// Require write capability for POST /api/v1/secrets
//	router.POST("/api/v1/secrets",
//	    AuthenticationMiddleware(tokenUseCase, tokenService, logger),
//	    AuthorizationMiddleware(authDomain.WriteCapability, logger),
//	    handler)
func AuthorizationMiddleware(
	capability authDomain.Capability,
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
		if !client.IsAllowed(path, capability) {
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

		// Continue to next handler
		c.Next()
	}
}
