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
		httputil.HandleBadRequestGin(c, err, h.logger)
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
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	algorithm, err := dto.ParseAlgorithm(req.Algorithm)
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
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
// POST /v1/tokenization/keys/:name/rotate - Requires RotateCapability.
// Returns 201 Created with new key version.
func (h *TokenizationKeyHandler) RotateHandler(c *gin.Context) {
	var req dto.RotateTokenizationKeyRequest

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

	// Get key name from URL parameter
	keyName := c.Param("name")
	if keyName == "" {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("key name is required in URL path"),
			h.logger)
		return
	}

	// Parse format type and algorithm
	formatType, err := dto.ParseFormatType(req.FormatType)
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	algorithm, err := dto.ParseAlgorithm(req.Algorithm)
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
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
		httputil.HandleBadRequestGin(c,
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

// GetByNameHandler retrieves a single tokenization key by its name.
// GET /v1/tokenization/keys/:name - Requires ReadCapability.
// Returns 200 OK with key details.
func (h *TokenizationKeyHandler) GetByNameHandler(c *gin.Context) {
	// Get key name from URL parameter
	keyName := c.Param("name")
	if keyName == "" {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("key name is required in URL path"),
			h.logger)
		return
	}

	// Call use case
	key, err := h.keyUseCase.GetByName(c.Request.Context(), keyName)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Map to response
	response := dto.MapTokenizationKeyToResponse(key)
	c.JSON(http.StatusOK, response)
}

// ListHandler retrieves tokenization keys with cursor-based pagination support.
// GET /v1/tokenization/keys?after_name=key-name&limit=50 - Requires ReadCapability.
// Returns 200 OK with paginated tokenization key list ordered by name ascending.
// Uses cursor pagination with after_name parameter.
func (h *TokenizationKeyHandler) ListHandler(c *gin.Context) {
	// Parse cursor and limit query parameters
	afterName, limit, err := httputil.ParseStringCursorPagination(c, "after_name")
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Call use case with limit + 1 to detect if there are more results
	keys, err := h.keyUseCase.ListCursor(c.Request.Context(), afterName, limit+1)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Determine if there are more results and set next cursor
	var nextCursor *string
	if len(keys) > limit {
		// More results exist, use the last visible item's name as cursor
		keys = keys[:limit]
		cursorValue := keys[len(keys)-1].Name
		nextCursor = &cursorValue
	}

	// Map to response
	response := dto.MapTokenizationKeysToListResponse(keys, nextCursor)
	c.JSON(http.StatusOK, response)
}
