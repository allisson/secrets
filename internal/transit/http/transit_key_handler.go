// Package http provides HTTP handlers for transit key management and cryptographic operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"

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
