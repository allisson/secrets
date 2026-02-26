// Package http provides HTTP handlers for tokenization key management and token operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/httputil"
	"github.com/allisson/secrets/internal/tokenization/http/dto"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// TokenizationKeyHandler handles HTTP requests for tokenization key management operations.
// Coordinates key creation, rotation, and deletion with TokenizationKeyUseCase.
type TokenizationKeyHandler struct {
	keyUseCase tokenizationUseCase.TokenizationKeyUseCase
	logger     *slog.Logger
}

// NewTokenizationKeyHandler creates a new tokenization key handler with required dependencies.
func NewTokenizationKeyHandler(
	keyUseCase tokenizationUseCase.TokenizationKeyUseCase,
	logger *slog.Logger,
) *TokenizationKeyHandler {
	return &TokenizationKeyHandler{
		keyUseCase: keyUseCase,
		logger:     logger,
	}
}

// CreateHandler creates a new tokenization key with version 1.
// POST /v1/tokenization/keys - Requires WriteCapability.
// Returns 201 Created with key details.
func (h *TokenizationKeyHandler) CreateHandler(c *gin.Context) {
	var req dto.CreateTokenizationKeyRequest

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

	// Parse format type and algorithm
	formatType, err := dto.ParseFormatType(req.FormatType)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	algorithm, err := dto.ParseAlgorithm(req.Algorithm)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Call use case
	key, err := h.keyUseCase.Create(
		c.Request.Context(),
		req.Name,
		formatType,
		req.IsDeterministic,
		algorithm,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.MapTokenizationKeyToResponse(key)
	c.JSON(http.StatusCreated, response)
}

// RotateHandler creates a new version of an existing tokenization key.
// POST /v1/tokenization/keys/:name/rotate - Requires WriteCapability.
// Returns 201 Created with new key version.
func (h *TokenizationKeyHandler) RotateHandler(c *gin.Context) {
	var req dto.RotateTokenizationKeyRequest

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

	// Get key name from URL parameter
	keyName := c.Param("name")
	if keyName == "" {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("key name is required in URL path"),
			h.logger)
		return
	}

	// Parse format type and algorithm
	formatType, err := dto.ParseFormatType(req.FormatType)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	algorithm, err := dto.ParseAlgorithm(req.Algorithm)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Call use case
	key, err := h.keyUseCase.Rotate(
		c.Request.Context(),
		keyName,
		formatType,
		req.IsDeterministic,
		algorithm,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.MapTokenizationKeyToResponse(key)
	c.JSON(http.StatusCreated, response)
}

// DeleteHandler soft-deletes a tokenization key by ID.
// DELETE /v1/tokenization/keys/:id - Requires DeleteCapability.
// Returns 204 No Content on success.
func (h *TokenizationKeyHandler) DeleteHandler(c *gin.Context) {
	// Parse and validate UUID
	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid key ID format: must be a valid UUID"),
			h.logger)
		return
	}

	// Call use case
	if err := h.keyUseCase.Delete(c.Request.Context(), keyID); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content
	c.Data(http.StatusNoContent, "application/json", nil)
}

// ListHandler retrieves tokenization keys with pagination support.
// GET /v1/tokenization/keys?offset=0&limit=50 - Requires ReadCapability.
// Returns 200 OK with paginated tokenization key list.
func (h *TokenizationKeyHandler) ListHandler(c *gin.Context) {
	// Parse offset and limit query parameters
	offset, limit, err := httputil.ParsePagination(c)
	if err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
		return
	}

	// Call use case
	keys, err := h.keyUseCase.List(c.Request.Context(), offset, limit)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Map to response
	response := dto.MapTokenizationKeysToListResponse(keys)
	c.JSON(http.StatusOK, response)
}
