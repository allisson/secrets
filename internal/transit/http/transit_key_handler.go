// Package http provides HTTP handlers for transit key management and cryptographic operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/allisson/secrets/internal/httputil"
	"github.com/allisson/secrets/internal/transit/http/dto"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// TransitKeyHandler handles HTTP requests for transit key management operations.
// It coordinates authentication, authorization, and audit logging with the TransitKeyUseCase.
type TransitKeyHandler struct {
	transitKeyUseCase transitUseCase.TransitKeyUseCase // Business logic for transit key lifecycle operations
	logger            *slog.Logger                     // Structured logger for request handling and error reporting
}

// NewTransitKeyHandler creates a new transit key handler with required dependencies.
func NewTransitKeyHandler(
	transitKeyUseCase transitUseCase.TransitKeyUseCase,
	logger *slog.Logger,
) *TransitKeyHandler {
	return &TransitKeyHandler{
		transitKeyUseCase: transitKeyUseCase,
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
		httputil.HandleBadRequestGin(c, err, h.logger)
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
		httputil.HandleBadRequestGin(c, err, h.logger)
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
// Returns 201 Created with new version metadata.
func (h *TransitKeyHandler) RotateHandler(c *gin.Context) {
	// Extract and validate name from URL parameter
	name := c.Param("name")
	if name == "" {
		httputil.HandleBadRequestGin(
			c,
			fmt.Errorf("transit key name cannot be empty"),
			h.logger,
		)
		return
	}

	var req dto.RotateTransitKeyRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
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
		httputil.HandleBadRequestGin(c, err, h.logger)
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
	c.JSON(http.StatusCreated, response)
}

// DeleteHandler soft deletes a transit key by name.
// DELETE /v1/transit/keys/:name - Requires DeleteCapability on path /v1/transit/keys/:name.
// Returns 204 No Content.
func (h *TransitKeyHandler) DeleteHandler(c *gin.Context) {
	// Extract and validate name from URL parameter
	name := c.Param("name")
	if name == "" {
		httputil.HandleBadRequestGin(
			c,
			fmt.Errorf("transit key name cannot be empty"),
			h.logger,
		)
		return
	}

	// Call use case
	if err := h.transitKeyUseCase.Delete(c.Request.Context(), name); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content with empty body
	c.Data(http.StatusNoContent, "application/json", nil)
}

// ListHandler retrieves transit keys with cursor-based pagination support.
// GET /v1/transit/keys?after_name=key-name&limit=50 - Requires ReadCapability.
// Returns 200 OK with paginated transit key list ordered by name ascending.
// Uses cursor pagination with after_name parameter.
func (h *TransitKeyHandler) ListHandler(c *gin.Context) {
	// Parse cursor and limit query parameters
	afterName, limit, err := httputil.ParseStringCursorPagination(c, "after_name")
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Call use case with limit + 1 to detect if there are more results
	transitKeys, err := h.transitKeyUseCase.ListCursor(c.Request.Context(), afterName, limit+1)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Determine if there are more results and set next cursor
	var nextCursor *string
	if len(transitKeys) > limit {
		// More results exist, use the last visible item's name as cursor
		transitKeys = transitKeys[:limit]
		cursorValue := transitKeys[len(transitKeys)-1].Name
		nextCursor = &cursorValue
	}

	// Map to response
	response := dto.MapTransitKeysToListResponse(transitKeys, nextCursor)
	c.JSON(http.StatusOK, response)
}

// GetHandler retrieves transit key metadata by name and optional version.
// GET /v1/transit/keys/:name?version=1 - Requires ReadCapability.
// Returns 200 OK with transit key metadata and algorithm.
func (h *TransitKeyHandler) GetHandler(c *gin.Context) {
	// Extract and validate name from URL parameter
	name := c.Param("name")
	if name == "" {
		httputil.HandleBadRequestGin(
			c,
			fmt.Errorf("transit key name cannot be empty"),
			h.logger,
		)
		return
	}

	// Extract and validate optional version from query parameter
	version := uint(0)
	versionStr := c.Query("version")
	if versionStr != "" {
		v, err := strconv.ParseUint(versionStr, 10, 32)
		if err != nil {
			httputil.HandleBadRequestGin(
				c,
				fmt.Errorf("invalid version format: must be a positive integer"),
				h.logger,
			)
			return
		}
		version = uint(v)
	}

	// Call use case
	transitKey, alg, err := h.transitKeyUseCase.Get(c.Request.Context(), name, version)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Map to response
	response := dto.MapTransitKeyToMetadataResponse(transitKey, string(alg))
	c.JSON(http.StatusOK, response)
}
