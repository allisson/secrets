// Package http provides HTTP handlers for secret management operations.
// Secrets are encrypted at rest using envelope encryption and can be versioned.
package http

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/httputil"
	secretsDomain "github.com/allisson/secrets/internal/secrets/domain"
	"github.com/allisson/secrets/internal/secrets/http/dto"
	secretsUseCase "github.com/allisson/secrets/internal/secrets/usecase"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// SecretHandler handles HTTP requests for secret management operations.
// It coordinates authentication, authorization, and audit logging with the SecretUseCase.
type SecretHandler struct {
	secretUseCase   secretsUseCase.SecretUseCase
	auditLogUseCase authUseCase.AuditLogUseCase
	logger          *slog.Logger
}

// NewSecretHandler creates a new secret handler with required dependencies.
func NewSecretHandler(
	secretUseCase secretsUseCase.SecretUseCase,
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
) *SecretHandler {
	return &SecretHandler{
		secretUseCase:   secretUseCase,
		auditLogUseCase: auditLogUseCase,
		logger:          logger,
	}
}

// CreateOrUpdateHandler creates a new secret or updates an existing one.
// POST /v1/secrets/*path - Requires EncryptCapability.
// Returns 201 Created with secret metadata (excludes plaintext value for security).
func (h *SecretHandler) CreateOrUpdateHandler(c *gin.Context) {
	// Extract and validate path from URL parameter
	path := strings.TrimPrefix(c.Param("path"), "/")
	if path == "" {
		httputil.HandleValidationErrorGin(
			c,
			fmt.Errorf("path cannot be empty"),
			h.logger,
		)
		return
	}

	var req dto.CreateOrUpdateSecretRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		httputil.HandleValidationErrorGin(c, customValidation.WrapValidationError(err), h.logger)
		return
	}

	// Decode base64 value
	value, err := base64.StdEncoding.DecodeString(req.Value)
	if err != nil {
		httputil.HandleValidationErrorGin(
			c,
			fmt.Errorf("invalid base64 value: %w", err),
			h.logger,
		)
		return
	}

	// Call use case with decoded bytes
	secret, err := h.secretUseCase.CreateOrUpdate(c.Request.Context(), path, value)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response with metadata only (no plaintext)
	response := dto.MapSecretToCreateResponse(secret)
	c.JSON(http.StatusCreated, response)
}

// GetHandler retrieves and decrypts a secret by path, optionally by version.
// GET /v1/secrets/*path?version=N - Requires DecryptCapability.
// Returns 200 OK with plaintext value. SECURITY: Plaintext is zeroed after response.
func (h *SecretHandler) GetHandler(c *gin.Context) {
	// Extract and validate path from URL parameter
	path := strings.TrimPrefix(c.Param("path"), "/")
	if path == "" {
		httputil.HandleValidationErrorGin(
			c,
			fmt.Errorf("path cannot be empty"),
			h.logger,
		)
		return
	}

	var secret *secretsDomain.Secret
	var err error

	// Check for version query parameter
	versionStr := c.Query("version")
	if versionStr != "" {
		version, parseErr := strconv.ParseUint(versionStr, 10, 32)
		if parseErr != nil {
			httputil.HandleValidationErrorGin(
				c,
				fmt.Errorf("invalid version parameter: must be a positive integer"),
				h.logger,
			)
			return
		}
		secret, err = h.secretUseCase.GetByVersion(c.Request.Context(), path, uint(version))
	} else {
		secret, err = h.secretUseCase.Get(c.Request.Context(), path)
	}

	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// SECURITY: Zero plaintext after mapping to response
	defer cryptoDomain.Zero(secret.Plaintext)

	// Map to response (includes plaintext value)
	response := dto.MapSecretToGetResponse(secret)
	c.JSON(http.StatusOK, response)
}

// DeleteHandler soft deletes a secret by its path.
// DELETE /v1/secrets/*path - Requires DeleteCapability.
// Returns 204 No Content.
func (h *SecretHandler) DeleteHandler(c *gin.Context) {
	// Extract and validate path from URL parameter
	path := strings.TrimPrefix(c.Param("path"), "/")
	if path == "" {
		httputil.HandleValidationErrorGin(
			c,
			fmt.Errorf("path cannot be empty"),
			h.logger,
		)
		return
	}

	// Call use case
	if err := h.secretUseCase.Delete(c.Request.Context(), path); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content with empty body
	c.Data(http.StatusNoContent, "application/json", nil)
}

// ListHandler retrieves secrets with pagination support.
// GET /v1/secrets?offset=0&limit=50 - Requires ReadCapability.
// Returns 200 OK with paginated secret list (excludes plaintext value for security).
func (h *SecretHandler) ListHandler(c *gin.Context) {
	// Parse offset and limit query parameters
	offset, limit, err := httputil.ParsePagination(c)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Call use case
	secrets, err := h.secretUseCase.List(c.Request.Context(), offset, limit)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Map to response
	response := dto.MapSecretsToListResponse(secrets)
	c.JSON(http.StatusOK, response)
}
