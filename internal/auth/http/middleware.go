// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
)

// AuthenticationMiddleware validates Bearer tokens and stores authenticated clients in request context.
//
// Extracts Bearer token from Authorization header, validates via tokenUseCase.Authenticate(),
// and stores the client for downstream handlers. Hash derivation is internal to the use case.
//
// Authorization header format: "Bearer <token>" (case-insensitive)
//
// Returns:
//   - 401 Unauthorized: Missing/malformed/invalid token
//   - 403 Forbidden: Inactive client
//   - 500 Internal Server Error: Other errors
func AuthenticationMiddleware(
	tokenUseCase authUseCase.TokenUseCase,
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

		client, err := tokenUseCase.Authenticate(c.Request.Context(), plainToken)
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
