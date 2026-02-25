// Package http provides HTTP handlers for transit key management and cryptographic operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/httputil"
	"github.com/allisson/secrets/internal/transit/http/dto"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// TransitKeyHandler handles HTTP requests for transit key management operations.
// It coordinates authentication, authorization, and audit logging with the TransitKeyUseCase.
type TransitKeyHandler struct {
	transitKeyUseCase transitUseCase.TransitKeyUseCase
	auditLogUseCase   authUseCase.AuditLogUseCase
	logger            *slog.Logger
}

// NewTransitKeyHandler creates a new transit key handler with required dependencies.
func NewTransitKeyHandler(
	transitKeyUseCase transitUseCase.TransitKeyUseCase,
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
) *TransitKeyHandler {
	return &TransitKeyHandler{
		transitKeyUseCase: transitKeyUseCase,
		auditLogUseCase:   auditLogUseCase,
		logger:            logger,
	}
}

// CreateHandler creates a new transit key with version 1.
// POST /v1/transit/keys - Requires WriteCapability on path /v1/transit/keys.
// Returns 201 Created with transit key metadata.
func (h *TransitKeyHandler) CreateHandler(c *gin.Context) {
	var req dto.CreateTransitKeyRequest

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

	// Parse algorithm
	alg, err := dto.ParseAlgorithm(req.Algorithm)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Call use case
	transitKey, err := h.transitKeyUseCase.Create(c.Request.Context(), req.Name, alg)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.MapTransitKeyToResponse(transitKey)
	c.JSON(http.StatusCreated, response)
}

// RotateHandler creates a new version of an existing transit key.
// POST /v1/transit/keys/:name/rotate - Requires RotateCapability.
// Returns 200 OK with new version metadata.
func (h *TransitKeyHandler) RotateHandler(c *gin.Context) {
	// Extract and validate name from URL parameter
	name := c.Param("name")
	if name == "" {
		httputil.HandleValidationErrorGin(
			c,
			fmt.Errorf("transit key name cannot be empty"),
			h.logger,
		)
		return
	}

	var req dto.RotateTransitKeyRequest

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

	// Parse algorithm
	alg, err := dto.ParseAlgorithm(req.Algorithm)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Call use case
	transitKey, err := h.transitKeyUseCase.Rotate(c.Request.Context(), name, alg)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.MapTransitKeyToResponse(transitKey)
	c.JSON(http.StatusOK, response)
}

// DeleteHandler soft deletes a transit key by ID.
// DELETE /v1/transit/keys/:id - Requires DeleteCapability on path /v1/transit/keys/:id.
// Returns 204 No Content.
func (h *TransitKeyHandler) DeleteHandler(c *gin.Context) {
	// Parse and validate UUID
	transitKeyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid transit key ID format: must be a valid UUID"),
			h.logger)
		return
	}

	// Call use case
	if err := h.transitKeyUseCase.Delete(c.Request.Context(), transitKeyID); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content with empty body
	c.Data(http.StatusNoContent, "application/json", nil)
}

// ListHandler retrieves transit keys with pagination support.
// GET /v1/transit/keys?offset=0&limit=50 - Requires ReadCapability.
// Returns 200 OK with paginated transit key list.
func (h *TransitKeyHandler) ListHandler(c *gin.Context) {
	// Parse offset query parameter (default: 0)
	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid offset parameter: must be a non-negative integer"),
			h.logger)
		return
	}

	// Parse limit query parameter (default: 50, max: 100)
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid limit parameter: must be between 1 and 100"),
			h.logger)
		return
	}

	// Call use case
	transitKeys, err := h.transitKeyUseCase.List(c.Request.Context(), offset, limit)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Map to response
	response := dto.MapTransitKeysToListResponse(transitKeys)
	c.JSON(http.StatusOK, response)
}
